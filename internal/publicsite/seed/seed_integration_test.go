package seed

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"

	"oge-ege/internal/publicsite"
	"oge-ege/internal/shared/clock"
	"oge-ege/internal/shared/db"
)

type seedTx struct {
	tx pgx.Tx
}

func (m seedTx) WithTx(_ context.Context, fn func(q db.Querier) error) error {
	return fn(m.tx)
}

func TestRunIsIdempotentIntegration(t *testing.T) {
	databaseURL, ok := os.LookupEnv("TEST_DATABASE_URL")
	if !ok || databaseURL == "" {
		t.Skip("TEST_DATABASE_URL не задан")
	}

	ctx := context.Background()
	pool, err := db.Open(ctx, databaseURL, 0, 2)
	if err != nil {
		t.Fatalf("открыть пул: %v", err)
	}
	defer pool.Close()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("начать транзакцию: %v", err)
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	txManager := seedTx{tx: tx}
	service := publicsite.NewService(publicsite.NewPgRepository(), tx, txManager, clock.New(), nil, nil, nil, logger)

	if err := Run(ctx, service, logger); err != nil {
		t.Fatalf("первый запуск: %v", err)
	}
	if err := Run(ctx, service, logger); err != nil {
		t.Fatalf("повторный запуск: %v", err)
	}

	tests := []struct {
		name  string
		query string
		arg   string
		want  int
	}{
		{name: "статья", query: `SELECT count(*) FROM public.articles WHERE slug = $1`, arg: "kak-gotovitsya-k-oge", want: 1},
		{name: "SEO-страница", query: `SELECT count(*) FROM public.seo_pages WHERE path = $1`, arg: "/bank", want: 1},
		{name: "редирект", query: `SELECT count(*) FROM public.redirects WHERE from_path = $1`, arg: "/oge", want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var count int
			if err := tx.QueryRow(ctx, tt.query, tt.arg).Scan(&count); err != nil {
				t.Fatalf("посчитать: %v", err)
			}
			if count != tt.want {
				t.Fatalf("записей = %d, ожидалось %d", count, tt.want)
			}
		})
	}
}
