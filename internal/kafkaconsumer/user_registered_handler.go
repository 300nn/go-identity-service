package kafkaconsumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
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

func (h *UserRegisteredHandler) Handle(ctx context.Context, event Event) error {
	var payload UserRegisteredPayload

	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal user registered payload: %w", err)
	}

	h.logger.Info(
		"user registered event consumed",
		"event_id", event.EventID,
		"user_id", payload.UserID,
		"email", payload.Email,
		"role", payload.Role,
	)

	return nil
}
