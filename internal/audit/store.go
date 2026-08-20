package audit

import "context"

type Store interface {
	CreateUserAuditEvent(ctx context.Context, user UserAuditEvent) (UserAuditEvent, error)
	CountBySourceEventID(ctx context.Context, sourceEventID string) (int, error)
}
