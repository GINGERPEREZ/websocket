package infrastructure

import (
	"context"
	"mesaYaWs/internal/realtime/application/port"
	"mesaYaWs/internal/realtime/domain"
)

type HandlerRegistry struct {
	handlers map[string]port.TopicHandler
}

func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{handlers: make(map[string]port.TopicHandler)}
}

func (r *HandlerRegistry) Register(h port.TopicHandler) {
	r.handlers[h.Topic()] = h
}

func (r *HandlerRegistry) Dispatch(ctx context.Context, msg *domain.Message) error {
	if handler, ok := r.handlers[msg.Topic]; ok {
		return handler.Handle(ctx, msg)
	}
	return nil
}
