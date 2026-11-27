// Package ui содержит компоненты дизайн-системы поверх Pico.css.
package ui

import (
	"crypto/sha256"
	"encoding/binary"
	"strconv"
	"strings"

	"github.com/a-h/templ"
)

// Tone — смысловой оттенок компонента.
type Tone string

const (
	ToneNeutral Tone = "neutral"
	ToneAccent  Tone = "accent"
	ToneOK      Tone = "ok"
	ToneErr     Tone = "err"
	ToneWarn    Tone = "warn"
	ToneTeal    Tone = "teal"
	ToneOlive   Tone = "olive"
	ToneBrown   Tone = "brown"
	ToneOchre   Tone = "ochre"
)

// ButtonVariant — визуальный вариант кнопки.
type ButtonVariant string

const (
	ButtonPrimary   ButtonVariant = "primary"
	ButtonSecondary ButtonVariant = "secondary"
	ButtonGhost     ButtonVariant = "ghost"
	ButtonDanger    ButtonVariant = "danger"
)

// ButtonSize — размер кнопки.
type ButtonSize string

const (
	ButtonSmall  ButtonSize = "sm"
	ButtonMedium ButtonSize = "md"
)

// ButtonProps описывает кнопку или ссылку-кнопку.
type ButtonProps struct {
	Label    string
	Href     string
	Type     string
	Variant  ButtonVariant
	Size     ButtonSize
	Disabled bool
	Loading  bool
	Name     string
	Value    string
}

func (p ButtonProps) buttonType() string {
	if p.Type == "" {
		return "button"
	}

	return p.Type
}

func (p ButtonProps) classes() string {
	variant := p.Variant
	if variant == "" {
		variant = ButtonPrimary
	}
	size := p.Size
	if size == "" {
		size = ButtonMedium
	}

	classes := []string{"v-btn", "v-btn--" + string(variant), "v-btn--" + string(size)}
	if p.Loading {
		classes = append(classes, "v-btn--loading")
	}

	return strings.Join(classes, " ")
}

// FieldProps описывает поле формы.
type FieldProps struct {
	Label        string
	Name         string
	Type         string
	Value        string
	Hint         string
	Error        string
	ID           string
	Autocomplete string
	Placeholder  string
	Required     bool
	Disabled     bool
	Multiline    bool
	Rows         int
	MinLength    int
	MaxLength    int
}

func (p FieldProps) id() string {
	if p.ID != "" {
		return p.ID
	}

	return "field-" + p.Name
}

func (p FieldProps) inputType() string {
	if p.Type == "" {
		return "text"
	}

	return p.Type
}

func (p FieldProps) rows() string {
	rows := p.Rows
	if rows <= 0 {
		rows = 4
	}

	return strconv.Itoa(rows)
}

func (p FieldProps) classes() string {
	if p.Error != "" {
		return "v-field v-field--invalid"
	}

	return "v-field"
}

func (p FieldProps) describedBy() string {
	if p.Error == "" {
		return ""
	}

	return p.id() + "-error"
}

// CardProps описывает карточку.
type CardProps struct {
	Title string
	Meta  string
	Href  string
}

// TabItem — вкладка.
type TabItem struct {
	Label    string
	Href     string
	Disabled bool
}

// TabsProps описывает набор вкладок.
type TabsProps struct {
	Items  []TabItem
	Active string
}

// TableProps описывает таблицу.
type TableProps struct {
	Head    []string
	Rows    [][]string
	Dense   bool
	Caption string
}

func (p TableProps) classes() string {
	if p.Dense {
		return "v-table v-table--dense"
	}

	return "v-table"
}

// PanelProps описывает боковую панель.
type PanelProps struct {
	Title string
	Side  string
}

func (p PanelProps) classes() string {
	side := p.Side
	if side != "left" {
		side = "right"
	}

	return "v-panel v-panel--" + side
}

// ModalProps описывает модальное окно.
type ModalProps struct {
	ID    string
	Title string
	Open  bool
}

// ToastProps описывает уведомление.
type ToastProps struct {
	Tone Tone
	Text string
}

// EmptyStateProps описывает пустое состояние.
type EmptyStateProps struct {
	Title       string
	Text        string
	ActionLabel string
	ActionHref  string
	Error       bool
}

// SkeletonProps описывает скелетон.
type SkeletonProps struct {
	Lines int
}

// ProgressProps описывает индикатор прогресса.
type ProgressProps struct {
	Value int
	Max   int
	Label string
}

func (p ProgressProps) max() int {
	if p.Max <= 0 {
		return 1
	}

	return p.Max
}

func (p ProgressProps) value() int {
	if p.Value < 0 {
		return 0
	}
	if p.Value > p.max() {
		return p.max()
	}

	return p.Value
}

func (p ProgressProps) text() string {
	return strconv.Itoa(p.value()) + " из " + strconv.Itoa(p.max())
}

// RingProps описывает круговой индикатор результата.
type RingProps struct {
	Value int
	Max   int
	Label string
}

func (p RingProps) max() int {
	if p.Max <= 0 {
		return 1
	}

	return p.Max
}

func (p RingProps) value() int {
	if p.Value < 0 {
		return 0
	}
	if p.Value > p.max() {
		return p.max()
	}

	return p.Value
}

func (p RingProps) percent() int {
	return p.value() * 100 / p.max()
}

func (p RingProps) text() string {
	return strconv.Itoa(p.value()) + " из " + strconv.Itoa(p.max())
}

