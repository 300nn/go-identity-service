package metrics

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec

	GRPCRequestsTotal   *prometheus.CounterVec
	GRPCRequestDuration *prometheus.HistogramVec

	OutboxProcessedTotal *prometheus.CounterVec
	OutboxFailedTotal    *prometheus.CounterVec

	KafkaConsumerProcessedTotal *prometheus.CounterVec
	KafkaConsumerFailedTotal    *prometheus.CounterVec

	RedisCacheHitsTotal   *prometheus.CounterVec
	RedisCacheMissesTotal *prometheus.CounterVec
}

func New(registry *prometheus.Registry) *Metrics {
	m := &Metrics{
		HTTPRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests.",
			},
			[]string{"method", "route", "status"},
		),
		HTTPRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "route", "status"},
		),

		GRPCRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "grpc_requests_total",
				Help: "Total number of gRPC requests.",
			},
			[]string{"method", "status"},
		),
		GRPCRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "grpc_request_duration_seconds",
				Help:    "gRPC request duration in seconds.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "status"},
		),

		OutboxProcessedTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "outbox_events_processed_total",
				Help: "Total number of successfully processed outbox events.",
			},
			[]string{"event_type"},
		),
		OutboxFailedTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "outbox_events_failed_total",
				Help: "Total number of failed outbox event processing attempts.",
			},
			[]string{"event_type"},
		),

		KafkaConsumerProcessedTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kafka_consumer_events_processed_total",
				Help: "Total number of successfully processed Kafka consumer events.",
			},
			[]string{"event_type"},
		),
		KafkaConsumerFailedTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kafka_consumer_events_failed_total",
				Help: "Total number of failed Kafka consumer event processing attempts.",
			},
			[]string{"event_type"},
		),

		RedisCacheHitsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "redis_cache_hits_total",
				Help: "Total number of Redis cache hits.",
			},
			[]string{"cache"},
		),
		RedisCacheMissesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "redis_cache_misses_total",
				Help: "Total number of Redis cache misses.",
			},
			[]string{"cache"},
		),
	}

	registry.MustRegister(
		m.HTTPRequestsTotal,
		m.HTTPRequestDuration,
		m.GRPCRequestsTotal,
		m.GRPCRequestDuration,
		m.OutboxProcessedTotal,
		m.OutboxFailedTotal,
		m.KafkaConsumerProcessedTotal,
		m.KafkaConsumerFailedTotal,
		m.RedisCacheHitsTotal,
		m.RedisCacheMissesTotal,
	)

	return m
}
