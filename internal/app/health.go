package app

import (
	"context"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"CrudTutorialProject/internal/response"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

type PingerFunc func(ctx context.Context) error

func (f PingerFunc) Ping(ctx context.Context) error {
	return f(ctx)
}

type HealthHandlers struct {
	db           Pinger
	redis        Pinger
	kafka        Pinger
	logger       *slog.Logger
	shuttingDown *atomic.Bool
}

func NewHealthHandlers(
	db Pinger,
	redis Pinger,
	kafka Pinger,
	logger *slog.Logger,
	shuttingDown *atomic.Bool,
) *HealthHandlers {
	return &HealthHandlers{
		db:           db,
		redis:        redis,
		kafka:        kafka,
		logger:       logger,
		shuttingDown: shuttingDown,
	}
}

func (h *HealthHandlers) Health(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
	return
}

type readinessResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

func (h *HealthHandlers) Ready(w http.ResponseWriter, r *http.Request) {
	if h.shuttingDown.Load() {
		response.JSON(w, http.StatusServiceUnavailable, readinessResponse{
			Status: "shutting_down",
			Checks: map[string]string{
				"database": "skipped",
				"redis":    "skipped",
				"kafka":    "skipped",
			},
		})
		return
	}

	checks := map[string]string{
		"database": "unknown",
		"redis":    "unknown",
		"kafka":    "unknown",
	}

	ready := true

	if err := h.check(r.Context(), "database", h.db, checks); err != nil {
		ready = false
	}

	if err := h.check(r.Context(), "redis", h.redis, checks); err != nil {
		ready = false
	}

	if err := h.check(r.Context(), "kafka", h.kafka, checks); err != nil {
		ready = false
	}

	if !ready {
		response.JSON(w, http.StatusServiceUnavailable, readinessResponse{
			Status: "not_ready",
			Checks: checks,
		})
		return
	}

	response.JSON(w, http.StatusOK, readinessResponse{
		Status: "ready",
		Checks: checks,
	})
}

func (h *HealthHandlers) check(
	parentCtx context.Context,
	name string,
	checker Pinger,
	checks map[string]string,
) error {
	ctx, cancel := context.WithTimeout(parentCtx, 700*time.Millisecond)
	defer cancel()

	if err := checker.Ping(ctx); err != nil {
		checks[name] = "unavailable"

		h.logger.Warn(
			"readiness dependency check failed",
			"dependency", name,
			slog.Any("error", err),
		)

		return err
	}

	checks[name] = "ok"
	return nil
}
