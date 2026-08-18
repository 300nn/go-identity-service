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
