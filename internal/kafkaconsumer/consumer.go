package kafkaconsumer

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/twmb/franz-go/pkg/kgo"
)

type Config struct {
	Brokers       []string
	Topic         string
	ConsumerGroup string
}

type Consumer struct {
	client    *kgo.Client
	handler   EventHandler
	txFactory TxFactory
	logger    *slog.Logger
}

func NewConsumer(cfg Config, handler EventHandler, txFactory TxFactory, logger *slog.Logger) (*Consumer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers are required")
	}
	if cfg.Topic == "" {
		return nil, fmt.Errorf("kafka topic is required")
	}
	if cfg.ConsumerGroup == "" {
		return nil, fmt.Errorf("kafka consumer group is required")
	}
	if handler == nil {
		return nil, fmt.Errorf("kafka event handler is required")
	}
	if txFactory == nil {
		return nil, fmt.Errorf("kafka consumer tx factory is required")
	}

	client, err := kgo.NewClient(
		kgo.SeedBrokers(cfg.Brokers...),
		kgo.ClientID("go-crud-api-consumer"),
		kgo.ConsumerGroup(cfg.ConsumerGroup),
		kgo.ConsumeTopics(cfg.Topic),
		kgo.DisableAutoCommit(),
	)

	if err != nil {
		return nil, fmt.Errorf("create kafka client: %w", err)
	}

	return &Consumer{
		client:    client,
		handler:   handler,
		txFactory: txFactory,
		logger:    logger,
	}, nil
}

func (c *Consumer) Run(ctx context.Context) error {
	c.logger.Info("kafka consumer started")

	for {
		fetches := c.client.PollFetches(ctx)
		if ctx.Err() != nil {
			c.logger.Info("kafka consumer stopped")
			return ctx.Err()
		}

		if errs := fetches.Errors(); len(errs) > 0 {
			for _, fetchErr := range errs {
				c.logger.Error(
					"kafka fetch error",
					"topic", fetchErr.Topic,
					"partition", fetchErr.Partition,
					slog.Any("error", fetchErr.Err),
				)
			}
			continue
		}
		c.processFetches(ctx, fetches)
	}
}

func (c *Consumer) processFetches(ctx context.Context, fetches kgo.Fetches) {
	fetches.EachPartition(func(partition kgo.FetchTopicPartition) {
		for _, record := range partition.Records {
			if err := c.processRecord(ctx, record); err != nil {
				c.logger.Error(
					"kafka record processing failed",
					"topic", record.Topic,
					"partition", record.Partition,
					"offset", record.Offset,
					slog.Any("error", err),
				)

				return
			}
		}
	})
}

func (c *Consumer) processRecord(ctx context.Context, record *kgo.Record) error {
	event, err := eventFromRecord(record)

	if err != nil {
		return err
	}

	var duplicate bool

	err = c.txFactory.WithinTx(ctx, func(stores TxStores) error {
		processed, err := stores.IdempotencyStore.WasProcessed(ctx, event.EventID)
		if err != nil {
			return err
		}

		if processed {
			duplicate = true
			return nil
		}

		if err := c.handler.Handle(ctx, event, stores); err != nil {
			return err
		}

		if err := stores.IdempotencyStore.MarkProcessed(ctx, event); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	if duplicate {
		c.logger.Info(
			"kafka event already processed",
			"event_id", event.EventID,
			"event_type", event.EventType,
			"topic", event.Topic,
			"partition", event.Partition,
			"offset", event.Offset,
		)
	}

	if err := c.client.CommitRecords(ctx, record); err != nil {
		return fmt.Errorf("commit kafka record: %w", err)
	}

	return nil
}

func eventFromRecord(record *kgo.Record) (Event, error) {
	headers := make(map[string]string, len(record.Headers))

	for _, header := range record.Headers {
		headers[header.Key] = string(header.Value)
	}

	eventID := headers["event_id"]

	if eventID == "" {
		return Event{}, fmt.Errorf("kafka record missing event_id header")
	}

	eventType := headers["event_type"]
	if eventType == "" {
		return Event{}, fmt.Errorf("kafka record missing event_type header")
	}

	aggregateType := headers["aggregate_type"]
	if aggregateType == "" {
		return Event{}, fmt.Errorf("kafka record missing aggregate_type header")
	}

	aggregateID := headers["aggregate_id"]
	if aggregateID == "" {
		return Event{}, fmt.Errorf("kafka record missing aggregate_id header")
	}

	return Event{
		EventID:       eventID,
		EventType:     eventType,
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		Payload:       record.Value,
		Topic:         record.Topic,
		Partition:     record.Partition,
		Offset:        record.Offset,
	}, nil
}

func (c *Consumer) Close() {
	c.client.Close()
}
