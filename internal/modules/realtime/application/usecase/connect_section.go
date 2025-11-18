package usecase

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"mesaYaWs/internal/modules/realtime/application/port"
	"mesaYaWs/internal/modules/realtime/domain"
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
		if entity != "" && !strings.EqualFold(entry.scope, entity) {
			continue
		}
		switch entry.kind {
		case cacheKindItem:
			uc.refreshItem(ctx, entry.scope, sectionID, entry, broadcaster)
		case cacheKindList:
			uc.refreshList(ctx, entry.scope, sectionID, entry, broadcaster)
		default:
			slog.Warn("connect-section unknown cache kind", slog.String("kind", entry.kind), slog.String("sectionId", sectionID))
		}
	}
}

func (uc *ConnectSectionUseCase) RefreshAllSections(ctx context.Context, entity string, broadcaster *BroadcastUseCase) {
	for _, sectionID := range uc.cache.sectionIDs() {
		uc.RefreshSectionSnapshots(ctx, entity, sectionID, broadcaster)
	}
}

func (uc *ConnectSectionUseCase) listScope(ctx context.Context, scope, token string, snapshotCtx port.SnapshotContext, params domain.PagedQuery) (*domain.SectionSnapshot, domain.PagedQuery, error) {
	sectionID := strings.TrimSpace(snapshotCtx.SectionID)
	options := params.Normalize("")
	queryKey := options.CanonicalKey()
	if cached, ok := uc.cache.get(sectionID, scope, cacheKindList, options, "", snapshotCtx.Audience); ok && cached.snapshot != nil {
		slog.Debug("connect-section list served from cache", slog.String("sectionId", sectionID), slog.String("scope", scope), slog.String("queryKey", queryKey), slog.Time("fetchedAt", cached.fetchedAt))
		return cached.snapshot, options, nil
	}
	slog.Debug("connect-section list request", slog.String("sectionId", sectionID), slog.String("scope", scope), slog.String("queryKey", queryKey))

	snapshot, err := uc.SnapshotFetcher.FetchEntityList(ctx, token, scope, snapshotCtx, options)
	switch {
	case errors.Is(err, port.ErrSnapshotNotFound):
		slog.Warn("connect-section list not found", slog.String("sectionId", sectionID), slog.String("scope", scope), slog.String("queryKey", queryKey))
		uc.cache.delete(sectionID, scope, cacheKindList, options, "", snapshotCtx.Audience)
		return nil, options, err
	case errors.Is(err, port.ErrSnapshotUnsupported):
		slog.Warn("connect-section list unsupported", slog.String("sectionId", sectionID), slog.String("scope", scope))
		return nil, options, err
	case err != nil:
		slog.Error("connect-section list fetch failed", slog.String("sectionId", sectionID), slog.String("scope", scope), slog.String("queryKey", queryKey), slog.Any("error", err))
		if cached, ok := uc.cache.get(sectionID, scope, cacheKindList, options, "", snapshotCtx.Audience); ok && cached.snapshot != nil {
			slog.Info("connect-section serving cached list", slog.String("sectionId", sectionID), slog.String("scope", scope), slog.String("queryKey", queryKey), slog.Time("fetchedAt", cached.fetchedAt))
			return cached.snapshot, options, nil
		}
		return nil, options, err
	default:
		uc.cache.set(sectionID, scope, cacheKindList, options, "", token, snapshotCtx.Audience, snapshot)
	}

	return snapshot, options, nil
}

