package outbox

import (
	"CrudTutorialProject/internal/timex"
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

type KafkaPublisherConfig struct {
	Brokers               []string
	Topic                 string
	ProducerBatchMaxBytes int32
	ProducerLinger        time.Duration
	ProduceTimeout        time.Duration
}

type KafkaPublisher struct {
	client         *kgo.Client
	topic          string
	produceTimeout time.Duration
}

func NewKafkaPublisher(cfg KafkaPublisherConfig) (*KafkaPublisher, error) {
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers are required")
	}

	if cfg.Topic == "" {
		return nil, fmt.Errorf("kafka topic is required")
	}

	opts := []kgo.Opt{
		kgo.SeedBrokers(cfg.Brokers...),
		kgo.DefaultProduceTopic(cfg.Topic),
		kgo.ClientID("go-crud-api"),
	}

	if cfg.ProducerBatchMaxBytes > 0 {
		opts = append(opts, kgo.ProducerBatchMaxBytes(cfg.ProducerBatchMaxBytes))
	}

	if cfg.ProducerLinger > 0 {
		opts = append(opts, kgo.ProducerLinger(cfg.ProducerLinger))
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("create kafka client: %w", err)
	}

	return &KafkaPublisher{
		client:         client,
		topic:          cfg.Topic,
		produceTimeout: cfg.ProduceTimeout,
	}, nil
}

func (p *KafkaPublisher) Publish(ctx context.Context, event Event) error {
	ctx, cancel := timex.WithTimeout(ctx, p.produceTimeout)
	defer cancel()

	record := kgo.Record{
		Topic: p.topic,
		Key:   []byte(event.AggregateType + ":" + event.AggregateID),
		Value: event.Payload,
		Headers: []kgo.RecordHeader{
			{Key: "event_id", Value: []byte(strconv.FormatInt(event.ID, 10))},
			{Key: "event_type", Value: []byte(event.EventType)},
			{Key: "aggregate_type", Value: []byte(event.AggregateType)},
			{Key: "aggregate_id", Value: []byte(event.AggregateID)},
			{Key: "content_type", Value: []byte(event.ContentType)},
			{Key: "proto_message", Value: []byte(event.ProtoMessage)},
			{Key: "event_version", Value: []byte(event.EventVersion)},
		},
		Timestamp: event.CreatedAt,
	}

	result := p.client.ProduceSync(ctx, &record)

	if err := result.FirstErr(); err != nil {
		return fmt.Errorf("produce kafka record: %w", err)
	}

	return nil
}

func (p *KafkaPublisher) Close() {
	p.client.Close()
}
