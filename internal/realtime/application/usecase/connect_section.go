package usecase

import (
	"context"
	"errors"
	"log"
	"strconv"
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

	log.Printf("connect-section: validating token section=%s", input.SectionID)

	claims, err := uc.Validator.Validate(input.Token)
	if err != nil {
		log.Printf("connect-section: token validation failed section=%s err=%v", input.SectionID, err)
		return nil, err
	}
	log.Printf("connect-section: token valid section=%s subject=%s session=%s roles=%v", input.SectionID, claims.RegisteredClaims.Subject, claims.SessionID, claims.Roles)

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
			log.Printf("connect-section: unknown cache kind kind=%s section=%s", entry.kind, sectionID)
		}
	}
}

func (uc *ConnectSectionUseCase) ListRestaurants(ctx context.Context, token, sectionID string, params domain.PagedQuery) (*domain.SectionSnapshot, domain.PagedQuery, error) {
	options := params.Normalize(sectionID)
	queryKey := options.CanonicalKey()
	log.Printf("connect-section: list request section=%s query=%s", sectionID, queryKey)

	snapshot, err := uc.SnapshotFetcher.FetchSection(ctx, token, sectionID, options)
	switch {
	case errors.Is(err, port.ErrSnapshotNotFound):
		log.Printf("connect-section: list not found section=%s query=%s", sectionID, queryKey)
		uc.cache.delete(sectionID, cacheKindList, options, "")
		return nil, options, err
	case err != nil:
		log.Printf("connect-section: list fetch failed section=%s query=%s err=%v", sectionID, queryKey, err)
		if cached, ok := uc.cache.get(sectionID, cacheKindList, options, ""); ok && cached.snapshot != nil {
			log.Printf("connect-section: serving cached list section=%s query=%s fetchedAt=%s", sectionID, queryKey, cached.fetchedAt.Format(time.RFC3339Nano))
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
	log.Printf("connect-section: detail request section=%s restaurant=%s", sectionID, resource)

	snapshot, err := uc.SnapshotFetcher.FetchRestaurant(ctx, token, resource)
	switch {
	case errors.Is(err, port.ErrSnapshotNotFound):
		log.Printf("connect-section: detail not found section=%s restaurant=%s", sectionID, resource)
		uc.cache.delete(sectionID, cacheKindItem, domain.PagedQuery{}, resource)
		return nil, err
	case err != nil:
		log.Printf("connect-section: detail fetch failed section=%s restaurant=%s err=%v", sectionID, resource, err)
		if cached, ok := uc.cache.get(sectionID, cacheKindItem, domain.PagedQuery{}, resource); ok && cached.snapshot != nil {
			log.Printf("connect-section: serving cached detail section=%s restaurant=%s fetchedAt=%s", sectionID, resource, cached.fetchedAt.Format(time.RFC3339Nano))
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
		log.Printf("connect-section: refresh list not found section=%s query=%s", sectionID, queryKey)
		uc.cache.delete(sectionID, cacheKindList, options, "")
		return
	case err != nil:
		log.Printf("connect-section: refresh list failed section=%s query=%s err=%v", sectionID, queryKey, err)
		return
	default:
		uc.cache.set(sectionID, cacheKindList, options, "", entry.token, snapshot)
	}

	metadata := options.Metadata(sectionID)

	message := &domain.Message{
		Topic:      entity + ".list",
		Entity:     entity,
		Action:     "list",
		ResourceID: sectionID,
		Metadata:   metadata,
		Data:       snapshot.Payload,
		Timestamp:  time.Now().UTC(),
	}
	broadcaster.Execute(ctx, message)
	log.Printf("connect-section: refreshed list broadcast section=%s entity=%s query=%s", sectionID, entity, queryKey)
}

func (uc *ConnectSectionUseCase) refreshItem(ctx context.Context, entity, sectionID string, entry *snapshotCacheEntry, broadcaster *BroadcastUseCase) {
	snapshot, err := uc.SnapshotFetcher.FetchRestaurant(ctx, entry.token, entry.resourceID)
	switch {
	case errors.Is(err, port.ErrSnapshotNotFound):
		log.Printf("connect-section: refresh detail not found section=%s restaurant=%s", sectionID, entry.resourceID)
		uc.cache.delete(sectionID, cacheKindItem, domain.PagedQuery{}, entry.resourceID)
		return
	case err != nil:
		log.Printf("connect-section: refresh detail failed section=%s restaurant=%s err=%v", sectionID, entry.resourceID, err)
		return
	default:
		uc.cache.set(sectionID, cacheKindItem, domain.PagedQuery{}, entry.resourceID, entry.token, snapshot)
	}

	metadata := map[string]string{
		"sectionId":    sectionID,
		"restaurantId": entry.resourceID,
	}

	message := &domain.Message{
		Topic:      entity + ".detail",
		Entity:     entity,
		Action:     "detail",
		ResourceID: entry.resourceID,
		Metadata:   metadata,
		Data:       snapshot.Payload,
		Timestamp:  time.Now().UTC(),
	}
	broadcaster.Execute(ctx, message)
	log.Printf("connect-section: refreshed detail broadcast section=%s entity=%s restaurant=%s", sectionID, entity, entry.resourceID)
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
	metadata := normalized.Metadata(sectionID)
	if snapshot.RestaurantList != nil {
		metadata["itemsCount"] = strconv.Itoa(len(snapshot.RestaurantList.Items))
		if snapshot.RestaurantList.Total > 0 {
			metadata["total"] = strconv.Itoa(snapshot.RestaurantList.Total)
		}
	}
	message := &domain.Message{
		Topic:      entity + ".list",
		Entity:     entity,
		Action:     "list",
		ResourceID: sectionID,
		Metadata:   metadata,
		Data:       snapshot.Payload,
		Timestamp:  time.Now().UTC(),
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
	metadata := map[string]string{
		"sectionId":    sectionID,
		"restaurantId": restaurantID,
	}
	if snapshot.Restaurant != nil {
		if trimmed := strings.TrimSpace(snapshot.Restaurant.Name); trimmed != "" {
			metadata["restaurantName"] = trimmed
		}
	}
	message := &domain.Message{
		Topic:      entity + ".detail",
		Entity:     entity,
		Action:     "detail",
		ResourceID: restaurantID,
		Metadata:   metadata,
		Data:       snapshot.Payload,
		Timestamp:  time.Now().UTC(),
	}
	return message, nil
}
