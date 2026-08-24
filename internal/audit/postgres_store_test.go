package audit_test

import (
	"testing"
	"time"

	"github.com/300nn/go-identity-service/internal/audit"
	"github.com/300nn/go-identity-service/internal/testutils"
)

func TestPostgresStore_CreateUserAuditEvent(t *testing.T) {
	ctx := t.Context()

	pool := testutils.NewTestPostgresPool(t)
	store := audit.NewPostgresStore(pool, time.Second)

	created, err := store.CreateUserAuditEvent(ctx, audit.UserAuditEvent{
		SourceEventID: "event-1",
		UserID:        123,
		EventType:     "user.registered",
		Payload:       `{"userId":123,"email":"alex@example.com","role":"USER"}`,
	})
	if err != nil {
		t.Fatalf("CreateUserAuditEvent returned error: %v", err)
	}

	if created.ID == 0 {
		t.Fatal("expected id to be set")
	}

	if created.SourceEventID != "event-1" {
		t.Fatalf("expected source event id %q, got %q", "event-1", created.SourceEventID)
	}

	count, err := store.CountBySourceEventID(ctx, "event-1")
	if err != nil {
		t.Fatalf("CountBySourceEventID returned error: %v", err)
	}

	if count != 1 {
		t.Fatalf("expected count 1, got %d", count)
	}
}

func TestPostgresStore_CreateUserAuditEvent_DuplicateSourceEventID(t *testing.T) {
	ctx := t.Context()

	pool := testutils.NewTestPostgresPool(t)
	store := audit.NewPostgresStore(pool, time.Second)

	event := audit.UserAuditEvent{
		SourceEventID: "event-1",
		UserID:        123,
		EventType:     "user.registered",
		Payload:       `{"userId":123}`,
	}

	if _, err := store.CreateUserAuditEvent(ctx, event); err != nil {
		t.Fatalf("first CreateUserAuditEvent returned error: %v", err)
	}

	if _, err := store.CreateUserAuditEvent(ctx, event); err == nil {
		t.Fatal("expected duplicate source_event_id error, got nil")
	}
}
