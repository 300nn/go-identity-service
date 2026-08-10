package user

import "time"

const EventTypeUserCreated = "USER_CREATED"

type Event struct {
	ID        int64
	UserID    int64
	EventType string
	Payload   string
	CreatedAt time.Time
}
