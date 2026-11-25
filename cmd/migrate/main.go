package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"oge-ege/internal/shared/config"
	"oge-ege/migrations"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "migrate: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errors.New("укажите команду: up, down [n] или version")
	}

	databaseURL, err := config.LoadDatabaseURL()
	if err != nil {
		return err
	}

	m, err := newMigrate(databaseURL)
	if err != nil {
		return err
	}
	defer func() {
		_, _ = m.Close()
	}()

	switch args[0] {
	case "up":
		err = m.Up()
	case "down":
		steps := 1
		if len(args) > 1 {
			steps, err = strconv.Atoi(args[1])
			if err != nil || steps < 1 {
				return fmt.Errorf("количество шагов отката должно быть положительным целым числом")
			}
		}
		err = m.Steps(-steps)
	case "version":
		return printVersion(m)
	default:
		return fmt.Errorf("неизвестная команда %q", args[0])
	}

	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("выполнить миграции: %w", err)
	}

	return nil
}

func printVersion(m *migrate.Migrate) error {
	version, dirty, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		if _, err := fmt.Fprintln(os.Stdout, "версия миграций: нет"); err != nil {
			return fmt.Errorf("вывести версию миграций: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("получить версию миграций: %w", err)
	}

	if _, err := fmt.Fprintf(os.Stdout, "версия миграций: %d, dirty: %t\n", version, dirty); err != nil {
		return fmt.Errorf("вывести версию миграций: %w", err)
	}

	return nil
}

func newMigrate(databaseURL string) (*migrate.Migrate, error) {
	source, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return nil, fmt.Errorf("открыть встроенные миграции: %w", err)
	}

	migrateURL, err := pgxURL(databaseURL)
	if err != nil {
		return nil, err
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, migrateURL)
	if err != nil {
		return nil, fmt.Errorf("создать мигратор: %w", err)
	}

	return m, nil
}

func pgxURL(databaseURL string) (string, error) {
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		return "", fmt.Errorf("разобрать DATABASE_URL: %w", err)
	}
	parsed.Scheme = "pgx5"

	return parsed.String(), nil
}
