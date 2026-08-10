package user

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TxRepositoryFactory interface {
	WithinTx(ctx context.Context, fn func(repo Repository) error) error
}

type PostgresTxRepositoryFactory struct {
	pool *pgxpool.Pool
}

func NewPostgresTxRepositoryFactory(pool *pgxpool.Pool) *PostgresTxRepositoryFactory {
	return &PostgresTxRepositoryFactory{
		pool: pool,
	}
}

func (f *PostgresTxRepositoryFactory) WithinTx(ctx context.Context, fn func(repo Repository) error) error {
	tx, err := f.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	txRepo := NewPostgresRepository(tx)

	if err := fn(txRepo); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
