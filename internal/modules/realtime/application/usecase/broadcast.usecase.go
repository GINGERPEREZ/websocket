package usecase

import (
	"context"

	"mesaYaWs/internal/modules/realtime/application/port"
	"mesaYaWs/internal/modules/realtime/domain"
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
