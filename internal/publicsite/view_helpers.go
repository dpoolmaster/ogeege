package publicsite

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"oge-ege/internal/content"
	"oge-ege/internal/shared/render"
	"oge-ege/internal/shared/render/ui"
)

func brandName(brand string) string {
	return render.Chrome{Brand: brand}.BrandName()
}

func uiCrumbs(crumbs []Breadcrumb) []ui.Crumb {
	items := make([]ui.Crumb, 0, len(crumbs))
	for _, crumb := range crumbs {
		items = append(items, ui.Crumb{Title: crumb.Title, Path: crumb.Path})
	}

	return items
}

func (p *BankPage) kimNumber(kimNumberID uuid.UUID) int {
	for _, kim := range p.KimNumbers {
		if kim.ID == kimNumberID {
			return kim.Number
		}
	}

	return 0
}

func (p *BankPage) topicTitle(task content.Task) string {
	if task.TopicID == nil {
		return ""
	}
	for _, topic := range p.Topics {
		if topic.ID == *task.TopicID {
			return topic.Title
		}
	}

	return ""
}

func (p *BankPage) taskCard(task content.Task) ui.TaskCardProps {
	return ui.TaskCardProps{
		KimNumber: p.kimNumber(task.KimNumberID),
		Topic:     p.topicTitle(task),
		Excerpt:   plainExcerpt(task.StatementMD, 160),
		Href:      "/bank/" + task.ID.String(),
	}
}

func (p *BankPage) nextURL() string {
	values := url.Values{}
	if p.Filter.ExamCode != "" {
		values.Set("exam", p.Filter.ExamCode)
	}
	if p.Filter.KimNumber > 0 {
		values.Set("kim", strconv.Itoa(p.Filter.KimNumber))
	}
	if p.Filter.TopicID != nil {
		values.Set("topic", p.Filter.TopicID.String())
	}
	values.Set("cursor", p.NextCursor.String())

	return "/bank?" + values.Encode()
}

// Fact — факт экзамена для блока статистики.
type Fact struct {
	Value string
	Label string
}

func examFacts(page *LandingPage) []Fact {
	if page.Year == nil {
		return nil
	}

	return []Fact{
		{Value: intString(page.Year.Year), Label: "учебный год"},
		{Value: intString(page.Year.DurationMinutes), Label: "минут на экзамен"},
		{Value: intString(len(page.KimNumbers)), Label: "номеров КИМ"},
	}
}

var (
	excerptLink = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	excerptTeX  = regexp.MustCompile("\\\\[a-zA-Z]+|[${}^_*`#]")
)

func plainExcerpt(markdown string, limit int) string {
	markdown = excerptLink.ReplaceAllString(markdown, "$1")
	markdown = excerptTeX.ReplaceAllString(markdown, "")
	cleaned := strings.Join(strings.Fields(markdown), " ")
	runes := []rune(cleaned)
	if len(runes) <= limit {
		return cleaned
	}

	return strings.TrimSpace(string(runes[:limit])) + "…"
}

func topicRows(topics []TopicResult) [][]string {
	rows := make([][]string, 0, len(topics))
	for _, topic := range topics {
		rows = append(rows, []string{
			topic.Title,
			intString(topic.Correct),
			intString(topic.Total),
		})
	}

	return rows
}

func answerTypeLabel(answerType content.AnswerType) string {
	switch answerType {
	case content.AnswerTypeShort:
		return "Краткий"
	case content.AnswerTypeExtended:
		return "Развёрнутый"
	default:
		return string(answerType)
	}
}

func kimRows(kimNumbers []content.KimNumber) [][]string {
	rows := make([][]string, 0, len(kimNumbers))
	for _, kim := range kimNumbers {
		rows = append(rows, []string{
			intString(kim.Number),
			intString(kim.Part),
			answerTypeLabel(kim.AnswerType),
			intString(kim.MaxPrimaryScore),
		})
	}

	return rows
}
