package outbox_test

import (
	"CrudTutorialProject/internal/outbox"
	"io"
	"log/slog"
	"testing"
	"time"
)

func newTestWorker(t *testing.T, store outbox.Store, publisher outbox.Publisher) *outbox.Worker {
	t.Helper()

	return outbox.NewWorker(
		store,
		publisher,
		discardLogger(t),
		outbox.WorkerConfig{
			Interval:    time.Second,
			BatchSize:   10,
			MaxAttempts: 3,
		},
	)
}

func discardLogger(t *testing.T) *slog.Logger {
	t.Helper()
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestWorker_ProcessOnce_PublishesAndMarksProcessed(t *testing.T) {
	ctx := t.Context()

	store := &fakeStore{
		events: []outbox.Event{
			{
				ID:            1,
				EventType:     outbox.EventTypeUserRegistered,
				AggregateType: outbox.AggregateUser,
				AggregateID:   "1",
				Payload:       `{"userId":1}`,
				Status:        outbox.StatusProcessing,
			},
		},
	}

	publisher := &fakePublisher{}

	worker := newTestWorker(t, store, publisher)

	if err := worker.ProcessOnce(ctx); err != nil {
		t.Fatalf("ProcessOnce returned error: %v", err)
	}

	if len(publisher.published) != 1 {
		t.Fatalf("expected 1 published event, got %d", len(publisher.published))
	}

	if publisher.published[0].ID != 1 {
		t.Fatalf("expected published event id 1, got %d", publisher.published[0].ID)
	}

	if store.markProcessedCalls != 1 {
		t.Fatalf("expected MarkProcessed calls 1, got %d", store.markProcessedCalls)
	}

	if len(store.processedIDs) != 1 || store.processedIDs[0] != 1 {
		t.Fatalf("expected event 1 to be marked processed, got %v", store.processedIDs)
	}
}

func TestWorker_ProcessOnce_PublishFails_MarksFailed(t *testing.T) {
	ctx := t.Context()

	store := &fakeStore{
		events: []outbox.Event{
			{
				ID:            1,
				EventType:     outbox.EventTypeUserRegistered,
				AggregateType: outbox.AggregateUser,
				AggregateID:   "1",
				Payload:       `{"userId":1}`,
				Status:        outbox.StatusProcessing,
			},
		},
	}

	publisher := &fakePublisher{
		err: errPublishFailed,
	}

	worker := newTestWorker(t, store, publisher)

	if err := worker.ProcessOnce(ctx); err != nil {
		t.Fatalf("ProcessOnce returned error: %v", err)
	}

	if store.markFailedCalls != 1 {
		t.Fatalf("expected MarkFailed calls 1, got %d", store.markFailedCalls)
	}

	if len(store.failedIDs) != 1 || store.failedIDs[0] != 1 {
		t.Fatalf("expected event 1 to be marked failed, got %v", store.failedIDs)
	}

	if store.lastReason == "" {
		t.Fatal("expected failure reason to be set")
	}
}

func TestWorker_ProcessOnce_FetchBatchFails(t *testing.T) {
	ctx := t.Context()

	store := &fakeStore{
		fetchErr: errFetchBatchFailed,
	}

	publisher := &fakePublisher{}
	worker := newTestWorker(t, store, publisher)

	err := worker.ProcessOnce(ctx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestWorker_ProcessOnce_MarkProcessedFails(t *testing.T) {
	ctx := t.Context()

	store := &fakeStore{
		events: []outbox.Event{
			{
				ID:            1,
				EventType:     outbox.EventTypeUserRegistered,
				AggregateType: outbox.AggregateUser,
				AggregateID:   "1",
				Payload:       `{"userId":1}`,
				Status:        outbox.StatusProcessing,
			},
		},
		markProcessedErr: errMarkProcessedFailed,
	}

	publisher := &fakePublisher{}
	worker := newTestWorker(t, store, publisher)

	if err := worker.ProcessOnce(ctx); err != nil {
		t.Fatalf("ProcessOnce returned error: %v", err)
	}

	if len(publisher.published) != 1 {
		t.Fatalf("expected event to be published")
	}

	if store.markProcessedCalls != 1 {
		t.Fatalf("expected MarkProcessed calls 1, got %d", store.markProcessedCalls)
	}
}
