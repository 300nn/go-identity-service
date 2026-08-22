package kafkaconsumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"CrudTutorialProject/internal/audit"
	"CrudTutorialProject/internal/eventcodec"
)

type UserRegisteredPayload struct {
	UserID int64  `json:"userId"`
	Email  string `json:"email"`
	Role   string `json:"role"`
}

type UserRegisteredHandler struct {
	logger *slog.Logger
}

func NewUserRegisteredHandler(logger *slog.Logger) *UserRegisteredHandler {
	return &UserRegisteredHandler{
		logger: logger,
	}
}

func (h *UserRegisteredHandler) Handle(ctx context.Context, event Event, stores TxStores) error {
	if event.ContentType != eventcodec.ContentTypeProtobuf {
		return fmt.Errorf("unsupported user registered content type %q", event.ContentType)
	}

	if event.ProtoMessage != eventcodec.ProtoMessageUserRegistered {
		return fmt.Errorf("unsupported user registered proto message %q", event.ProtoMessage)
	}

	if event.EventVersion != eventcodec.EventVersionV1 {
		return fmt.Errorf("unsupported user registered event version %q", event.EventVersion)
	}

	payload, err := eventcodec.UnmarshalUserRegistered(event.Payload)
	if err != nil {
		return err
	}

	auditPayload, err := json.Marshal(UserRegisteredPayload{
		UserID: payload.UserId,
		Email:  payload.Email,
		Role:   payload.Role,
	})

	if err != nil {
		return fmt.Errorf("marshal user registered audit payload: %w", err)
	}

	_, err = stores.UserAuditStore.CreateUserAuditEvent(ctx, audit.UserAuditEvent{
		SourceEventID: event.EventID,
		UserID:        payload.UserId,
		EventType:     event.EventType,
		Payload:       string(auditPayload),
	})

	if err != nil {
		return fmt.Errorf("create user audit event: %w", err)
	}

	h.logger.Info(
		"user registered event consumed",
		"event_id", event.EventID,
		"user_id", payload.UserId,
		"email", payload.Email,
		"role", payload.Role,
	)

	return nil
}
