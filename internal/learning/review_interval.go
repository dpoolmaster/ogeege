package learning

import (
	"encoding/json"
	"math"
	"time"

	"github.com/google/uuid"
)

const (
	maxIntervalDays = 60
	maxEase         = 2.5
	minEase         = 1.3
)

// NextReview рассчитывает интервал, легкость и дату следующего повторения темы.
func NextReview(intervalDays int, ease float64, success bool, now time.Time) (int, float64, time.Time) {
	if intervalDays <= 0 {
		intervalDays = defaultIntervalDays
	}
	if ease <= 0 {
		ease = defaultEase
	}

	if success {
		intervalDays = int(math.Round(float64(intervalDays) * ease))
		if intervalDays < 1 {
			intervalDays = 1
		}
		if intervalDays > maxIntervalDays {
			intervalDays = maxIntervalDays
		}
		ease = math.Min(ease+0.1, maxEase)
	} else {
		intervalDays = defaultIntervalDays
		ease = math.Max(ease-0.2, minEase)
	}

	return intervalDays, ease, now.AddDate(0, 0, intervalDays)
}

func applyIntervals(
	userID uuid.UUID,
	topicMastery map[uuid.UUID]TopicMastery,
	items []SessionItem,
	answers map[uuid.UUID]Answer,
	now time.Time,
) {
	answered := make(map[uuid.UUID]bool)
	wrong := make(map[uuid.UUID]bool)
	for _, item := range items {
		answer, ok := answers[item.ID]
		if !ok {
			continue
		}
		var snapshot TaskSnapshot
		if err := json.Unmarshal(item.TaskSnapshot, &snapshot); err != nil || snapshot.TopicID == nil {
			continue
		}
		topicID := *snapshot.TopicID
		answered[topicID] = true
		if !answer.IsCorrect {
			wrong[topicID] = true
		}
	}

	for topicID := range answered {
		record := topicMastery[topicID]
		record.UserID = userID
		record.TopicID = topicID
		intervalDays, ease, nextReviewAt := NextReview(record.IntervalDays, record.Ease, !wrong[topicID], now)
		record.IntervalDays = intervalDays
		record.Ease = ease
		record.NextReviewAt = &nextReviewAt
		record.LastPracticedAt = &now
		topicMastery[topicID] = record
	}
}
