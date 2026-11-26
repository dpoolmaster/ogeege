package help

import (
	"io/fs"
	"strings"
	"testing"
)

func TestRegistryCompleteness(t *testing.T) {
	seen := make(map[string]struct{})

	for _, section := range Sections() {
		section := section
		t.Run(section.ID, func(t *testing.T) {
			if section.ID == "" {
				t.Fatal("пустой идентификатор раздела")
			}
			if _, ok := seen[section.ID]; ok {
				t.Fatalf("идентификатор %q повторяется", section.ID)
			}
			seen[section.ID] = struct{}{}

			if strings.TrimSpace(section.Title) == "" {
				t.Fatal("не задан заголовок раздела")
			}
			if section.File == "" {
				t.Fatal("не задан файл документа")
			}
			if len(section.Roles) == 0 {
				t.Fatal("не заданы роли раздела")
			}

			source, err := fs.ReadFile(Docs(), section.File)
			if err != nil {
				t.Fatalf("прочитать документ: %v", err)
			}
			if strings.TrimSpace(string(source)) == "" {
				t.Fatal("документ пуст")
			}
		})
	}

	if len(seen) != 11 {
		t.Fatalf("разделов в реестре = %d, ожидалось 11", len(seen))
	}
}
