package db

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMigrationChecker(t *testing.T) {
	tests := []struct {
		name        string
		state       MigrationState
		readErr     error
		expected    uint
		wantErr     bool
		errContains string
	}{
		{
			name:     "миграции применены",
			state:    MigrationState{Version: 1, Exists: true},
			expected: 1,
		},
		{
			name:        "миграции не применены",
			state:       MigrationState{Version: 0, Exists: true},
			expected:    1,
			wantErr:     true,
			errContains: "ожидается 1",
		},
		{
			name:        "таблица отсутствует",
			state:       MigrationState{},
			expected:    1,
			wantErr:     true,
			errContains: "таблица миграций отсутствует",
		},
		{
			name:        "dirty-состояние",
			state:       MigrationState{Version: 1, Dirty: true, Exists: true},
			expected:    1,
			wantErr:     true,
			errContains: "dirty",
		},
		{
			name:        "ошибка чтения",
			readErr:     errors.New("нет связи"),
			expected:    1,
			wantErr:     true,
			errContains: "нет связи",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewMigrationChecker(fakeStateReader{state: tt.state, err: tt.readErr}, tt.expected)
			err := checker.Check(context.Background())
			if tt.wantErr {
				if err == nil {
					t.Fatal("ожидалась ошибка")
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("ошибка %q не содержит %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("неожиданная ошибка: %v", err)
			}
		})
	}
}

func TestQuerierSupportsPoolAndTx(t *testing.T) {
	type repository struct {
		db Querier
	}

	var pool *pgxpool.Pool
	var tx pgx.Tx

	_ = repository{db: pool}
	_ = repository{db: tx}
}

type fakeStateReader struct {
	state MigrationState
	err   error
}

func (f fakeStateReader) ReadMigrationState(context.Context) (MigrationState, error) {
	return f.state, f.err
}

func TestOpenIntegration(t *testing.T) {
	pool := openTestPool(t)
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		t.Fatalf("ping: %v", err)
	}
}

func TestWithTxIntegration(t *testing.T) {
	pool := openTestPool(t)
	defer pool.Close()

	ctx := context.Background()
	if _, err := pool.Exec(ctx, `DROP TABLE IF EXISTS public.tx_test`); err != nil {
		t.Fatalf("удалить тестовую таблицу: %v", err)
	}
	if _, err := pool.Exec(ctx, `CREATE TABLE public.tx_test (id integer PRIMARY KEY)`); err != nil {
		t.Fatalf("создать тестовую таблицу: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DROP TABLE IF EXISTS public.tx_test`)
	})

	if err := WithTx(ctx, pool, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO public.tx_test (id) VALUES (1)`)
		return err
	}); err != nil {
		t.Fatalf("commit: %v", err)
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM public.tx_test`).Scan(&count); err != nil {
		t.Fatalf("посчитать строки: %v", err)
	}
	if count != 1 {
		t.Fatalf("после commit ожидалась 1 строка, получено %d", count)
	}

	sentinel := errors.New("откат")
	err := WithTx(ctx, pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO public.tx_test (id) VALUES (2)`); err != nil {
			return err
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("ошибка должна сохранять причину, получено %v", err)
	}

	if err := pool.QueryRow(ctx, `SELECT count(*) FROM public.tx_test`).Scan(&count); err != nil {
		t.Fatalf("посчитать строки: %v", err)
	}
	if count != 1 {
		t.Fatalf("после rollback ожидалась 1 строка, получено %d", count)
	}
}

func openTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL, ok := os.LookupEnv("TEST_DATABASE_URL")
	if !ok || databaseURL == "" {
		t.Skip("TEST_DATABASE_URL не задан")
	}

	pool, err := Open(context.Background(), databaseURL, 0, 2)
	if err != nil {
		t.Fatalf("открыть пул: %v", err)
	}

	return pool
}
