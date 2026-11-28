package learning

import (
	"sort"
	"time"

	"github.com/google/uuid"

	"oge-ege/internal/content"
	"oge-ege/internal/shared/render/ui"
)

const recencyThresholdDays = 14

// TopicPriority описывает приоритет повторения темы.
type TopicPriority struct {
	TopicID    uuid.UUID
	Title      string
	OrderIndex int
	Weight     float64
	Mastery    float64
	Attempts   int
	Priority   float64
}

// ComputeTopicWeights рассчитывает вес темы в первичном балле.
func ComputeTopicWeights(topics []content.Topic, kims []content.KimNumber, links []content.KimTopic) map[uuid.UUID]float64 {
	kimByID := make(map[uuid.UUID]content.KimNumber, len(kims))
	for _, kim := range kims {
		kimByID[kim.ID] = kim
	}

	numerator := make(map[uuid.UUID]float64, len(topics))
	denominator := make(map[uuid.UUID]float64, len(topics))
	for _, link := range links {
		kim, ok := kimByID[link.KimNumberID]
		if !ok {
			continue
		}
		numerator[link.TopicID] += float64(kim.MaxPrimaryScore) * link.Weight
		denominator[link.TopicID] += float64(kim.MaxPrimaryScore)
	}

	weights := make(map[uuid.UUID]float64, len(topics))
	for _, topic := range topics {
		if denominator[topic.ID] <= 0 {
			continue
		}
		weights[topic.ID] = numerator[topic.ID] / denominator[topic.ID]
	}

	return weights
}

// BuildPriorities рассчитывает приоритеты повторения тем.
func BuildPriorities(topics []content.Topic, weights map[uuid.UUID]float64, mastery map[uuid.UUID]TopicMastery, now time.Time) []TopicPriority {
	priorities := make([]TopicPriority, 0, len(topics))
	for _, topic := range topics {
		record := mastery[topic.ID]
		confidence := 1.0
		if LowConfidence(record.Attempts) {
			confidence = 1.2
		}

		recency := 1.0
		if record.LastPracticedAt == nil || now.Sub(*record.LastPracticedAt) > recencyThresholdDays*24*time.Hour {
			recency = 1.0 + 0.2*(1-record.Mastery)
		}

		weight := weights[topic.ID]
		priorities = append(priorities, TopicPriority{
			TopicID:    topic.ID,
			Title:      topic.Title,
			OrderIndex: topic.OrderIndex,
			Weight:     weight,
			Mastery:    record.Mastery,
			Attempts:   record.Attempts,
			Priority:   weight * (1 - record.Mastery) * confidence * recency,
		})
	}

	sort.SliceStable(priorities, func(i, j int) bool {
		left := priorities[i]
		right := priorities[j]
		if (left.Attempts == 0) != (right.Attempts == 0) {
			return left.Attempts == 0
		}
		if left.Priority != right.Priority {
			return left.Priority > right.Priority
		}

		return left.OrderIndex < right.OrderIndex
	})

	return priorities
}

// BuildGapMap собирает карту «тема × номер КИМ».
func BuildGapMap(
	topics []content.Topic,
	kims []content.KimNumber,
	links []content.KimTopic,
	topicMastery map[uuid.UUID]TopicMastery,
	kimMastery map[uuid.UUID]KimMastery,
	now time.Time,
) GapMapView {
	orderedTopics := make([]content.Topic, len(topics))
	copy(orderedTopics, topics)
	sort.Slice(orderedTopics, func(i, j int) bool {
		if orderedTopics[i].OrderIndex != orderedTopics[j].OrderIndex {
			return orderedTopics[i].OrderIndex < orderedTopics[j].OrderIndex
		}

		return orderedTopics[i].ID.String() < orderedTopics[j].ID.String()
	})

	orderedKims := make([]content.KimNumber, len(kims))
	copy(orderedKims, kims)
	sort.Slice(orderedKims, func(i, j int) bool { return orderedKims[i].Number < orderedKims[j].Number })

	linked := make(map[[2]uuid.UUID]struct{}, len(links))
	for _, link := range links {
		linked[[2]uuid.UUID{link.TopicID, link.KimNumberID}] = struct{}{}
	}

	view := GapMapView{
		Topics:     make([]string, 0, len(orderedTopics)),
		KimNumbers: make([]int, 0, len(orderedKims)),
		Cells:      make([][]ui.GapCell, 0, len(orderedTopics)),
	}
	for _, topic := range orderedTopics {
		view.Topics = append(view.Topics, topic.Title)
	}
	for _, kim := range orderedKims {
		view.KimNumbers = append(view.KimNumbers, kim.Number)
	}

	hasData := false
	for _, topic := range orderedTopics {
		record := topicMastery[topic.ID]
		if record.Attempts > 0 {
			hasData = true
		}
		row := make([]ui.GapCell, 0, len(orderedKims))
		for _, kim := range orderedKims {
			if _, ok := linked[[2]uuid.UUID{topic.ID, kim.ID}]; !ok {
				row = append(row, ui.GapCell{State: ui.GapNone, Label: "нет связи"})
				continue
			}
			kimRecord := kimMastery[kim.ID]
			if kimRecord.Attempts > 0 {
				hasData = true
			}
			row = append(row, gapCell(record.Mastery, kimRecord.Mastery))
		}
		view.Cells = append(view.Cells, row)
	}
	view.HasData = hasData

	weights := ComputeTopicWeights(orderedTopics, orderedKims, links)
	priorities := BuildPriorities(orderedTopics, weights, topicMastery, now)
	view.Recommendations = buildRecommendations(priorities, orderedTopics, links, topicMastery, kimMastery)

	return view
}

func gapCell(topicMastery, kimMastery float64) ui.GapCell {
	combined := (topicMastery + kimMastery) / 2
	switch {
	case combined >= 0.7:
		return ui.GapCell{State: ui.GapCorrect, Label: "освоено"}
	case combined < 0.4:
		return ui.GapCell{State: ui.GapWrong, Label: "пробел"}
	default:
		return ui.GapCell{State: ui.GapNone, Label: "частично"}
	}
}

func buildRecommendations(
	priorities []TopicPriority,
	topics []content.Topic,
	links []content.KimTopic,
	topicMastery map[uuid.UUID]TopicMastery,
	kimMastery map[uuid.UUID]KimMastery,
) []ui.GapSummary {
	linkedKims := make(map[uuid.UUID][]uuid.UUID, len(topics))
	for _, link := range links {
		linkedKims[link.TopicID] = append(linkedKims[link.TopicID], link.KimNumberID)
	}

	limit := len(priorities)
	if limit > 3 {
		limit = 3
	}
	summaries := make([]ui.GapSummary, 0, limit)
	for _, priority := range priorities[:limit] {
		count := 0
		for _, kimID := range linkedKims[priority.TopicID] {
			if kimMastery[kimID].Mastery < 0.7 {
				count++
			}
		}
		if count == 0 && topicMastery[priority.TopicID].Mastery < 0.7 {
			count = 1
		}
		summaries = append(summaries, ui.GapSummary{Title: priority.Title, Count: count, Tone: ui.ToneWarn})
	}

	return summaries
}
