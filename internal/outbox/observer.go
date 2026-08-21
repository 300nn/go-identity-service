package outbox

type Observer interface {
	EventProcessed(eventType string)
	EventFailed(eventType string)
}

type NoopObserver struct{}

func (NoopObserver) EventProcessed(eventType string) {}
func (NoopObserver) EventFailed(eventType string)    {}
