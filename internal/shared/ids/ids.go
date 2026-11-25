// Package ids предоставляет единый способ генерации идентификаторов.
package ids

import (
	"fmt"

	"github.com/google/uuid"
)

// New возвращает новый UUID версии 7.
func New() (uuid.UUID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, fmt.Errorf("сгенерировать UUID v7: %w", err)
	}

	return id, nil
}
