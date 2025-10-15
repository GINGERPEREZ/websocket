package handler

import (
	"context"
	"strings"

	"mesaYaWs/internal/realtime/application/port"
	"mesaYaWs/internal/realtime/application/usecase"
	"mesaYaWs/internal/realtime/domain"
)

// EntityStreamHandler reenvía eventos de un tópico Kafka asociado a una entidad a los clientes WebSocket.
// Permite filtrar acciones permitidas para evitar ruido innecesario.
type EntityStreamHandler struct {
	kafkaTopic     string
	allowedActions map[string]struct{}
	useCase        *usecase.BroadcastUseCase
}

func NewEntityStreamHandler(kafkaTopic string, allowedActions []string, uc *usecase.BroadcastUseCase) *EntityStreamHandler {
	actionSet := make(map[string]struct{}, len(allowedActions))
	for _, a := range allowedActions {
		if v := strings.TrimSpace(strings.ToLower(a)); v != "" {
			actionSet[v] = struct{}{}
		}
	}
	return &EntityStreamHandler{kafkaTopic: kafkaTopic, allowedActions: actionSet, useCase: uc}
}

func (h *EntityStreamHandler) Topic() string { return h.kafkaTopic }

func (h *EntityStreamHandler) Handle(ctx context.Context, msg *domain.Message) error {
	if len(h.allowedActions) > 0 {
		if _, ok := h.allowedActions[strings.ToLower(msg.Action)]; !ok {
			return nil
		}
	}
	if msg.Topic == "" && msg.Entity != "" && msg.Action != "" {
		msg.Topic = msg.Entity + "." + msg.Action
	}
	h.useCase.Execute(ctx, msg)
	return nil
}

var _ port.TopicHandler = (*EntityStreamHandler)(nil)
