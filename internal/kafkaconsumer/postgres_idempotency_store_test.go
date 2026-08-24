package kafkaconsumer_test

import (
	"testing"

	"github.com/300nn/go-identity-service/internal/kafkaconsumer"
	"github.com/300nn/go-identity-service/internal/testutils"
)

func TestPostgresIdempotencyStore_MarkAndCheckProcessed(t *testing.T) {
	ctx := t.Context()

	pool := testutils.NewTestPostgresPool(t)
	store := kafkaconsumer.NewPostgresIdempotencyStore(pool)

	event := kafkaconsumer.Event{
		EventID:   "event-1",
		EventType: "user.registered",
		Topic:     "outbox.events",
		Partition: 0,
		Offset:    1,
	}

	processed, err := store.WasProcessed(ctx, event.EventID)
	if err != nil {
		t.Fatalf("WasProcessed returned error: %v", err)
	}
	if processed {
		t.Fatal("expected event to be unprocessed")
	}

	if err := store.MarkProcessed(ctx, event); err != nil {
		t.Fatalf("MarkProcessed returned error: %v", err)
	}

	processed, err = store.WasProcessed(ctx, event.EventID)
	if err != nil {
		t.Fatalf("WasProcessed returned error: %v", err)
	}
	if !processed {
		t.Fatal("expected event to be processed")
	}

	if err := store.MarkProcessed(ctx, event); err != nil {
		t.Fatalf("second MarkProcessed should be idempotent, got: %v", err)
	}
}
