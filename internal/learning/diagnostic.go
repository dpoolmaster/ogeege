package learning

import (
	"sort"

	"github.com/google/uuid"

	"oge-ege/internal/content"
)

// EstimateMinutes оценивает длительность задания по сложности.
func EstimateMinutes(difficulty content.Difficulty) int {
	switch difficulty {
	case content.DifficultyEasy:
		return 1
	case content.DifficultyHard:
		return 3
	default:
		return 2
	}
}

func difficultyRank(difficulty content.Difficulty) int {
	switch difficulty {
	case content.DifficultyEasy:
		return 0
	case content.DifficultyMedium:
		return 1
	case content.DifficultyHard:
		return 2
	default:
		return 3
	}
}

// BuildDiagnosticSample формирует выборку по одному заданию на каждый номер КИМ.
// Задания упорядочены по возрастанию номера, длительность возвращается как ориентир.
func BuildDiagnosticSample(numbers []content.KimNumber, tasks []content.Task) ([]content.Task, int, error) {
	grouped := make(map[uuid.UUID][]content.Task, len(numbers))
	for _, task := range tasks {
		grouped[task.KimNumberID] = append(grouped[task.KimNumberID], task)
	}

	ordered := make([]content.KimNumber, len(numbers))
	copy(ordered, numbers)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Number < ordered[j].Number })

	selected := make([]content.Task, 0, len(ordered))
	for _, number := range ordered {
		candidates := grouped[number.ID]
		if len(candidates) == 0 {
			return nil, 0, ErrInsufficientContent
		}
		selected = append(selected, bestCandidate(candidates))
	}

	duration := 0
	for _, task := range selected {
		duration += EstimateMinutes(task.Difficulty)
	}

	return selected, duration, nil
}

func bestCandidate(candidates []content.Task) content.Task {
	best := candidates[0]
	for _, candidate := range candidates[1:] {
		if preferCandidate(candidate, best) {
			best = candidate
		}
	}

	return best
}

func preferCandidate(candidate, current content.Task) bool {
	candidateRank := difficultyRank(candidate.Difficulty)
	currentRank := difficultyRank(current.Difficulty)
	if candidateRank != currentRank {
		return candidateRank < currentRank
	}
	if !candidate.CreatedAt.Equal(current.CreatedAt) {
		return candidate.CreatedAt.Before(current.CreatedAt)
	}

	return candidate.ID.String() < current.ID.String()
}
