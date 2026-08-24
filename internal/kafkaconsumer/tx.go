package kafkaconsumer

import (
	"context"
	"fmt"

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
	pool *pgxpool.Pool
}

func NewPostgresTxFactory(pool *pgxpool.Pool) *PostgresTxFactory {
	return &PostgresTxFactory{
		pool: pool,
	}
}

func (f *PostgresTxFactory) WithinTx(ctx context.Context, fn func(TxStores) error) error {
	tx, err := f.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin kafka consumer tx: %w", err)
	}

	stores := TxStores{
		IdempotencyStore: NewPostgresIdempotencyStore(tx),
		UserAuditStore:   audit.NewPostgresStore(tx),
	}

	if err := fn(stores); err != nil {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
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
