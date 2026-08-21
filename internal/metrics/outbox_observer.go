package metrics

type OutboxObserver struct {
	metrics *Metrics
}

func NewOutboxObserver(metrics *Metrics) *OutboxObserver {
	return &OutboxObserver{
		metrics: metrics,
	}
}

func (o *OutboxObserver) EventProcessed(eventType string) {
	o.metrics.OutboxProcessedTotal.WithLabelValues(eventType).Inc()
}

func (o *OutboxObserver) EventFailed(eventType string) {
	o.metrics.OutboxFailedTotal.WithLabelValues(eventType).Inc()
}
