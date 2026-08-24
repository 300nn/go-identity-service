package kafkaconsumer

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/300nn/go-identity-service/internal/timex"

	"github.com/twmb/franz-go/pkg/kgo"
)

type Config struct {
	Brokers        []string
	Topic          string
	ConsumerGroup  string
	ProcessTimeout time.Duration
}

type ConsumerOption func(*Consumer)

func WithObserver(observer Observer) ConsumerOption {
	return func(c *Consumer) {
		if observer != nil {
			c.observer = observer
		}
	}
}

type Consumer struct {
	client         *kgo.Client
	handler        EventHandler
	txFactory      TxFactory
	logger         *slog.Logger
	observer       Observer
	processTimeout time.Duration
}

func NewConsumer(cfg Config, handler EventHandler, txFactory TxFactory, logger *slog.Logger, opts ...ConsumerOption) (*Consumer, error) {
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
		kgo.ClientID("identity-service-consumer"),
		kgo.ConsumerGroup(cfg.ConsumerGroup),
		kgo.ConsumeTopics(cfg.Topic),
		kgo.DisableAutoCommit(),
	)

	if err != nil {
		return nil, fmt.Errorf("create kafka client: %w", err)
	}

	cons := &Consumer{
		client:         client,
		handler:        handler,
		txFactory:      txFactory,
		logger:         logger,
		observer:       NoopObserver{},
		processTimeout: cfg.ProcessTimeout,
	}

	for _, opt := range opts {
		opt(cons)
	}

	return cons, nil
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
				c.observer.EventFailed(eventTypeFromRecord(record))
				c.logger.Error(
					"kafka record processing failed",
					"topic", record.Topic,
					"partition", record.Partition,
					"offset", record.Offset,
					slog.Any("error", err),
				)

				return
			}
			c.observer.EventProcessed(eventTypeFromRecord(record))
		}
	})
}

func (c *Consumer) processRecord(ctx context.Context, record *kgo.Record) error {
	event, err := eventFromRecord(record)

	if err != nil {
		return err
	}

	var duplicate bool

	processCtx, cancel := timex.WithTimeout(ctx, c.processTimeout)
	defer cancel()

	err = c.txFactory.WithinTx(processCtx, func(stores TxStores) error {
		processed, err := stores.IdempotencyStore.WasProcessed(processCtx, event.EventID)
		if err != nil {
			return err
		}

		if processed {
			duplicate = true
			return nil
		}

		if err := c.handler.Handle(processCtx, event, stores); err != nil {
			return err
		}

		if err := stores.IdempotencyStore.MarkProcessed(processCtx, event); err != nil {
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

	contentType := headers["content_type"]
	if contentType == "" {
		return Event{}, fmt.Errorf("kafka record missing content_type header")
	}

	protoMessage := headers["proto_message"]
	if contentType == "application/x-protobuf" && protoMessage == "" {
		return Event{}, fmt.Errorf("kafka protobuf record missing proto_message header")
	}

	eventVersion := headers["event_version"]
	if eventVersion == "" {
		return Event{}, fmt.Errorf("kafka record missing event_version header")
	}

	return Event{
		EventID:       eventID,
		EventType:     eventType,
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		ContentType:   contentType,
		ProtoMessage:  protoMessage,
		EventVersion:  eventVersion,
		Payload:       record.Value,
		Topic:         record.Topic,
		Partition:     record.Partition,
		Offset:        record.Offset,
	}, nil
}

func (c *Consumer) Close() {
	c.client.Close()
}

func eventTypeFromRecord(record *kgo.Record) string {
	for _, header := range record.Headers {
		if header.Key == "event_type" {
			return string(header.Value)
		}
	}
	return "unknown"
}
