package app

import (
	"CrudTutorialProject/internal/response"
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"
)

type HealthHandlers struct {
	db           *pgxpool.Pool
	logger       *slog.Logger
	shuttingDown *atomic.Bool
}

func NewHealthHandlers(db *pgxpool.Pool, logger *slog.Logger, shuttingDown *atomic.Bool) *HealthHandlers {
	return &HealthHandlers{
		db:           db,
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

func (h *HealthHandlers) Ready(w http.ResponseWriter, r *http.Request) {
	if h.shuttingDown.Load() {
		response.JSON(w, http.StatusOK, map[string]string{
			"status": "shutting_down",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 500*time.Microsecond)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		h.logger.Warn("readiness check failed", slog.Any("error", err))

		response.JSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "not_ready",
			"reason": "database_unavailable",
		})
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"status": "ready",
	})
}
