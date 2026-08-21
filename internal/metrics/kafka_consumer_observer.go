package metrics

type KafkaConsumerObserver struct {
	metrics *Metrics
}

func NewKafkaConsumerObserver(metrics *Metrics) *KafkaConsumerObserver {
	return &KafkaConsumerObserver{
		metrics: metrics,
	}
}

func (o *KafkaConsumerObserver) EventProcessed(eventType string) {
	o.metrics.KafkaConsumerProcessedTotal.WithLabelValues(eventType).Inc()
}

func (o *KafkaConsumerObserver) EventFailed(eventType string) {
	o.metrics.KafkaConsumerFailedTotal.WithLabelValues(eventType).Inc()
}
