package outbox_test

import (
	"testing"
	"time"

	"github.com/300nn/go-identity-service/internal/outbox"
	"github.com/300nn/go-identity-service/internal/testutils"
)

func TestPostgresRepository_Create(t *testing.T) {
	ctx := t.Context()

	pool := testutils.NewTestPostgresPool(t)
	repo := outbox.NewPostgresRepository(pool, time.Second)

	created, err := repo.Create(ctx, outbox.Event{
		EventType:     outbox.EventTypeUserRegistered,
		AggregateType: outbox.AggregateUser,
		AggregateID:   "123",
		Payload:       []byte(`{"userId":1}`),
		ContentType:   "application/json",
		EventVersion:  "v1",
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if created.ID == 0 {
		t.Fatal("expected event ID to be set")
	}

	if created.Status != outbox.StatusNew {
		t.Fatalf("expected status to be %s, got %s", outbox.StatusNew, created.Status)
	}

	if created.EventType != outbox.EventTypeUserRegistered {
		t.Fatalf("expected event type to be %s, got %s", outbox.EventTypeUserRegistered, created.EventType)
	}

	if len(created.Payload) == 0 {
		t.Fatal("expected payload to be set")
	}
}

func TestPostgresRepository_Create_InvalidContentType(t *testing.T) {
	ctx := t.Context()

	pool := testutils.NewTestPostgresPool(t)
	repo := outbox.NewPostgresRepository(pool, time.Second)

	_, err := repo.Create(ctx, outbox.Event{
		EventType:     outbox.EventTypeUserRegistered,
		AggregateType: outbox.AggregateUser,
		AggregateID:   "123",
		Payload:       []byte("payload"),
		ContentType:   "invalid/content-type",
		EventVersion:  "v1",
	})
	if err == nil {
		t.Fatal("expected error for invalid content type")
	}
}

func TestPostgresRepository_FetchBatch(t *testing.T) {
	ctx := t.Context()

	pool := testutils.NewTestPostgresPool(t)
	repo := outbox.NewPostgresRepository(pool, time.Second)

	_, err := repo.Create(ctx, outbox.Event{
		EventType:     outbox.EventTypeUserRegistered,
		AggregateType: outbox.AggregateUser,
		AggregateID:   "1",
		Payload:       []byte(`{"userId":1}`),
		ContentType:   "application/json",
		EventVersion:  "v1",
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	events, err := repo.FetchBatch(ctx, 10, time.Minute)
	if err != nil {
		t.Fatalf("FetchBatch returned error: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0].Status != outbox.StatusProcessing {
		t.Fatalf("expected status %q, got %q", outbox.StatusProcessing, events[0].Status)
	}

	if events[0].Attempts != 1 {
		t.Fatalf("expected attempts 1, got %d", events[0].Attempts)
	}

	if events[0].LockedAt == nil {
		t.Fatal("expected locked_at to be set")
	}
}

func TestPostgresRepository_MarkProcessed(t *testing.T) {
	ctx := t.Context()

	pool := testutils.NewTestPostgresPool(t)
	repo := outbox.NewPostgresRepository(pool, time.Second)

	_, err := repo.Create(ctx, outbox.Event{
		EventType:     outbox.EventTypeUserRegistered,
		AggregateType: outbox.AggregateUser,
		AggregateID:   "1",
		Payload:       []byte(`{"userId":1}`),
		ContentType:   "application/json",
		EventVersion:  "v1",
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	events, err := repo.FetchBatch(ctx, 10, time.Minute)
	if err != nil {
		t.Fatalf("FetchBatch returned error: %v", err)
	}

	if err := repo.MarkProcessed(ctx, events[0].ID); err != nil {
		t.Fatalf("MarkProcessed returned error: %v", err)
	}

	events, err = repo.FetchBatch(ctx, 10, time.Minute)
	if err != nil {
		t.Fatalf("FetchBatch returned error: %v", err)
	}

	if len(events) != 0 {
		t.Fatalf("expected processed event not to be fetched again, got %d", len(events))
	}
}

func TestPostgresRepository_MarkFailed_ReturnsToNewWhenAttemptsRemain(t *testing.T) {
	ctx := t.Context()

	pool := testutils.NewTestPostgresPool(t)
	repo := outbox.NewPostgresRepository(pool, time.Second)

	_, err := repo.Create(ctx, outbox.Event{
		EventType:     outbox.EventTypeUserRegistered,
		AggregateType: outbox.AggregateUser,
		AggregateID:   "1",
		Payload:       []byte(`{"userId":1}`),
		ContentType:   "application/json",
		EventVersion:  "v1",
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	events, err := repo.FetchBatch(ctx, 10, time.Minute)
	if err != nil {
		t.Fatalf("FetchBatch returned error: %v", err)
	}

	if err := repo.MarkFailed(ctx, events[0].ID, "temporary error", 3); err != nil {
		t.Fatalf("MarkFailed returned error: %v", err)
	}

	events, err = repo.FetchBatch(ctx, 10, time.Minute)
	if err != nil {
		t.Fatalf("FetchBatch returned error: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected event to be retried, got %d events", len(events))
	}

	if events[0].Attempts != 2 {
		t.Fatalf("expected attempts 2, got %d", events[0].Attempts)
	}

	if events[0].LockedAt == nil {
		t.Fatal("expected locked_at to be set after refetch")
	}
}

func TestPostgresRepository_MarkFailed_MarksFailedWhenMaxAttemptsReached(t *testing.T) {
	ctx := t.Context()

	pool := testutils.NewTestPostgresPool(t)
	repo := outbox.NewPostgresRepository(pool, time.Second)

	_, err := repo.Create(ctx, outbox.Event{
		EventType:     outbox.EventTypeUserRegistered,
		AggregateType: outbox.AggregateUser,
		AggregateID:   "1",
		Payload:       []byte(`{"userId":1}`),
		ContentType:   "application/json",
		EventVersion:  "v1",
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	events, err := repo.FetchBatch(ctx, 10, time.Minute)
	if err != nil {
		t.Fatalf("FetchBatch returned error: %v", err)
	}

	if err := repo.MarkFailed(ctx, events[0].ID, "permanent error", 1); err != nil {
		t.Fatalf("MarkFailed returned error: %v", err)
	}

	events, err = repo.FetchBatch(ctx, 10, time.Minute)
	if err != nil {
		t.Fatalf("FetchBatch returned error: %v", err)
	}

	if len(events) != 0 {
		t.Fatalf("expected FAILED event not to be fetched again, got %d", len(events))
	}
}

func TestPostgresRepository_FetchBatch_DoesNotFetchFreshProcessingEvent(t *testing.T) {
	ctx := t.Context()

	pool := testutils.NewTestPostgresPool(t)
	repo := outbox.NewPostgresRepository(pool, time.Second)

	_, err := repo.Create(ctx, outbox.Event{
		EventType:     outbox.EventTypeUserRegistered,
		AggregateType: outbox.AggregateUser,
		AggregateID:   "1",
		Payload:       []byte(`{"userId":1}`),
		ContentType:   "application/json",
		EventVersion:  "v1",
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	firstBatch, err := repo.FetchBatch(ctx, 10, time.Minute)
	if err != nil {
		t.Fatalf("first FetchBatch returned error: %v", err)
	}

	if len(firstBatch) != 1 {
		t.Fatalf("expected 1 event, got %d", len(firstBatch))
	}

	secondBatch, err := repo.FetchBatch(ctx, 10, time.Minute)
	if err != nil {
		t.Fatalf("second FetchBatch returned error: %v", err)
	}

	if len(secondBatch) != 0 {
		t.Fatalf("expected fresh PROCESSING event not to be fetched, got %d", len(secondBatch))
	}
}

func TestPostgresRepository_FetchBatch_RefetchesStaleProcessingEvent(t *testing.T) {
	ctx := t.Context()

	pool := testutils.NewTestPostgresPool(t)
	repo := outbox.NewPostgresRepository(pool, time.Second)

	_, err := repo.Create(ctx, outbox.Event{
		EventType:     outbox.EventTypeUserRegistered,
		AggregateType: outbox.AggregateUser,
		AggregateID:   "1",
		Payload:       []byte(`{"userId":1}`),
		ContentType:   "application/json",
		EventVersion:  "v1",
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	firstBatch, err := repo.FetchBatch(ctx, 10, time.Minute)
	if err != nil {
		t.Fatalf("first FetchBatch returned error: %v", err)
	}

	if len(firstBatch) != 1 {
		t.Fatalf("expected 1 event, got %d", len(firstBatch))
	}

	_, err = pool.Exec(
		ctx,
		`UPDATE outbox_events SET locked_at = now() - interval '10 minutes' WHERE id = $1`,
		firstBatch[0].ID,
	)
	if err != nil {
		t.Fatalf("make event stale: %v", err)
	}

	secondBatch, err := repo.FetchBatch(ctx, 10, time.Minute)
	if err != nil {
		t.Fatalf("second FetchBatch returned error: %v", err)
	}

	if len(secondBatch) != 1 {
		t.Fatalf("expected stale PROCESSING event to be fetched, got %d", len(secondBatch))
	}

	if secondBatch[0].ID != firstBatch[0].ID {
		t.Fatalf("expected event id %d, got %d", firstBatch[0].ID, secondBatch[0].ID)
	}

	if secondBatch[0].Attempts != 2 {
		t.Fatalf("expected attempts 2, got %d", secondBatch[0].Attempts)
	}
}
