package outbox

import (
	"context"
	"log/slog"
)

type LogPublisher struct {
	logger *slog.Logger
}

func NewLogPublisher(logger *slog.Logger) *LogPublisher {
	return &LogPublisher{
		logger: logger,
	}
}

func (p *LogPublisher) Publish(ctx context.Context, event Event) error {
	p.logger.Info("outbox event published",
		"event_id", event.ID,
		"event_type", event.EventType,
		"aggregate_type", event.AggregateType,
		"aggregate_id", event.AggregateID,
	)

	return nil
}
