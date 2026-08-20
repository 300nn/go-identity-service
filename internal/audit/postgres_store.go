package audit

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type PostgresStore struct {
	db DBTX
}

func NewPostgresStore(db DBTX) *PostgresStore {
	return &PostgresStore{
		db: db,
	}
}

func (s *PostgresStore) CreateUserAuditEvent(ctx context.Context, event UserAuditEvent) (UserAuditEvent, error) {
	const query = `
		insert into user_audit_events (
		    source_event_id,
		    user_id,
		    event_type,
		    payload
		) values ($1, $2, $3, $4::jsonb)
		returning 
			id,
		    source_event_id,
		    user_id,
		    event_type,
		    payload::text,
		    created_at
	`

	var created UserAuditEvent

	err := s.db.QueryRow(
		ctx,
		query,
		event.SourceEventID,
		event.UserID,
		event.EventType,
		event.Payload,
	).Scan(
		&created.ID,
		&created.SourceEventID,
		&created.UserID,
		&created.EventType,
		&created.Payload,
		&created.CreatedAt,
	)

	if err != nil {
		return UserAuditEvent{}, fmt.Errorf("create audit event: %w", err)
	}

	return created, nil
}
func (s *PostgresStore) CountBySourceEventID(ctx context.Context, sourceEventID string) (int, error) {
	const query = `
		select count(*) 
		from user_audit_events 
		where source_event_id = $1	
	`

	var count int

	if err := s.db.QueryRow(ctx, query, sourceEventID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count user audit events by source event id: %w", err)
	}

	return count, nil
}
