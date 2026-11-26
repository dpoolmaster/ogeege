package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TxManager выполняет функцию в транзакции.
// Интерфейс объявляют сервисы, чтобы тесты могли подменять транзакции.
type TxManager struct {
	pool *pgxpool.Pool
}

// NewTxManager создаёт менеджер транзакций.
func NewTxManager(pool *pgxpool.Pool) *TxManager {
	return &TxManager{pool: pool}
}

// WithTx выполняет fn в транзакции и передаёт её как Querier.
func (m *TxManager) WithTx(ctx context.Context, fn func(q Querier) error) error {
	return WithTx(ctx, m.pool, func(tx pgx.Tx) error {
		return fn(tx)
	})
}
