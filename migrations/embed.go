// Package migrations содержит встроенные SQL-миграции.
package migrations

import (
	"embed"
	"fmt"
	"io/fs"
	"strconv"
	"strings"
)

// FS содержит все SQL-миграции, встроенные в бинарник.
//
//go:embed *.sql
var FS embed.FS

// LatestVersion возвращает номер последней встроенной UP-миграции.
func LatestVersion() (uint, error) {
	entries, err := fs.ReadDir(FS, ".")
	if err != nil {
		return 0, fmt.Errorf("прочитать встроенные миграции: %w", err)
	}

	var latest uint
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}

		prefix, _, found := strings.Cut(entry.Name(), "_")
		if !found {
			return 0, fmt.Errorf("некорректное имя миграции %q", entry.Name())
		}

		version, err := strconv.ParseUint(prefix, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("некорректный номер миграции %q: %w", entry.Name(), err)
		}
		if uint(version) > latest {
			latest = uint(version)
		}
	}

	if latest == 0 {
		return 0, fmt.Errorf("встроенные миграции не найдены")
	}

	return latest, nil
}
