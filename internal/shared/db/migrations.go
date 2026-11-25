package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const migrationStateQuery = `SELECT version, dirty FROM public.schema_migrations`

// MigrationState описывает текущее состояние миграций в базе.
type MigrationState struct {
	Version uint
	Dirty   bool
	Exists  bool
}

// MigrationStateReader читает состояние миграций; интерфейс упрощает тесты.
type MigrationStateReader interface {
	ReadMigrationState(ctx context.Context) (MigrationState, error)
}

// PgMigrationStateReader читает состояние таблицы golang-migrate.
type PgMigrationStateReader struct {
	db Querier
}

// NewPgMigrationStateReader создаёт читатель состояния миграций.
func NewPgMigrationStateReader(db Querier) *PgMigrationStateReader {
	return &PgMigrationStateReader{db: db}
}

// ReadMigrationState возвращает версию миграций из public.schema_migrations.
func (r *PgMigrationStateReader) ReadMigrationState(ctx context.Context) (MigrationState, error) {
	var (
		version int64
		dirty   bool
	)

	err := r.db.QueryRow(ctx, migrationStateQuery).Scan(&version, &dirty)
	if errors.Is(err, pgx.ErrNoRows) {
		return MigrationState{Exists: true}, nil
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "42P01" {
			return MigrationState{}, nil
		}
		return MigrationState{}, fmt.Errorf("прочитать состояние миграций: %w", err)
	}

	return MigrationState{Version: uint(version), Dirty: dirty, Exists: true}, nil
}

// MigrationChecker сравнивает применённую версию с ожидаемой.
type MigrationChecker struct {
	reader   MigrationStateReader
	expected uint
}

// NewMigrationChecker создаёт проверку готовности миграций.
func NewMigrationChecker(reader MigrationStateReader, expected uint) *MigrationChecker {
	return &MigrationChecker{reader: reader, expected: expected}
}

// Check возвращает причину неготовности, если миграции не применены.
func (c *MigrationChecker) Check(ctx context.Context) error {
	state, err := c.reader.ReadMigrationState(ctx)
	if err != nil {
		return fmt.Errorf("проверить миграции: %w", err)
	}
	if !state.Exists {
		return errors.New("таблица миграций отсутствует")
	}
	if state.Dirty {
		return fmt.Errorf("миграции в состоянии dirty (версия %d)", state.Version)
	}
	if state.Version != c.expected {
		return fmt.Errorf("применена версия миграций %d, ожидается %d", state.Version, c.expected)
	}

	return nil
}
