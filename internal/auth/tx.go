package auth

import (
	"CrudTutorialProject/internal/user"
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TxStores struct {
	UserStore         UserStore
	RefreshTokenStore RefreshTokenStore
}

type TxFactory interface {
	WithinTx(ctx context.Context, fn func(stores TxStores) error) error
}

type PostgresTxFactory struct {
	pool *pgxpool.Pool
}

func NewPostgresTxFactory(pool *pgxpool.Pool) *PostgresTxFactory {
	return &PostgresTxFactory{
		pool: pool,
	}
}

func (f *PostgresTxFactory) WithinTx(ctx context.Context, fn func(stores TxStores) error) error {
	tx, err := f.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin auth transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	stores := TxStores{
		UserStore:         user.NewPostgresRepository(tx),
		RefreshTokenStore: NewRefreshTokenRepository(tx),
	}

	if err := fn(stores); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit auth transaction: %w", err)
	}

	return nil
}
