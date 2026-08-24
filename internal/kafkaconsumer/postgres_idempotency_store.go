package kafkaconsumer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/300nn/go-identity-service/internal/timex"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type DBTX interface {
	Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, query string, args ...any) pgx.Row
}

type PostgresIdempotencyStore struct {
	db           DBTX
	queryTimeout time.Duration
}

func NewPostgresIdempotencyStore(db DBTX, timeout time.Duration) *PostgresIdempotencyStore {
	return &PostgresIdempotencyStore{db: db, queryTimeout: timeout}
}

func (s *PostgresIdempotencyStore) WasProcessed(ctx context.Context, eventID string) (bool, error) {
	ctx, cancel := timex.WithTimeout(ctx, s.queryTimeout)
	defer cancel()

	const query = `
		select 1 
		from processed_kafka_events 
		where event_id = $1
	`

	var one int
	err := s.db.QueryRow(ctx, query, eventID).Scan(&one)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}

		return false, fmt.Errorf("error querying processed kafka_events: %w", err)
	}

	return true, nil
}

func (s *PostgresIdempotencyStore) MarkProcessed(ctx context.Context, event Event) error {
	ctx, cancel := timex.WithTimeout(ctx, s.queryTimeout)
	defer cancel()

	const query = `
		INSERT INTO processed_kafka_events (
		    event_id,
		    topic,
		    partition,
		    offset_value,
		    event_type
		)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (event_id) DO NOTHING
	`

	_, err := s.db.Exec(
		ctx,
		query,
		event.EventID,
		event.Topic,
		event.Partition,
		event.Offset,
		event.EventType,
	)
	if err != nil {
		return fmt.Errorf("mark kafka event processed: %w", err)
	}

	return nil
}
