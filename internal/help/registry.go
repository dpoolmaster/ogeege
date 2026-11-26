// Package help отдаёт справочные документы по разделам административного интерфейса.
package help

import (
	"embed"
	"io/fs"

	"oge-ege/internal/auth"
)

//go:embed docs
var docsFS embed.FS

// Section описывает раздел справки: заголовок, файл документа и допустимые роли.
type Section struct {
	ID    string
	Title string
	File  string
	Roles []auth.Role
}

var sections = []Section{
	{ID: "admin-exams", Title: "Треки", File: "docs/admin-exams.md", Roles: contentRoles()},
	{ID: "admin-exam-years", Title: "Учебные годы", File: "docs/admin-exam-years.md", Roles: contentRoles()},
	{ID: "admin-kim-numbers", Title: "Номера КИМ", File: "docs/admin-kim-numbers.md", Roles: contentRoles()},
	{ID: "admin-topics", Title: "Темы", File: "docs/admin-topics.md", Roles: contentRoles()},
	{ID: "admin-theory", Title: "Теория", File: "docs/admin-theory.md", Roles: contentRoles()},
	{ID: "admin-tasks", Title: "Задания", File: "docs/admin-tasks.md", Roles: contentRoles()},
	{ID: "admin-variants", Title: "Варианты", File: "docs/admin-variants.md", Roles: contentRoles()},
	{ID: "admin-rubrics", Title: "Рубрики", File: "docs/admin-rubrics.md", Roles: contentRoles()},
	{ID: "admin-score-scales", Title: "Шкалы", File: "docs/admin-score-scales.md", Roles: contentRoles()},
	{ID: "admin-users", Title: "Пользователи", File: "docs/admin-users.md", Roles: adminRoles()},
	{ID: "admin-audit", Title: "Аудит", File: "docs/admin-audit.md", Roles: adminRoles()},
}

// Sections возвращает копию реестра разделов в порядке объявления.
func Sections() []Section {
	result := make([]Section, len(sections))
	copy(result, sections)
	return result
}

// Lookup находит раздел по идентификатору.
func Lookup(id string) (Section, bool) {
	for _, section := range sections {
		if section.ID == id {
			return section, true
		}
	}

	return Section{}, false
}

// Docs возвращает встроенную файловую систему с документами.
func Docs() fs.FS {
	return docsFS
}

func contentRoles() []auth.Role {
	return []auth.Role{auth.RoleMethodist, auth.RoleAdmin}
}

func adminRoles() []auth.Role {
	return []auth.Role{auth.RoleAdmin}
}
