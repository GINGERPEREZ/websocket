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
	FetchEntityList(ctx context.Context, token, entity, sectionID string, query domain.PagedQuery) (*domain.SectionSnapshot, error)
	FetchEntityDetail(ctx context.Context, token, entity, resourceID string) (*domain.SectionSnapshot, error)
}
