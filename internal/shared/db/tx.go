package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WithTx выполняет fn в транзакции и фиксирует её только при успехе.
func WithTx(ctx context.Context, pool *pgxpool.Pool, fn func(pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("начать транзакцию: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := fn(tx); err != nil {
		return fmt.Errorf("выполнить транзакцию: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("зафиксировать транзакцию: %w", err)
	}

	return nil
}
