package audit

import "time"

type UserAuditEvent struct {
	ID            int64
	SourceEventID string
	UserID        int64
	EventType     string
	Payload       string
	CreatedAt     time.Time
}
