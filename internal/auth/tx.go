package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/300nn/go-identity-service/internal/outbox"
	"github.com/300nn/go-identity-service/internal/user"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TxStores struct {
	UserStore         UserStore
	RefreshTokenStore RefreshTokenStore
	OutboxStore       OutboxStore
}

type OutboxStore interface {
	Create(ctx context.Context, event outbox.Event) (outbox.Event, error)
}

type TxFactory interface {
	WithinTx(ctx context.Context, fn func(stores TxStores) error) error
}

type PostgresTxFactory struct {
	pool         *pgxpool.Pool
	queryTimeout time.Duration
}

func NewPostgresTxFactory(pool *pgxpool.Pool, queryTimeout time.Duration) *PostgresTxFactory {
	return &PostgresTxFactory{
		pool:         pool,
		queryTimeout: queryTimeout,
	}
}

func (f *PostgresTxFactory) WithinTx(ctx context.Context, fn func(stores TxStores) error) error {
	tx, err := f.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin auth transaction: %w", err)
	}

	defer func() {
		rollbackCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = tx.Rollback(rollbackCtx)
	}()

	stores := TxStores{
		UserStore:         user.NewPostgresRepository(tx, f.queryTimeout),
		RefreshTokenStore: NewRefreshTokenRepository(tx, f.queryTimeout),
		OutboxStore:       outbox.NewPostgresRepository(tx, f.queryTimeout),
	}

	if err := fn(stores); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit auth transaction: %w", err)
	}

	return nil
}
