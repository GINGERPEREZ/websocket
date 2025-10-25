package port

import (
	"context"
	"errors"

	"mesaYaWs/internal/modules/realtime/domain"
)

var (
	ErrSnapshotForbidden   = errors.New("section snapshot forbidden")
	ErrSnapshotNotFound    = errors.New("section snapshot not found")
	ErrSnapshotUnsupported = errors.New("section snapshot entity unsupported")
)

type SectionSnapshotFetcher interface {
	FetchSection(ctx context.Context, token, sectionID string, query domain.PagedQuery) (*domain.SectionSnapshot, error)
	FetchRestaurant(ctx context.Context, token, restaurantID string) (*domain.SectionSnapshot, error)
	FetchTable(ctx context.Context, token, tableID string) (*domain.SectionSnapshot, error)
	FetchReservation(ctx context.Context, token, reservationID string) (*domain.SectionSnapshot, error)
	FetchEntityList(ctx context.Context, token, entity, sectionID string, query domain.PagedQuery) (*domain.SectionSnapshot, error)
	FetchEntityDetail(ctx context.Context, token, entity, resourceID string) (*domain.SectionSnapshot, error)
}
