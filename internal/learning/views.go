package learning

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"oge-ege/internal/shared/render/ui"
)

func parseFormInt(value string) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0
	}

	return parsed
}

func cabinetNav(active string) []ui.NavGroup {
	return []ui.NavGroup{{
		Title: "Подготовка",
		Items: []ui.NavItem{
			{Label: "Сегодня", Href: "/app", Active: active == "today"},
			{Label: "Диагностика", Href: "/app/diagnostics", Active: active == "diagnostics"},
			{Label: "Карта пробелов", Href: "/app/gaps", Active: active == "gaps"},
			{Label: "План", Href: "/app/plan", Active: active == "plan"},
			{Label: "Цель", Href: "/app/goal", Active: active == "goal"},
		},
	}}
}

func selectClass(fieldError string) string {
	if fieldError != "" {
		return "v-field v-field--invalid"
	}

	return "v-field"
}

func ariaInvalid(fieldError string) string {
	if fieldError != "" {
		return "true"
	}

	return "false"
}

func kindLabel(kind ItemKind) string {
	switch kind {
	case KindDiagnostic:
		return "Диагностика"
	case KindTheory:
		return "Теория"
	case KindPractice:
		return "Практика"
	case KindReview:
		return "Повторение"
	default:
		return string(kind)
	}
}

func phaseLabel(phase Phase) string {
	switch phase {
	case PhaseDiagnostics:
		return "Диагностика"
	case PhaseBase:
		return "База"
	case PhasePractice:
		return "Практика"
	case PhaseReview:
		return "Повторение"
	default:
		return string(phase)
	}
}

func statusLabel(status ItemStatus) string {
	if status == ItemDone {
		return "выполнено"
	}

	return "запланировано"
}

func formatDate(value time.Time) string {
	return value.Format("02.01.2006")
}

func scoreText(value float64) string {
	if value == float64(int(value)) {
		return strconv.Itoa(int(value))
	}

	return strconv.FormatFloat(value, 'f', 1, 64)
}

func resultPercent(value, max float64) int {
	if max <= 0 {
		return 0
	}

	return int(value * 100 / max)
}

func answerText(answer *Answer) string {
	if answer == nil {
		return ""
	}

	return answer.RawAnswer
}

func itemTarget(item PlanItemView) string {
	switch {
	case item.TopicTitle != "":
		return item.TopicTitle
	case item.KimNumber > 0:
		return fmt.Sprintf("Номер КИМ %d", item.KimNumber)
	default:
		return "Общая подготовка"
	}
}

func resultRows(view ResultView) [][]string {
	rows := make([][]string, 0, len(view.Items))
	for _, item := range view.Items {
		result := "неверно"
		if item.Correct {
			result = "верно"
		}
		rows = append(rows, []string{
			strconv.Itoa(item.Position),
			strconv.Itoa(item.KimNumber),
			item.TopicTitle,
			result,
			scoreText(item.Awarded) + " из " + scoreText(item.MaxPoints),
			item.Reason,
		})
	}

	return rows
}
