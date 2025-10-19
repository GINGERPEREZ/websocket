package port

import (
	"context"
	"errors"
	"strings"

	"mesaYaWs/internal/realtime/domain"
)

var (
	ErrSnapshotForbidden = errors.New("section snapshot forbidden")
	ErrSnapshotNotFound  = errors.New("section snapshot not found")
)

type SectionSnapshotFetcher interface {
	FetchSection(ctx context.Context, token, sectionID string, options SectionListOptions) (*domain.SectionSnapshot, error)
	FetchRestaurant(ctx context.Context, token, restaurantID string) (*domain.SectionSnapshot, error)
}

type SectionListOptions struct {
	Page      int
	Limit     int
	Search    string
	SortBy    string
	SortOrder string
}

func NormalizeSectionListOptions(sectionID string, opts SectionListOptions) SectionListOptions {
	normalized := opts
	if normalized.Page <= 0 {
		normalized.Page = 1
	}
	if normalized.Limit <= 0 {
		normalized.Limit = 20
	}
	if normalized.Limit > 100 {
		normalized.Limit = 100
	}

	normalized.Search = strings.TrimSpace(normalized.Search)
	if normalized.Search == "" {
		normalized.Search = strings.TrimSpace(sectionID)
	}

	normalized.SortBy = strings.TrimSpace(normalized.SortBy)
	normalized.SortOrder = strings.ToUpper(strings.TrimSpace(normalized.SortOrder))

	return normalized
}
