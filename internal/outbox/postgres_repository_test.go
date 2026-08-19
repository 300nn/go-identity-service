package outbox_test

import (
	"CrudTutorialProject/internal/outbox"
	"CrudTutorialProject/internal/testutils"
	"testing"
)

func TestPostgresRepository_Create(t *testing.T) {
	ctx := t.Context()

	pool := testutils.NewTestPostgresPool(t)
	repo := outbox.NewPostgresRepository(pool)

	created, err := repo.Create(ctx, outbox.Event{
		EventType:     outbox.EventTypeUserRegistered,
		AggregateType: outbox.AggregateUser,
		AggregateID:   "123",
		Payload:       `{"userId":123,"email":"alex@example.com","role":"USER"}`,
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

	if created.Payload == "" {
		t.Fatal("expected payload to be set")
	}
}

func TestPostgresRepository_Create_InvalidPayload(t *testing.T) {
	ctx := t.Context()

	pool := testutils.NewTestPostgresPool(t)
	repo := outbox.NewPostgresRepository(pool)

	_, err := repo.Create(ctx, outbox.Event{
		EventType:     outbox.EventTypeUserRegistered,
		AggregateType: outbox.AggregateUser,
		AggregateID:   "123",
		Payload:       `{invalid-json`,
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON payload")
	}
}

func TestPostgresRepository_FetchBatch(t *testing.T) {
	ctx := t.Context()

	pool := testutils.NewTestPostgresPool(t)
	repo := outbox.NewPostgresRepository(pool)

	_, err := repo.Create(ctx, outbox.Event{
		EventType:     outbox.EventTypeUserRegistered,
		AggregateType: outbox.AggregateUser,
		AggregateID:   "1",
		Payload:       `{"userId":1}`,
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	events, err := repo.FetchBatch(ctx, 10)
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
	repo := outbox.NewPostgresRepository(pool)

	_, err := repo.Create(ctx, outbox.Event{
		EventType:     outbox.EventTypeUserRegistered,
		AggregateType: outbox.AggregateUser,
		AggregateID:   "1",
		Payload:       `{"userId":1}`,
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	events, err := repo.FetchBatch(ctx, 10)
	if err != nil {
		t.Fatalf("FetchBatch returned error: %v", err)
	}

	if err := repo.MarkProcessed(ctx, events[0].ID); err != nil {
		t.Fatalf("MarkProcessed returned error: %v", err)
	}

	events, err = repo.FetchBatch(ctx, 10)
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
	repo := outbox.NewPostgresRepository(pool)

	_, err := repo.Create(ctx, outbox.Event{
		EventType:     outbox.EventTypeUserRegistered,
		AggregateType: outbox.AggregateUser,
		AggregateID:   "1",
		Payload:       `{"userId":1}`,
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	events, err := repo.FetchBatch(ctx, 10)
	if err != nil {
		t.Fatalf("FetchBatch returned error: %v", err)
	}

	if err := repo.MarkFailed(ctx, events[0].ID, "temporary error", 3); err != nil {
		t.Fatalf("MarkFailed returned error: %v", err)
	}

	events, err = repo.FetchBatch(ctx, 10)
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
	repo := outbox.NewPostgresRepository(pool)

	_, err := repo.Create(ctx, outbox.Event{
		EventType:     outbox.EventTypeUserRegistered,
		AggregateType: outbox.AggregateUser,
		AggregateID:   "1",
		Payload:       `{"userId":1}`,
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	events, err := repo.FetchBatch(ctx, 10)
	if err != nil {
		t.Fatalf("FetchBatch returned error: %v", err)
	}

	if err := repo.MarkFailed(ctx, events[0].ID, "permanent error", 1); err != nil {
		t.Fatalf("MarkFailed returned error: %v", err)
	}

	events, err = repo.FetchBatch(ctx, 10)
	if err != nil {
		t.Fatalf("FetchBatch returned error: %v", err)
	}

	if len(events) != 0 {
		t.Fatalf("expected FAILED event not to be fetched again, got %d", len(events))
	}
}
