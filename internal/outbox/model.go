package outbox

import "time"

type Status string

const (
	StatusNew        Status = "NEW"
	StatusProcessing Status = "PROCESSING"
	StatusProcessed  Status = "PROCESSED"
	StatusFailed     Status = "FAILED"
)

const (
	AggregateUser = "user"
)

const (
	EventTypeUserRegistered = "user.registered"
	EventTypeUserUpdated    = "user.updated"
	EventTypeUserDeleted    = "user.deleted"
)

type Event struct {
	ID            int64
	EventType     string
	AggregateType string
	AggregateID   string
	Payload       string
	Status        Status
	Attempts      int
	LastError     *string
	LockedAt      *time.Time
	ProcessedAt   *time.Time
	CreatedAt     time.Time
}
