package port

import (
	"context"
	"errors"
	"net/url"

	"mesaYaWs/internal/realtime/domain"
)

var (
	ErrSnapshotForbidden = errors.New("section snapshot forbidden")
	ErrSnapshotNotFound  = errors.New("section snapshot not found")
)

type SectionSnapshotFetcher interface {
	FetchSection(ctx context.Context, token, sectionID string, query url.Values) (*domain.SectionSnapshot, error)
}
