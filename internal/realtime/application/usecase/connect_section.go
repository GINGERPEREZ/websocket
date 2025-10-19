package usecase

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"mesaYaWs/internal/realtime/application/port"
	"mesaYaWs/internal/realtime/domain"
	"mesaYaWs/internal/shared/auth"
)

type ConnectSectionInput struct {
	Token     string
	SectionID string
}

type ConnectSectionOutput struct {
	Claims   *auth.Claims
	Snapshot *domain.SectionSnapshot
}

type ConnectSectionUseCase struct {
	Validator       auth.TokenValidator
	SnapshotFetcher port.SectionSnapshotFetcher
	cache           *snapshotCache
}

var (
	ErrMissingToken   = errors.New("missing token")
	ErrMissingSection = errors.New("missing section id")
)

func NewConnectSectionUseCase(validator auth.TokenValidator, fetcher port.SectionSnapshotFetcher) *ConnectSectionUseCase {
	return &ConnectSectionUseCase{
		Validator:       validator,
		SnapshotFetcher: fetcher,
		cache:           newSnapshotCache(),
	}
}

func (uc *ConnectSectionUseCase) Execute(ctx context.Context, input ConnectSectionInput) (*ConnectSectionOutput, error) {
	if strings.TrimSpace(input.Token) == "" {
		return nil, ErrMissingToken
	}
	if strings.TrimSpace(input.SectionID) == "" {
		return nil, ErrMissingSection
	}

	slog.Info("connect-section validating token", slog.String("sectionId", input.SectionID))

	claims, err := uc.Validator.Validate(input.Token)
	if err != nil {
		slog.Warn("connect-section token validation failed", slog.String("sectionId", input.SectionID), slog.Any("error", err))
		return nil, err
	}
	slog.Info("connect-section token valid", slog.String("sectionId", input.SectionID), slog.String("subject", claims.RegisteredClaims.Subject), slog.String("sessionId", claims.SessionID), slog.Any("roles", claims.Roles))

	return &ConnectSectionOutput{Claims: claims, Snapshot: nil}, nil
}

func (uc *ConnectSectionUseCase) RefreshSectionSnapshots(ctx context.Context, entity, sectionID string, broadcaster *BroadcastUseCase) {
	entries := uc.cache.entriesForSection(sectionID)
	if len(entries) == 0 {
		return
	}
	for _, entry := range entries {
		switch entry.kind {
		case cacheKindItem:
			uc.refreshItem(ctx, entity, sectionID, entry, broadcaster)
		case cacheKindList:
			uc.refreshList(ctx, entity, sectionID, entry, broadcaster)
		default:
			slog.Warn("connect-section unknown cache kind", slog.String("kind", entry.kind), slog.String("sectionId", sectionID))
		}
	}
}

func (uc *ConnectSectionUseCase) ListRestaurants(ctx context.Context, token, sectionID string, params domain.PagedQuery) (*domain.SectionSnapshot, domain.PagedQuery, error) {
	options := params.Normalize(sectionID)
	queryKey := options.CanonicalKey()
	slog.Debug("connect-section list request", slog.String("sectionId", sectionID), slog.String("queryKey", queryKey))

	snapshot, err := uc.SnapshotFetcher.FetchSection(ctx, token, sectionID, options)
	switch {
	case errors.Is(err, port.ErrSnapshotNotFound):
		slog.Warn("connect-section list not found", slog.String("sectionId", sectionID), slog.String("queryKey", queryKey))
		uc.cache.delete(sectionID, cacheKindList, options, "")
		return nil, options, err
	case err != nil:
		slog.Error("connect-section list fetch failed", slog.String("sectionId", sectionID), slog.String("queryKey", queryKey), slog.Any("error", err))
		if cached, ok := uc.cache.get(sectionID, cacheKindList, options, ""); ok && cached.snapshot != nil {
			slog.Info("connect-section serving cached list", slog.String("sectionId", sectionID), slog.String("queryKey", queryKey), slog.Time("fetchedAt", cached.fetchedAt))
			return cached.snapshot, options, nil
		}
		return nil, options, err
	default:
		uc.cache.set(sectionID, cacheKindList, options, "", token, snapshot)
	}

	return snapshot, options, nil
}

func (uc *ConnectSectionUseCase) GetRestaurant(ctx context.Context, token, sectionID, restaurantID string) (*domain.SectionSnapshot, error) {
	resource := strings.TrimSpace(restaurantID)
	if resource == "" {
		return nil, port.ErrSnapshotNotFound
	}
	slog.Debug("connect-section detail request", slog.String("sectionId", sectionID), slog.String("restaurantId", resource))

	snapshot, err := uc.SnapshotFetcher.FetchRestaurant(ctx, token, resource)
	switch {
	case errors.Is(err, port.ErrSnapshotNotFound):
		slog.Warn("connect-section detail not found", slog.String("sectionId", sectionID), slog.String("restaurantId", resource))
		uc.cache.delete(sectionID, cacheKindItem, domain.PagedQuery{}, resource)
		return nil, err
	case err != nil:
		slog.Error("connect-section detail fetch failed", slog.String("sectionId", sectionID), slog.String("restaurantId", resource), slog.Any("error", err))
		if cached, ok := uc.cache.get(sectionID, cacheKindItem, domain.PagedQuery{}, resource); ok && cached.snapshot != nil {
			slog.Info("connect-section serving cached detail", slog.String("sectionId", sectionID), slog.String("restaurantId", resource), slog.Time("fetchedAt", cached.fetchedAt))
			return cached.snapshot, nil
		}
		return nil, err
	default:
		uc.cache.set(sectionID, cacheKindItem, domain.PagedQuery{}, resource, token, snapshot)
	}

	return snapshot, nil
}

