package port

import (
	"context"
	"errors"

	"mesaYaWs/internal/realtime/domain"
)

var (
	ErrSnapshotForbidden = errors.New("section snapshot forbidden")
	ErrSnapshotNotFound  = errors.New("section snapshot not found")
)

type SectionSnapshotFetcher interface {
	FetchSection(ctx context.Context, token, sectionID string, query domain.PagedQuery) (*domain.SectionSnapshot, error)
	FetchRestaurant(ctx context.Context, token, restaurantID string) (*domain.SectionSnapshot, error)
}
