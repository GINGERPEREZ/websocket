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

type SnapshotAudience string

const (
	SnapshotAudienceAdmin SnapshotAudience = "admin"
	SnapshotAudienceOwner SnapshotAudience = "owner"
	SnapshotAudienceUser  SnapshotAudience = "user"
)

type SnapshotContext struct {
	SectionID string
	Audience  SnapshotAudience
}

type SectionSnapshotFetcher interface {
	FetchEntityList(ctx context.Context, token, entity string, snapshotCtx SnapshotContext, query domain.PagedQuery) (*domain.SectionSnapshot, error)
	FetchEntityDetail(ctx context.Context, token, entity string, snapshotCtx SnapshotContext, resourceID string) (*domain.SectionSnapshot, error)
}
