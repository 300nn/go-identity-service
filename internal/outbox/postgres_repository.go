package outbox

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type PostgresRepository struct {
	db DBTX
}

func NewPostgresRepository(db DBTX) *PostgresRepository {
	return &PostgresRepository{
		db: db,
	}
}

func (r *PostgresRepository) Create(ctx context.Context, event Event) (Event, error) {
	const query = `
		insert into outbox_events (
		    event_type,
		    aggregate_type, 
		    aggregate_id,
		    payload
		)
		values ($1, $2, $3, $4::jsonb)
		returning 
		    id,
		    event_type,
		    aggregate_type,
		    aggregate_id,
		    payload::text,
		    status,
		    attempts,
		    last_error,
		    locked_at,
		    processed_at,
		    created_at
	`

	var created Event

	err := r.db.QueryRow(ctx, query, event.EventType, event.AggregateType, event.AggregateID, event.Payload).Scan(
		&created.ID,
		&created.EventType,
		&created.AggregateType,
		&created.AggregateID,
		&created.Payload,
		&created.Status,
		&created.Attempts,
		&created.LastError,
		&created.LockedAt,
		&created.ProcessedAt,
		&created.CreatedAt,
	)

	if err != nil {
		return Event{}, fmt.Errorf("insert outbox event: %w", err)
	}

	return created, nil
}
