package help

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"oge-ege/internal/auth"
	"oge-ege/internal/shared/markdown"
)

var (
	// ErrNotFound возвращается для незарегистрированного раздела.
	ErrNotFound = errors.New("раздел справки не найден")
	// ErrForbidden возвращается, когда роль не имеет доступа к разделу.
	ErrForbidden = errors.New("нет доступа к справке раздела")
)

// Document — готовая справка раздела.
type Document struct {
	Title string
	HTML  string
}

// Service загружает документы справки и рендерит их в безопасный HTML.
type Service struct {
	docs fs.FS
}

// NewService создаёт сервис справки на встроенных документах.
func NewService() *Service {
	return &Service{docs: docsFS}
}

// Get возвращает справку раздела с учётом роли пользователя.
func (s *Service) Get(ctx context.Context, id string, role auth.Role) (Document, error) {
	section, ok := Lookup(id)
	if !ok {
		return Document{}, ErrNotFound
	}
	if !allows(section, role) {
		return Document{}, ErrForbidden
	}

	source, err := fs.ReadFile(s.docs, section.File)
	if err != nil {
		return Document{}, fmt.Errorf("прочитать документ справки %q: %w", section.ID, err)
	}

	html, err := markdown.Render(string(source))
	if err != nil {
		return Document{}, fmt.Errorf("отрендерить документ справки %q: %w", section.ID, err)
	}

	return Document{Title: section.Title, HTML: html}, nil
}

func allows(section Section, role auth.Role) bool {
	for _, allowed := range section.Roles {
		if allowed == role {
			return true
		}
	}

	return false
}