func (uc *ConnectSectionUseCase) getScope(ctx context.Context, scope, token string, snapshotCtx port.SnapshotContext, resourceID string, fetch func(context.Context, string, port.SnapshotContext, string) (*domain.SectionSnapshot, error)) (*domain.SectionSnapshot, error) {
	sectionID := strings.TrimSpace(snapshotCtx.SectionID)
	resource := strings.TrimSpace(resourceID)
	if resource == "" {
		return nil, port.ErrSnapshotNotFound
	}
	if cached, ok := uc.cache.get(sectionID, scope, cacheKindItem, domain.PagedQuery{}, resource, snapshotCtx.Audience); ok && cached.snapshot != nil {
		slog.Debug("connect-section detail served from cache", slog.String("sectionId", sectionID), slog.String("scope", scope), slog.String("resourceId", resource), slog.Time("fetchedAt", cached.fetchedAt))
		return cached.snapshot, nil
	}
	slog.Debug("connect-section detail request", slog.String("sectionId", sectionID), slog.String("scope", scope), slog.String("resourceId", resource))

	snapshot, err := fetch(ctx, token, snapshotCtx, resource)
	switch {
	case errors.Is(err, port.ErrSnapshotNotFound):
		slog.Warn("connect-section detail not found", slog.String("sectionId", sectionID), slog.String("scope", scope), slog.String("resourceId", resource))
		uc.cache.delete(sectionID, scope, cacheKindItem, domain.PagedQuery{}, resource, snapshotCtx.Audience)
		return nil, err
	case errors.Is(err, port.ErrSnapshotUnsupported):
		slog.Warn("connect-section detail unsupported", slog.String("sectionId", sectionID), slog.String("scope", scope), slog.String("resourceId", resource))
		return nil, err
	case err != nil:
		slog.Error("connect-section detail fetch failed", slog.String("sectionId", sectionID), slog.String("scope", scope), slog.String("resourceId", resource), slog.Any("error", err))
		if cached, ok := uc.cache.get(sectionID, scope, cacheKindItem, domain.PagedQuery{}, resource, snapshotCtx.Audience); ok && cached.snapshot != nil {
			slog.Info("connect-section serving cached detail", slog.String("sectionId", sectionID), slog.String("scope", scope), slog.String("resourceId", resource), slog.Time("fetchedAt", cached.fetchedAt))
			return cached.snapshot, nil
		}
		return nil, err
	default:
		uc.cache.set(sectionID, scope, cacheKindItem, domain.PagedQuery{}, resource, token, snapshotCtx.Audience, snapshot)
	}

	return snapshot, nil
}

func (uc *ConnectSectionUseCase) refreshList(ctx context.Context, scope, sectionID string, entry *snapshotCacheEntry, broadcaster *BroadcastUseCase) {
	options := entry.listOptions
	queryKey := options.CanonicalKey()
	snapshotCtx := port.SnapshotContext{SectionID: sectionID, Audience: entry.audience}
	snapshot, err := uc.SnapshotFetcher.FetchEntityList(ctx, entry.token, scope, snapshotCtx, options)
	switch {
	case errors.Is(err, port.ErrSnapshotNotFound):
		slog.Warn("connect-section refresh list not found", slog.String("sectionId", sectionID), slog.String("scope", scope), slog.String("queryKey", queryKey))
		uc.cache.delete(sectionID, scope, cacheKindList, options, "", entry.audience)
		return
	case errors.Is(err, port.ErrSnapshotUnsupported):
		slog.Warn("connect-section refresh list unsupported", slog.String("sectionId", sectionID), slog.String("scope", scope), slog.String("queryKey", queryKey))
		return
	case err != nil:
		slog.Error("connect-section refresh list failed", slog.String("sectionId", sectionID), slog.String("scope", scope), slog.String("queryKey", queryKey), slog.Any("error", err))
		return
	default:
		uc.cache.set(sectionID, scope, cacheKindList, options, "", entry.token, entry.audience, snapshot)
	}

	message := domain.BuildListMessage(scope, sectionID, snapshot, options, time.Now().UTC(), domain.Metadata{"origin": "refresh"})
	if message == nil {
		slog.Debug("connect-section refresh list skipped", slog.String("sectionId", sectionID), slog.String("scope", scope), slog.String("queryKey", queryKey))
		return
	}
	broadcaster.Execute(ctx, message)
	slog.Info("connect-section refreshed list broadcast", slog.String("sectionId", sectionID), slog.String("scope", scope), slog.String("queryKey", queryKey))
}

