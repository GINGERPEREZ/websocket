package handler

import (
	"context"
	"log/slog"
	"strings"

	"mesaYaWs/internal/realtime/application/port"
	"mesaYaWs/internal/realtime/application/usecase"
	"mesaYaWs/internal/realtime/domain"
)

// EntityStreamHandler reenvía eventos de un tópico Kafka asociado a una entidad a los clientes WebSocket.
// Permite filtrar acciones permitidas para evitar ruido innecesario.
type EntityStreamHandler struct {
	entity         string
	kafkaTopic     string
	allowedActions map[string]struct{}
	broadcastUC    *usecase.BroadcastUseCase
	connectUC      *usecase.ConnectSectionUseCase
}

func NewEntityStreamHandler(entity, kafkaTopic string, allowedActions []string, broadcastUC *usecase.BroadcastUseCase, connectUC *usecase.ConnectSectionUseCase) *EntityStreamHandler {
	actionSet := make(map[string]struct{}, len(allowedActions))
	for _, a := range allowedActions {
		if v := strings.TrimSpace(strings.ToLower(a)); v != "" {
			actionSet[v] = struct{}{}
		}
	}
	return &EntityStreamHandler{
		entity:         strings.TrimSpace(entity),
		kafkaTopic:     kafkaTopic,
		allowedActions: actionSet,
		broadcastUC:    broadcastUC,
		connectUC:      connectUC,
	}
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
	h.broadcastUC.Execute(ctx, msg)
	h.refreshSnapshot(ctx, msg)
	return nil
}

func (h *EntityStreamHandler) refreshSnapshot(ctx context.Context, msg *domain.Message) {
	if h.connectUC == nil {
		return
	}
	if strings.EqualFold(msg.Action, "snapshot") {
		return
	}
	sectionID := strings.TrimSpace(msg.ResourceID)
	if sectionID == "" && msg.Metadata != nil {
		sectionID = strings.TrimSpace(msg.Metadata["sectionId"])
	}
	entityName := h.entity
	if entityName == "" {
		entityName = strings.TrimSpace(msg.Entity)
	}
	if entityName == "" {
		return
	}
	if sectionID != "" {
		slog.Info("entity-stream refresh", slog.String("entity", entityName), slog.String("action", msg.Action), slog.String("sectionId", sectionID))
		h.connectUC.RefreshSectionSnapshots(ctx, entityName, sectionID, h.broadcastUC)
		return
	}
	slog.Info("entity-stream refresh all sections", slog.String("entity", entityName), slog.String("action", msg.Action))
	h.connectUC.RefreshAllSections(ctx, entityName, h.broadcastUC)
}

var _ port.TopicHandler = (*EntityStreamHandler)(nil)
