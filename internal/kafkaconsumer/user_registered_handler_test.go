package kafkaconsumer_test

import (
	"CrudTutorialProject/internal/audit"
	"CrudTutorialProject/internal/kafkaconsumer"
	"CrudTutorialProject/internal/testutils"
	"io"
	"log/slog"
	"testing"
)

func TestUserRegisteredHandler_Handle_CreateAuditEvent(t *testing.T) {
	ctx := t.Context()

	pool := testutils.NewTestPostgresPool(t)
	txFactory := kafkaconsumer.NewPostgresTxFactory(pool)
	auditStore := audit.NewPostgresStore(pool)

	handler := kafkaconsumer.NewUserRegisteredHandler(discardLogger())

	event := kafkaconsumer.Event{
		EventID:       "event-1",
		EventType:     "user.registered",
		AggregateType: "user",
		AggregateID:   "123",
		Payload:       []byte(`{"userId":123,"email":"alex@example.com","role":"USER"}`),
		Topic:         "outbox.events",
		Partition:     0,
		Offset:        1,
	}

	err := txFactory.WithinTx(ctx, func(stores kafkaconsumer.TxStores) error {
		if err := handler.Handle(ctx, event, stores); err != nil {
			return err
		}

		return stores.IdempotencyStore.MarkProcessed(ctx, event)
	})

	if err != nil {
		t.Fatalf("WithinTx returned error: %v", err)
	}

	count, err := auditStore.CountBySourceEventID(ctx, "event-1")

	if err != nil {
		t.Fatalf("CountBySourceEventID returned error: %v", err)
	}

	if count != 1 {
		t.Fatalf("CountBySourceEventID expected 1, got %d", count)
	}
}

func TestUserRegisteredHandler_Handle_RollbackAuditEvent(t *testing.T) {
	ctx := t.Context()

	pool := testutils.NewTestPostgresPool(t)
	txFactory := kafkaconsumer.NewPostgresTxFactory(pool)
	auditStore := audit.NewPostgresStore(pool)

	handler := kafkaconsumer.NewUserRegisteredHandler(discardLogger())

	event := kafkaconsumer.Event{
		EventID:       "event-1",
		EventType:     "user.registered",
		AggregateType: "user",
		AggregateID:   "123",
		Payload:       []byte(`{"userId":123,"email":"alex@example.com","role":"USER"}`),
		Topic:         "outbox.events",
		Partition:     0,
		Offset:        1,
	}

	err := txFactory.WithinTx(ctx, func(stores kafkaconsumer.TxStores) error {
		if err := handler.Handle(ctx, event, stores); err != nil {
			return err
		}

		return errTxFailed
	})

	if err == nil {
		t.Fatal("expected tx error, got nil")
	}

	count, err := auditStore.CountBySourceEventID(ctx, "event-1")

	if err != nil {
		t.Fatalf("CountBySourceEventID returned error: %v", err)
	}

	if count != 0 {
		t.Fatalf("CountBySourceEventID expected 0 after rollback, got %d", count)
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
