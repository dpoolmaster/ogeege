package learning

import (
	"time"

	"github.com/google/uuid"
)

// ItemBlueprint описывает пункт плана до сохранения.
type ItemBlueprint struct {
	DayOfWeek   int
	Kind        ItemKind
	TopicID     *uuid.UUID
	KimNumberID *uuid.UUID
	TaskCount   int
}

// WeekBlueprint описывает неделю плана до сохранения.
type WeekBlueprint struct {
	WeekStart time.Time
	Phase     Phase
	Items     []ItemBlueprint
}

// BuildWeekBlueprints строит недели от текущей недели до недели экзамена.
func BuildWeekBlueprints(currentWeek, examDate time.Time, weeklyMinutes int, priorities []TopicPriority, diagnosticDone bool) []WeekBlueprint {
	currentWeek = WeekStart(currentWeek)
	examWeek := WeekStart(examDate)
	if examWeek.Before(currentWeek) {
		return nil
	}

	totalWeeks := WeeksBetween(currentWeek, examWeek) + 1
	blueprints := make([]WeekBlueprint, 0, totalWeeks)
	topicCursor := 0
	minutes := distributeMinutes(weeklyMinutes)

	for index := 0; index < totalWeeks; index++ {
		phase := phaseFor(index, totalWeeks)
		weekStart := currentWeek.AddDate(0, 0, 7*index)
		items := make([]ItemBlueprint, 0, len(minutes))
		for day, value := range minutes {
			if value <= 0 {
				continue
			}

			kind := kindFor(phase)
			if index == 0 && day == 0 && !diagnosticDone {
				kind = KindDiagnostic
			}

			var topicID *uuid.UUID
			if kind != KindDiagnostic && len(priorities) > 0 {
				topic := priorities[topicCursor%len(priorities)]
				id := topic.TopicID
				topicID = &id
				topicCursor++
			}

			taskCount := value / 5
			if taskCount < 1 {
				taskCount = 1
			}

			items = append(items, ItemBlueprint{
				DayOfWeek: day + 1,
				Kind:      kind,
				TopicID:   topicID,
				TaskCount: taskCount,
			})
		}

		blueprints = append(blueprints, WeekBlueprint{WeekStart: weekStart, Phase: phase, Items: items})
	}

	return blueprints
}

func phaseFor(index, total int) Phase {
	if total <= 1 || index == 0 {
		return PhaseDiagnostics
	}
	if index == total-1 {
		return PhaseReview
	}

	middle := total - 2
	middleIndex := index - 1
	baseCount := (middle + 1) / 2
	if middleIndex < baseCount {
		return PhaseBase
	}

	return PhasePractice
}

func kindFor(phase Phase) ItemKind {
	switch phase {
	case PhasePractice:
		return KindPractice
	case PhaseReview:
		return KindReview
	default:
		return KindTheory
	}
}

func distributeMinutes(weeklyMinutes int) []int {
	if weeklyMinutes <= 0 {
		return make([]int, 7)
	}

	base := weeklyMinutes / 7
	remainder := weeklyMinutes % 7
	minutes := make([]int, 7)
	for day := 0; day < 7; day++ {
		minutes[day] = base
		if day < remainder {
			minutes[day]++
		}
	}

	return minutes
}
