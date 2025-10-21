package handler

import (
    "context"

    "mesaYaWs/internal/modules/realtime/application/port"
    "mesaYaWs/internal/modules/realtime/application/usecase"
    "mesaYaWs/internal/modules/realtime/domain"
)

// UserCreatedHandler maneja eventos user.created y delega al usecase de broadcast.
type UserCreatedHandler struct {
    UseCase *usecase.BroadcastUseCase
}

func (h *UserCreatedHandler) Topic() string { return "user.created" }

func (h *UserCreatedHandler) Handle(ctx context.Context, msg *domain.Message) error {
    h.UseCase.Execute(ctx, msg)
    return nil
}

var _ port.TopicHandler = (*UserCreatedHandler)(nil)
