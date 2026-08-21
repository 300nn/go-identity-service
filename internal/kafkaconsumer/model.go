package kafkaconsumer

type Event struct {
	EventID       string
	EventType     string
	AggregateType string
	AggregateID   string

	ContentType  string
	ProtoMessage string
	EventVersion string
	Payload      []byte

	Topic     string
	Partition int32
	Offset    int64
}
