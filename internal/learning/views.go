package learning

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

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
			{Label: "Практика", Href: "/app/practice", Active: active == "practice"},
			{Label: "Повторение", Href: "/app/review", Active: active == "review"},
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

func sessionTitle(kind string) string {
	switch kind {
	case SessionKindPractice:
		return "Практика"
	case SessionKindReview:
		return "Работа над ошибками"
	default:
		return "Диагностика"
	}
}

func sessionResultTitle(kind string) string {
	return "Результат: " + strings.ToLower(sessionTitle(kind))
}

func sessionItemMeta(kind string) string {
	switch kind {
	case SessionKindPractice:
		return "Задание практики"
	case SessionKindReview:
		return "Задание повторения"
	default:
		return "Задание диагностики"
	}
}

func sessionFinishLabel(kind string) string {
	switch kind {
	case SessionKindPractice:
		return "Завершить практику"
	case SessionKindReview:
		return "Завершить повторение"
	default:
		return "Завершить диагностику"
	}
}

func sessionNav(kind string) string {
	switch kind {
	case SessionKindPractice:
		return "practice"
	case SessionKindReview:
		return "review"
	default:
		return "diagnostics"
	}
}

func choiceInputType(spec json.RawMessage) string {
	var payload struct {
		OptionIDs []string `json:"option_ids"`
	}
	if err := json.Unmarshal(spec, &payload); err == nil && len(payload.OptionIDs) > 1 {
		return "checkbox"
	}

	return "radio"
}

func answerHasOption(answer *Answer, optionID uuid.UUID) bool {
	if answer == nil {
		return false
	}
	for _, value := range strings.FieldsFunc(answer.RawAnswer, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\n' || r == '\t'
	}) {
		if value == optionID.String() {
			return true
		}
	}

	return false
}
