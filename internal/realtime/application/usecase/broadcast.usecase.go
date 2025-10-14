package usecase

import (
	"context"
	"mesaYaWs/internal/realtime/domain"
)

type BroadcastUseCase struct {
	broadcaster Broadcaster
}

func NewBroadcastUseCase(b Broadcaster) *BroadcastUseCase {
	return &BroadcastUseCase{broadcaster: b}
}

func (uc *BroadcastUseCase) Execute(ctx context.Context, msg *domain.Message) {
	uc.broadcaster.Broadcast(ctx, msg)
}
