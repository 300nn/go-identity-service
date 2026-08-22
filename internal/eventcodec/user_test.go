package eventcodec_test

import (
	"testing"

	"CrudTutorialProject/internal/eventcodec"
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

	if payload.GetUserId() != 123 {
		t.Fatalf("expected user id 123, got %d", payload.GetUserId())
	}

	if payload.GetEmail() != "t@ex.com" {
		t.Fatalf("expected payload.Email to be t@ex.com, got %s", payload.GetEmail())
	}

	if payload.GetRole() != "USER" {
		t.Fatalf("expected payload.Role to be USER, got %s", payload.GetRole())
	}
}
