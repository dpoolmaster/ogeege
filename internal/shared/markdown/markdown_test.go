package markdown

import (
	"strings"
	"testing"
)

func TestRenderSanitizesHTML(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		contains string
		absent   string
	}{
		{
			name:     "удаляет script",
			source:   `Текст <script>alert(1)</script>`,
			contains: "Текст",
			absent:   "<script>",
		},
		{
			name:     "удаляет обработчик события",
			source:   `<p onclick="alert(1)">Текст</p>`,
			contains: "<p>Текст</p>",
			absent:   "onclick",
		},
		{
			name:     "сохраняет разметку",
			source:   "**жирный** и *курсив*\n\n- один\n- два",
			contains: "<strong>жирный</strong>",
			absent:   "<script>",
		},
		{
			name:     "сохраняет таблицу",
			source:   "| a | b |\n|---|---|\n| 1 | 2 |",
			contains: "<table>",
			absent:   "<script>",
		},
		{
			name:     "сохраняет формулу",
			source:   "Формула $x^2 + y^2$",
			contains: "$x^2 + y^2$",
			absent:   "<script>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Render(tt.source)
			if err != nil {
				t.Fatalf("Render: %v", err)
			}
			if !strings.Contains(got, tt.contains) {
				t.Fatalf("в результате нет %q: %s", tt.contains, got)
			}
			if strings.Contains(got, tt.absent) {
				t.Fatalf("в результате остался %q: %s", tt.absent, got)
			}
		})
	}
}

func TestValidateFormulas(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantErr bool
	}{
		{name: "корректная формула", source: `$x^2$`},
		{name: "корректная дробь", source: `$$\frac{1}{2}$$`},
		{name: "корректное окружение", source: `$\begin{matrix}1\end{matrix}$`},
		{name: "незакрытая скобка", source: `$x^{2$`, wantErr: true},
		{name: "лишняя скобка", source: `$x^2}$`, wantErr: true},
		{name: "незакрытое окружение", source: `$\begin{matrix}1$`, wantErr: true},
		{name: "несбалансированный left", source: `$\left( x$`, wantErr: true},
		{name: "пустая формула", source: `$$`, wantErr: true},
		{name: "одиночный доллар в тексте", source: `цена 5$`},
		{name: "доллар в середине текста", source: `стоимость $5 без формулы`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFormulas(tt.source)
			if tt.wantErr && err == nil {
				t.Fatal("ожидалась ошибка")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("неожиданная ошибка: %v", err)
			}
		})
	}
}