func (uc *ConnectSectionUseCase) refreshList(ctx context.Context, entity, sectionID string, entry *snapshotCacheEntry, broadcaster *BroadcastUseCase) {
	options := entry.listOptions
	queryKey := options.CanonicalKey()
	snapshot, err := uc.SnapshotFetcher.FetchSection(ctx, entry.token, sectionID, options)
	switch {
	case errors.Is(err, port.ErrSnapshotNotFound):
		slog.Warn("connect-section refresh list not found", slog.String("sectionId", sectionID), slog.String("queryKey", queryKey))
		uc.cache.delete(sectionID, cacheKindList, options, "")
		return
	case err != nil:
		slog.Error("connect-section refresh list failed", slog.String("sectionId", sectionID), slog.String("queryKey", queryKey), slog.Any("error", err))
		return
	default:
		uc.cache.set(sectionID, cacheKindList, options, "", entry.token, snapshot)
	}

	message := domain.BuildListMessage(entity, sectionID, snapshot, options, time.Now().UTC())
	if message == nil {
		slog.Debug("connect-section refresh list skipped", slog.String("sectionId", sectionID), slog.String("queryKey", queryKey))
		return
	}
	broadcaster.Execute(ctx, message)
	slog.Info("connect-section refreshed list broadcast", slog.String("sectionId", sectionID), slog.String("entity", entity), slog.String("queryKey", queryKey))
}

func (uc *ConnectSectionUseCase) refreshItem(ctx context.Context, entity, sectionID string, entry *snapshotCacheEntry, broadcaster *BroadcastUseCase) {
	snapshot, err := uc.SnapshotFetcher.FetchRestaurant(ctx, entry.token, entry.resourceID)
	switch {
	case errors.Is(err, port.ErrSnapshotNotFound):
		slog.Warn("connect-section refresh detail not found", slog.String("sectionId", sectionID), slog.String("restaurantId", entry.resourceID))
		uc.cache.delete(sectionID, cacheKindItem, domain.PagedQuery{}, entry.resourceID)
		return
	case err != nil:
		slog.Error("connect-section refresh detail failed", slog.String("sectionId", sectionID), slog.String("restaurantId", entry.resourceID), slog.Any("error", err))
		return
	default:
		uc.cache.set(sectionID, cacheKindItem, domain.PagedQuery{}, entry.resourceID, entry.token, snapshot)
	}

	message := domain.BuildDetailMessage(entity, sectionID, entry.resourceID, snapshot, time.Now().UTC())
	if message == nil {
		slog.Debug("connect-section refresh detail skipped", slog.String("sectionId", sectionID), slog.String("restaurantId", entry.resourceID))
		return
	}
	broadcaster.Execute(ctx, message)
	slog.Info("connect-section refreshed detail broadcast", slog.String("sectionId", sectionID), slog.String("entity", entity), slog.String("restaurantId", entry.resourceID))
}

// HandleListRestaurantsCommand executes the list command end-to-end and returns a domain message ready for broadcasting.
func (uc *ConnectSectionUseCase) HandleListRestaurantsCommand(ctx context.Context, token, sectionID string, command domain.ListRestaurantsCommand, entity string) (*domain.Message, error) {
	query := domain.PagedQuery{
		Page:      command.Page,
		Limit:     command.Limit,
		Search:    command.Search,
		SortBy:    command.SortBy,
		SortOrder: command.SortOrder,
	}
	snapshot, normalized, err := uc.ListRestaurants(ctx, token, sectionID, query)
	if err != nil {
		return nil, err
	}
	message := domain.BuildListMessage(entity, sectionID, snapshot, normalized, time.Now().UTC())
	if message == nil {
		return nil, port.ErrSnapshotNotFound
	}
	return message, nil
}

// HandleGetRestaurantCommand executes the detail command end-to-end and returns a domain message ready for broadcasting.
func (uc *ConnectSectionUseCase) HandleGetRestaurantCommand(ctx context.Context, token, sectionID string, command domain.GetRestaurantCommand, entity string) (*domain.Message, error) {
	restaurantID := strings.TrimSpace(command.ID)
	if restaurantID == "" {
		return nil, port.ErrSnapshotNotFound
	}
	snapshot, err := uc.GetRestaurant(ctx, token, sectionID, restaurantID)
	if err != nil {
		return nil, err
	}
	message := domain.BuildDetailMessage(entity, sectionID, restaurantID, snapshot, time.Now().UTC())
	if message == nil {
		return nil, port.ErrSnapshotNotFound
	}
	return message, nil
}