func (uc *ConnectSectionUseCase) refreshItem(ctx context.Context, scope, sectionID string, entry *snapshotCacheEntry, broadcaster *BroadcastUseCase) {
	var (
		snapshot *domain.SectionSnapshot
		err      error
	)
	snapshotCtx := port.SnapshotContext{SectionID: sectionID, Audience: entry.audience}
	snapshot, err = uc.SnapshotFetcher.FetchEntityDetail(ctx, entry.token, scope, snapshotCtx, entry.resourceID)
	switch {
	case errors.Is(err, port.ErrSnapshotNotFound):
		slog.Warn("connect-section refresh detail not found", slog.String("sectionId", sectionID), slog.String("scope", scope), slog.String("resourceId", entry.resourceID))
		uc.cache.delete(sectionID, scope, cacheKindItem, domain.PagedQuery{}, entry.resourceID, entry.audience)
		return
	case errors.Is(err, port.ErrSnapshotUnsupported):
		slog.Warn("connect-section refresh detail unsupported", slog.String("sectionId", sectionID), slog.String("scope", scope), slog.String("resourceId", entry.resourceID))
		return
	case err != nil:
		slog.Error("connect-section refresh detail failed", slog.String("sectionId", sectionID), slog.String("scope", scope), slog.String("resourceId", entry.resourceID), slog.Any("error", err))
		return
	default:
		uc.cache.set(sectionID, scope, cacheKindItem, domain.PagedQuery{}, entry.resourceID, entry.token, entry.audience, snapshot)
	}

	message := domain.BuildDetailMessage(scope, sectionID, entry.resourceID, snapshot, time.Now().UTC(), domain.Metadata{"origin": "refresh"})
	if message == nil {
		slog.Debug("connect-section refresh detail skipped", slog.String("sectionId", sectionID), slog.String("scope", scope), slog.String("resourceId", entry.resourceID))
		return
	}
	broadcaster.Execute(ctx, message)
	slog.Info("connect-section refreshed detail broadcast", slog.String("sectionId", sectionID), slog.String("scope", scope), slog.String("resourceId", entry.resourceID))
}

func (uc *ConnectSectionUseCase) HandleListEntityCommand(ctx context.Context, token string, snapshotCtx port.SnapshotContext, command domain.ListEntityCommand, entity string) (*domain.Message, error) {
	return uc.handleListCommand(ctx, token, snapshotCtx, entity, newPagedQuery(command.Page, command.Limit, command.Search, command.SortBy, command.SortOrder, command.Filters))
}

func (uc *ConnectSectionUseCase) HandleGetEntityCommand(ctx context.Context, token string, snapshotCtx port.SnapshotContext, command domain.GetEntityCommand, entity string) (*domain.Message, error) {
	return uc.handleDetailCommand(ctx, token, snapshotCtx, entity, command.ID)
}

func (uc *ConnectSectionUseCase) handleListCommand(
	ctx context.Context,
	token string,
	snapshotCtx port.SnapshotContext,
	entity string,
	query domain.PagedQuery,
) (*domain.Message, error) {
	snapshot, normalized, err := uc.listScope(ctx, entity, token, snapshotCtx, query)
	if err != nil {
		return nil, err
	}
	message := domain.BuildListMessage(entity, snapshotCtx.SectionID, snapshot, normalized, time.Now().UTC(), domain.Metadata{"origin": "request"})
	if message == nil {
		return nil, port.ErrSnapshotNotFound
	}
	return message, nil
}

func (uc *ConnectSectionUseCase) handleDetailCommand(
	ctx context.Context,
	token string,
	snapshotCtx port.SnapshotContext,
	entity, resourceID string,
) (*domain.Message, error) {
	if strings.TrimSpace(resourceID) == "" {
		return nil, port.ErrSnapshotNotFound
	}
	snapshot, err := uc.getScope(ctx, entity, token, snapshotCtx, resourceID, func(c context.Context, t string, ctxSnapshot port.SnapshotContext, resource string) (*domain.SectionSnapshot, error) {
		return uc.SnapshotFetcher.FetchEntityDetail(c, t, entity, ctxSnapshot, resource)
	})
	if err != nil {
		return nil, err
	}
	message := domain.BuildDetailMessage(entity, snapshotCtx.SectionID, resourceID, snapshot, time.Now().UTC(), domain.Metadata{"origin": "request"})
	if message == nil {
		return nil, port.ErrSnapshotNotFound
	}
	return message, nil
}

func newPagedQuery(page, limit int, search, sortBy, sortOrder string, filters map[string]string) domain.PagedQuery {
	return domain.PagedQuery{
		Page:      page,
		Limit:     limit,
		Search:    search,
		SortBy:    sortBy,
		SortOrder: sortOrder,
		Filters:   filters,
	}
}
