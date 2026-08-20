package kafkaconsumer

import (
	"context"
	"fmt"
)

type Router struct {
	handlers map[string]EventHandler
}

func NewRouter() *Router {
	return &Router{
		handlers: make(map[string]EventHandler),
	}
}

func (r *Router) Register(eventType string, handler EventHandler) {
	r.handlers[eventType] = handler
}

func (r *Router) Handle(ctx context.Context, event Event) error {
	handler, ok := r.handlers[event.EventType]

	if !ok {
		return fmt.Errorf("no handler registered for event type %s", event.EventType)
	}

	return handler.Handle(ctx, event)
}
