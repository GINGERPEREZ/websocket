package usecase

import (
	"context"
	"mesaYaWs/internal/modules/restaurants/application/port"
	"mesaYaWs/internal/modules/restaurants/domain"
)

type BroadcastUseCase struct {
	broadcaster port.Broadcaster
}

func NewBroadcastUseCase(b port.Broadcaster) *BroadcastUseCase {
	return &BroadcastUseCase{broadcaster: b}
}

func (uc *BroadcastUseCase) Execute(ctx context.Context, msg *domain.Message) {
	uc.broadcaster.Broadcast(ctx, msg)
}
