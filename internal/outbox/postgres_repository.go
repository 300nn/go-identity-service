package outbox

import (
	"context"
	"fmt"
	"time"

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
		INSERT INTO outbox_events (
		    event_type,
		    aggregate_type,
		    aggregate_id,
		    payload_bytes,
		    content_type,
		    proto_message,
		    event_version
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING
		    id,
		    event_type,
		    aggregate_type,
		    aggregate_id,
		    payload_bytes,
		    content_type,
		    proto_message,
		    event_version,
		    status,
		    attempts,
		    last_error,
		    locked_at,
		    processed_at,
		    created_at
	`

	created, err := scanEvent(r.db.QueryRow(
		ctx,
		query,
		event.EventType,
		event.AggregateType,
		event.AggregateID,
		event.Payload,
		event.ContentType,
		event.ProtoMessage,
		event.EventVersion,
	))

	if err != nil {
		return Event{}, fmt.Errorf("insert outbox event: %w", err)
	}

	return created, nil
}

func (r *PostgresRepository) FetchBatch(ctx context.Context, limit int, lockTimeout time.Duration) ([]Event, error) {
	if limit <= 0 {
		return nil, nil
	}

	lockedBefore := time.Now().UTC().Add(-lockTimeout)

	const query = `
		update outbox_events
		set 
		    status = 'PROCESSING',
			locked_at = now(),
			attempts = attempts + 1
		where id in (
		    select id
		    from outbox_events
		    where status = 'NEW' 
		       or (
		           status = 'PROCESSING'
		           and locked_at is not null
		           and locked_at < $2
		       )
		    order by created_at
		    limit $1
		    for update skip locked 
		)
		RETURNING
			id,
			event_type,
			aggregate_type,
			aggregate_id,
			payload_bytes,
			content_type,
			proto_message,
			event_version,
			status,
			attempts,
			last_error,
			locked_at,
			processed_at,
			created_at
		`

	rows, err := r.db.Query(ctx, query, limit, lockedBefore)

	if err != nil {
		return nil, fmt.Errorf("fetch outbox events: %w", err)
	}
	defer rows.Close()

	events := make([]Event, 0)

	for rows.Next() {
		event, err := scanEventFromRows(rows)
		if err != nil {
			return nil, fmt.Errorf("scan outbox events: %w", err)
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate outbox events: %w", err)
	}

	return events, nil
}

func (r *PostgresRepository) MarkProcessed(ctx context.Context, id int64) error {
	const query = `
		update outbox_events
		set 
		    status = 'PROCESSED',
		    processed_at = now(),
		    locked_at = null,
		    last_error = null
		where id = $1 
		  and status = 'PROCESSING'
    	`
	tag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("mark outbox events: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("outbox event %d not found or not processing", id)
	}

	return nil
}

func (r *PostgresRepository) MarkFailed(ctx context.Context, id int64, reason string, maxAttempts int) error {
	const query = `
		update outbox_events
		set 
		    status = case 
		    	when attempts >= $3 then 'FAILED'
				else 'NEW'
			end,
		    locked_at = null,
		    last_error = $2
		where id = $1 
			and status = 'PROCESSING'
	`

	tag, err := r.db.Exec(ctx, query, id, reason, maxAttempts)
	if err != nil {
		return fmt.Errorf("mark outbox events: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("outbox event %d not found or not processing", id)
	}

	return nil
}

func scanEvent(row pgx.Row) (Event, error) {
	var event Event

	err := row.Scan(
		&event.ID,
		&event.EventType,
		&event.AggregateType,
		&event.AggregateID,
		&event.Payload,
		&event.ContentType,
		&event.ProtoMessage,
		&event.EventVersion,
		&event.Status,
		&event.Attempts,
		&event.LastError,
		&event.LockedAt,
		&event.ProcessedAt,
		&event.CreatedAt,
	)

	if err != nil {
		return Event{}, err
	}

	return event, nil
}

func scanEventFromRows(rows pgx.Rows) (Event, error) {
	var event Event

	err := rows.Scan(
		&event.ID,
		&event.EventType,
		&event.AggregateType,
		&event.AggregateID,
		&event.Payload,
		&event.ContentType,
		&event.ProtoMessage,
		&event.EventVersion,
		&event.Status,
		&event.Attempts,
		&event.LastError,
		&event.LockedAt,
		&event.ProcessedAt,
		&event.CreatedAt,
	)
	if err != nil {
		return Event{}, err
	}

	return event, nil
}
