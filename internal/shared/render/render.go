// Package render рендерит Templ-компоненты в HTTP-ответ.
package render

import (
	"context"
	"fmt"
	"io"

	"github.com/a-h/templ"
)

// Component рендерит компонент в writer.
func Component(ctx context.Context, w io.Writer, component templ.Component) error {
	if err := component.Render(ctx, w); err != nil {
		return fmt.Errorf("отрендерить компонент: %w", err)
	}

	return nil
}
