package kafkaconsumer

import (
	"context"
	"fmt"
	"time"

	"github.com/300nn/go-identity-service/internal/audit"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TxStores struct {
	IdempotencyStore IdempotencyStore
	UserAuditStore   audit.Store
}

type TxFactory interface {
	WithinTx(ctx context.Context, fn func(TxStores) error) error
}

type PostgresTxFactory struct {
	pool         *pgxpool.Pool
	queryTimeout time.Duration
}

func NewPostgresTxFactory(pool *pgxpool.Pool, timeout time.Duration) *PostgresTxFactory {
	return &PostgresTxFactory{
		pool:         pool,
		queryTimeout: timeout,
	}
}

func (f *PostgresTxFactory) WithinTx(ctx context.Context, fn func(TxStores) error) error {
	tx, err := f.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin kafka consumer tx: %w", err)
	}

	stores := TxStores{
		IdempotencyStore: NewPostgresIdempotencyStore(tx, f.queryTimeout),
		UserAuditStore:   audit.NewPostgresStore(tx, f.queryTimeout),
	}

	if err := fn(stores); err != nil {
		rollbackCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if rollbackErr := tx.Rollback(rollbackCtx); rollbackErr != nil {
			return fmt.Errorf(
				"rollback kafka consumer tx: %v; original error: %w",
				rollbackErr,
				err,
			)
		}

		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit kafka consumer tx: %w", err)
	}

	return nil
}