// StepperProps описывает пошаговый индикатор.
type StepperProps struct {
	Current int
	Total   int
	Label   string
}

func (p StepperProps) total() int {
	if p.Total <= 0 {
		return 1
	}

	return p.Total
}

func (p StepperProps) current() int {
	if p.Current < 1 {
		return 1
	}
	if p.Current > p.total() {
		return p.total()
	}

	return p.Current
}

func (p StepperProps) text() string {
	return "Шаг " + strconv.Itoa(p.current()) + " из " + strconv.Itoa(p.total())
}

// ResultPanelProps описывает панель результата проверки.
type ResultPanelProps struct {
	Correct   bool
	Title     string
	Text      string
	NextHref  string
	NextLabel string
}

// Crumb — элемент хлебных крошек.
type Crumb struct {
	Title string
	Path  string
}

// TaskCardProps описывает карточку задания.
type TaskCardProps struct {
	KimNumber int
	Topic     string
	Excerpt   string
	Href      string
}

// GapState — состояние ячейки карты пробелов.
type GapState string

const (
	GapNone    GapState = "none"
	GapCorrect GapState = "correct"
	GapWrong   GapState = "wrong"
)

// GapCell — ячейка карты пробелов.
type GapCell struct {
	State GapState
	Label string
}

// GapHeatmapProps описывает карту пробелов.
type GapHeatmapProps struct {
	Topics     []string
	KimNumbers []int
	Cells      [][]GapCell
}

// GapSummary — краткая сводка по теме.
type GapSummary struct {
	Title string
	Count int
	Tone  Tone
}

// NavItem — пункт навигации.
type NavItem struct {
	Label  string
	Href   string
	Active bool
}

// NavGroup — группа навигации.
type NavGroup struct {
	Title string
	Items []NavItem
}

// AppShellProps описывает каркас закрытой части.
type AppShellProps struct {
	Brand     string
	PageTitle string
	Nav       []NavGroup
	Actions   templ.Component
}

// TodayScreenProps описывает экран-заглушку «Сегодня».
type TodayScreenProps struct {
	NextStep    string
	NextHref    string
	Gaps        []GapSummary
	Goal        string
	GoalCurrent int
	GoalMax     int
}

// SubjectTone детерминированно выбирает оттенок по коду трека.
func SubjectTone(code string) Tone {
	code = strings.TrimSpace(code)
	if code == "" {
		return ToneNeutral
	}

	sum := sha256.Sum256([]byte(code))
	palette := []Tone{ToneTeal, ToneOlive, ToneBrown, ToneOchre}
	index := binary.LittleEndian.Uint32(sum[len(sum)-4:]) % uint32(len(palette))

	return palette[index]
}

func toneClass(prefix string, tone Tone) string {
	switch tone {
	case ToneAccent, ToneOK, ToneErr, ToneWarn, ToneTeal, ToneOlive, ToneBrown, ToneOchre:
		return prefix + "--" + string(tone)
	default:
		return prefix + "--neutral"
	}
}

func gapSymbol(state GapState) string {
	switch state {
	case GapCorrect:
		return "✓"
	case GapWrong:
		return "✕"
	default:
		return "·"
	}
}

func gapLabel(cell GapCell) string {
	if cell.Label != "" {
		return cell.Label
	}

	switch cell.State {
	case GapCorrect:
		return "верно"
	case GapWrong:
		return "ошибка"
	default:
		return "не было"
	}
}

func gapTone(state GapState) Tone {
	switch state {
	case GapCorrect:
		return ToneOK
	case GapWrong:
		return ToneErr
	default:
		return ToneNeutral
	}
}

func cellAt(cells [][]GapCell, row, column int) GapCell {
	if row < 0 || row >= len(cells) {
		return GapCell{State: GapNone}
	}
	if column < 0 || column >= len(cells[row]) {
		return GapCell{State: GapNone}
	}

	return cells[row][column]
}

func ariaBool(value bool) string {
	if value {
		return "true"
	}

	return "false"
}

func intAttr(value int) string {
	if value <= 0 {
		return ""
	}

	return strconv.Itoa(value)
}

func tableLabel(caption string) string {
	if caption != "" {
		return caption
	}

	return "Таблица"
}

func emptyClasses(isError bool) string {
	if isError {
		return "v-empty v-empty--error"
	}

	return "v-empty"
}

func emptyRole(isError bool) string {
	if isError {
		return "alert"
	}

	return "status"
}

func skeletonLines(lines int) int {
	if lines <= 0 {
		return 1
	}

	return lines
}

func stepperLabel(label, fallback string) string {
	if label != "" {
		return label
	}

	return fallback
}

func stepClass(step, current int) string {
	switch {
	case step < current:
		return "v-stepper__step v-stepper__step--done"
	case step == current:
		return "v-stepper__step v-stepper__step--current"
	default:
		return "v-stepper__step"
	}
}

func resultTitle(title, fallback string) string {
	if title != "" {
		return title
	}

	return fallback
}

func ringStyle(percent int) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	return "background: conic-gradient(var(--v-accent) " + strconv.Itoa(percent) + "%, var(--v-line) 0);"
}

func ringLabel(label, fallback string) string {
	if label != "" {
		return label + ": " + fallback
	}

	return fallback
}

func toastRole(tone Tone) string {
	if tone == ToneErr {
		return "alert"
	}

	return "status"
}

func brandName(brand string) string {
	brand = strings.TrimSpace(brand)
	if brand == "" {
		return "Верно"
	}

	return brand
}
