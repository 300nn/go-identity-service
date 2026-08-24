package user

import (
	"context"
	"fmt"
	"time"

	"github.com/300nn/go-identity-service/internal/outbox"
	"github.com/300nn/go-identity-service/internal/timex"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TxRepositoryFactory interface {
	WithinTx(ctx context.Context, fn func(stores TxStores) error) error
}

type TxStores struct {
	UserRepo    Repository
	OutBoxStore OutboxStore
}

type OutboxStore interface {
	Create(ctx context.Context, event outbox.Event) (outbox.Event, error)
}

type PostgresTxRepositoryFactory struct {
	pool         *pgxpool.Pool
	queryTimeout time.Duration
}

func NewPostgresTxRepositoryFactory(pool *pgxpool.Pool, timeout time.Duration) *PostgresTxRepositoryFactory {
	return &PostgresTxRepositoryFactory{
		pool:         pool,
		queryTimeout: timeout,
	}
}

func (f *PostgresTxRepositoryFactory) WithinTx(ctx context.Context, fn func(stores TxStores) error) error {
	ctx, cancel := timex.WithTimeout(ctx, f.queryTimeout)
	defer cancel()

	tx, err := f.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		rollbackCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = tx.Rollback(rollbackCtx)
	}()

	stores := TxStores{
		UserRepo:    NewPostgresRepository(tx, f.queryTimeout),
		OutBoxStore: outbox.NewPostgresRepository(tx, f.queryTimeout),
	}

	if err := fn(stores); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
