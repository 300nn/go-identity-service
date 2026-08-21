package eventcodec_test

import (
	"CrudTutorialProject/internal/eventcodec"
	"testing"
)

func TestMarshalAndUnmarshalUserRegistered(t *testing.T) {
	data, err := eventcodec.MarshalUserRegistered(
		123,
		"t@ex.com",
		"USER",
	)

	if err != nil {
		t.Fatalf("MarshalUserRegistered() error = %v", err)
	}

	if len(data) == 0 {
		t.Fatal("expected protobuf payload to be non-empty")
	}

	payload, err := eventcodec.UnmarshalUserRegistered(data)
	if err != nil {
		t.Fatalf("UnmarshalUserRegistered() error = %v", err)
	}

	if payload.UserId != 123 {
		t.Fatalf("expected payload.UserId to be 123, got %d", payload.UserId)
	}

	if payload.Email != "t@ex.com" {
		t.Fatalf("expected payload.Email to be t@ex.com, got %s", payload.Email)
	}

	if payload.Role != "USER" {
		t.Fatalf("expected payload.Role to be USER, got %s", payload.Role)
	}
}
