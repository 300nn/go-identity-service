package kafkaconsumer_test

import (
	"errors"
	"testing"
	"time"

	"github.com/300nn/go-identity-service/internal/kafkaconsumer"
	"github.com/300nn/go-identity-service/internal/testutils"
)

var errTxFailed = errors.New("tx failed")

func TestPostgresTxFactory_RollsBackOnError(t *testing.T) {
	ctx := t.Context()

	pool := testutils.NewTestPostgresPool(t)
	txFactory := kafkaconsumer.NewPostgresTxFactory(pool, time.Second)
	idempotencyStore := kafkaconsumer.NewPostgresIdempotencyStore(pool, time.Second)

	event := kafkaconsumer.Event{
		EventID:   "event-1",
		EventType: "user.registered",
		Topic:     "outbox.events",
		Partition: 0,
		Offset:    1,
	}

	err := txFactory.WithinTx(ctx, func(stores kafkaconsumer.TxStores) error {
		if err := stores.IdempotencyStore.MarkProcessed(ctx, event); err != nil {
			return err
		}

		return errTxFailed
	})
	if !errors.Is(err, errTxFailed) {
		t.Fatalf("expected errTxFailed, got %v", err)
	}

	processed, err := idempotencyStore.WasProcessed(ctx, event.EventID)
	if err != nil {
		t.Fatalf("WasProcessed returned error: %v", err)
	}

	if processed {
		t.Fatal("expected event mark to be rolled back")
	}
}

func TestPostgresTxFactory_CommitsOnSuccess(t *testing.T) {
	ctx := t.Context()

	pool := testutils.NewTestPostgresPool(t)
	txFactory := kafkaconsumer.NewPostgresTxFactory(pool, time.Second)
	idempotencyStore := kafkaconsumer.NewPostgresIdempotencyStore(pool, time.Second)

	event := kafkaconsumer.Event{
		EventID:   "event-1",
		EventType: "user.registered",
		Topic:     "outbox.events",
		Partition: 0,
		Offset:    1,
	}

	err := txFactory.WithinTx(ctx, func(stores kafkaconsumer.TxStores) error {
		return stores.IdempotencyStore.MarkProcessed(ctx, event)
	})
	if err != nil {
		t.Fatalf("WithinTx returned error: %v", err)
	}

	processed, err := idempotencyStore.WasProcessed(ctx, event.EventID)
	if err != nil {
		t.Fatalf("WasProcessed returned error: %v", err)
	}

	if !processed {
		t.Fatal("expected event mark to be committed")
	}
}
