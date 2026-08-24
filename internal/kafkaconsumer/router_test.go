package kafkaconsumer_test

import (
	"testing"

	"github.com/300nn/go-identity-service/internal/kafkaconsumer"
)

func TestRouter_Handle_RoutesByEventType(t *testing.T) {
	ctx := t.Context()

	handler := &fakeEventHandler{}
	router := kafkaconsumer.NewRouter()
	router.Register("user.registered", handler)

	err := router.Handle(ctx, kafkaconsumer.Event{
		EventID:   "1",
		EventType: "user.registered",
		Payload:   []byte(`{"userId":1}`),
	}, kafkaconsumer.TxStores{})
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	if handler.calls != 1 {
		t.Fatalf("expected handler calls 1, got %d", handler.calls)
	}
}
