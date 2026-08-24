package outbox

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

type WorkerConfig struct {
	Interval       time.Duration `yaml:"interval" env:"INTERVAL" env-default:"5s" validate:"gt=0"`
	BatchSize      int           `yaml:"batch_size" env:"BATCH_SIZE" env-default:"10" validate:"gt=0"`
	MaxAttempts    int           `yaml:"max_attempts" env:"MAX_ATTEMPTS" env-default:"3" validate:"gt=0"`
	LockTimeout    time.Duration `yaml:"lock_timeout" env:"LOCK_TIMEOUT" env-default:"1m" validate:"gt=0"`
	ProcessTimeout time.Duration `yaml:"process_timeout" env:"PROCESS_TIMEOUT" env-default:"30s" validate:"gt=0"`
}

type WorkerOption func(*Worker)

func WithObserver(observer Observer) WorkerOption {
	return func(w *Worker) {
		if observer != nil {
			w.observer = observer
		}
	}
}

type Worker struct {
	store     Store
	publisher Publisher
	logger    *slog.Logger
	cfg       WorkerConfig
	observer  Observer
}

func NewWorker(
	store Store,
	publisher Publisher,
	logger *slog.Logger,
	cfg WorkerConfig,
	opts ...WorkerOption,
) *Worker {
	worker := &Worker{
		store:     store,
		publisher: publisher,
		logger:    logger,
		cfg:       cfg,
		observer:  NoopObserver{},
	}

	for _, opt := range opts {
		opt(worker)
	}

	return worker
}

func (w *Worker) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.cfg.Interval)
	defer ticker.Stop()

	w.logger.Info(
		"outbox worker started",
		"interval", w.cfg.Interval,
		"batch_size", w.cfg.BatchSize,
	)

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("outbox worker shutting down")
			return ctx.Err()
		default:
		}

		processCtx, cancel := context.WithTimeout(ctx, w.cfg.ProcessTimeout)

		if err := w.ProcessOnce(processCtx); err != nil {
			w.logger.Error("outbox process once failed", slog.Any("error", err))
		}
		cancel()

		select {
		case <-ctx.Done():
			w.logger.Info("outbox worker stopped")
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (w *Worker) ProcessOnce(ctx context.Context) error {
	events, err := w.store.FetchBatch(ctx, w.cfg.BatchSize, w.cfg.LockTimeout)
	if err != nil {
		return fmt.Errorf("fetch outbox batch: %w", err)
	}

	for _, event := range events {
		if err := w.processEvent(ctx, event); err != nil {
			w.logger.Error(
				"outbox event processing failed",
				"event_id", event.ID,
				"event_type", event.EventType,
				slog.Any("error", err),
			)
		}
	}

	return nil
}

func (w *Worker) processEvent(ctx context.Context, event Event) error {
	if err := w.publisher.Publish(ctx, event); err != nil {
		reason := err.Error()

		if len(reason) > 1000 {
			reason = reason[:1000]
		}

		if markErr := w.store.MarkFailed(ctx, event.ID, reason, w.cfg.MaxAttempts); markErr != nil {
			return fmt.Errorf("publish failed: %w; mark failed: %v", err, markErr)
		}

		w.observer.EventFailed(event.EventType)
		return err
	}

	if err := w.store.MarkProcessed(ctx, event.ID); err != nil {
		w.observer.EventFailed(event.EventType)
		return fmt.Errorf("mark processed: %w", err)
	}

	w.observer.EventProcessed(event.EventType)
	return nil
}
