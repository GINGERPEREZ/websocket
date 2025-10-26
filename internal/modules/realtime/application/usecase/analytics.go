package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"time"

	"mesaYaWs/internal/modules/realtime/application/port"
	"mesaYaWs/internal/modules/realtime/domain"
	"mesaYaWs/internal/shared/auth"
)

var (
	// ErrAnalyticsMissingIdentifier indicates a required identifier was not provided.
	ErrAnalyticsMissingIdentifier = errors.New("missing analytics identifier")
)

// AnalyticsEndpointConfig describes how to resolve a REST endpoint for analytics data.
type AnalyticsEndpointConfig struct {
	Key                string
	Scope              string
	Entity             string
	PathTemplate       string
	RequiresIdentifier bool
	IdentifierParam    string
	QueryParams        []string
	RequireToken       bool
}

// BuildPath resolves the HTTP path for the configured endpoint.
func (cfg AnalyticsEndpointConfig) BuildPath(identifier string) (string, error) {
	path := strings.TrimSpace(cfg.PathTemplate)
	if path == "" {
		return "", fmt.Errorf("analytics endpoint %s missing path", cfg.Key)
	}
	if cfg.RequiresIdentifier {
		trimmed := strings.TrimSpace(identifier)
		if trimmed == "" {
			return "", ErrAnalyticsMissingIdentifier
		}
		return fmt.Sprintf(path, url.PathEscape(trimmed)), nil
	}
	return path, nil
}

// RequestFromValues builds an analytics request from HTTP query parameters.
func (cfg AnalyticsEndpointConfig) RequestFromValues(values url.Values) domain.AnalyticsRequest {
	req := domain.AnalyticsRequest{}
	if cfg.IdentifierParam != "" {
		req.Identifier = strings.TrimSpace(values.Get(cfg.IdentifierParam))
	}
	if len(cfg.QueryParams) > 0 {
		req.Query = make(map[string]string, len(cfg.QueryParams))
		for _, key := range cfg.QueryParams {
			trimmedKey := strings.TrimSpace(key)
			if trimmedKey == "" {
				continue
			}
			if value := strings.TrimSpace(values.Get(trimmedKey)); value != "" {
				req.Query[trimmedKey] = value
			}
		}
	}
	return cfg.SanitizeRequest(req)
}

// SanitizeRequest filters the provided request to the allowed parameters for the endpoint.
func (cfg AnalyticsEndpointConfig) SanitizeRequest(req domain.AnalyticsRequest) domain.AnalyticsRequest {
	sanitized := domain.AnalyticsRequest{Identifier: strings.TrimSpace(req.Identifier)}
	if len(cfg.QueryParams) == 0 {
		if len(req.Query) > 0 {
			sanitized.Query = map[string]string{}
		}
		return sanitized
	}

	query := make(map[string]string, len(cfg.QueryParams))
	for _, key := range cfg.QueryParams {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		value, ok := req.Query[trimmedKey]
		if !ok {
			continue
		}
		trimmedValue := strings.TrimSpace(value)
		if trimmedValue == "" {
			continue
		}
		query[trimmedKey] = trimmedValue
	}
	if len(query) > 0 {
		sanitized.Query = query
	}
	return sanitized
}

// AnalyticsUseCase orchestrates token validation and REST fetching for analytics websockets.
type AnalyticsUseCase struct {
	validator auth.TokenValidator
	fetcher   port.AnalyticsFetcher
	endpoints map[string]AnalyticsEndpointConfig
	mu        sync.RWMutex
	sessions  map[string]*analyticsSessionEntry
}

// AnalyticsConnectOutput captures the data needed to initialise an analytics websocket session.
type AnalyticsConnectOutput struct {
	Claims  *auth.Claims
	Message *domain.Message
	Request domain.AnalyticsRequest
	Config  AnalyticsEndpointConfig
}

// NewAnalyticsUseCase builds a new analytics use case with the default endpoint registry.
func NewAnalyticsUseCase(validator auth.TokenValidator, fetcher port.AnalyticsFetcher) *AnalyticsUseCase {
	return &AnalyticsUseCase{
		validator: validator,
		fetcher:   fetcher,
		endpoints: defaultAnalyticsEndpoints(),
		sessions:  make(map[string]*analyticsSessionEntry),
	}
}

// Endpoint retrieves the configuration for the given analytics key.
func (uc *AnalyticsUseCase) Endpoint(key string) (AnalyticsEndpointConfig, bool) {
	cfg, ok := uc.endpoints[strings.TrimSpace(key)]
	return cfg, ok
}

// Connect validates the token (when required) and fetches the initial analytics payload.
func (uc *AnalyticsUseCase) Connect(ctx context.Context, key, token string, request domain.AnalyticsRequest) (*AnalyticsConnectOutput, error) {
	cfg, ok := uc.Endpoint(key)
	if !ok {
		slog.Warn("analytics connect endpoint unsupported", slog.String("key", key))
		return nil, port.ErrAnalyticsUnsupported
	}

	sanitized := cfg.SanitizeRequest(request)
	trimmedToken := strings.TrimSpace(token)

	if cfg.RequireToken && trimmedToken == "" {
		slog.Warn("analytics connect missing token", slog.String("key", key))
		return nil, ErrMissingToken
	}

	var claims *auth.Claims
	var err error
	if trimmedToken != "" {
		claims, err = uc.validator.Validate(trimmedToken)
		if err != nil {
			slog.Warn("analytics connect token invalid", slog.String("key", key), slog.Any("error", err))
			return nil, err
		}
	}

	path, err := cfg.BuildPath(sanitized.Identifier)
	if err != nil {
		slog.Warn("analytics connect path build failed", slog.String("key", key), slog.Any("error", err))
		return nil, err
	}

	slog.Info("analytics connect fetch", slog.String("key", key), slog.String("entity", cfg.Entity), slog.Any("query", sanitized.Query))
	snapshot, err := uc.fetcher.Fetch(ctx, trimmedToken, path, sanitized.Query)
	if err != nil {
		slog.Warn("analytics connect fetch failed", slog.String("key", key), slog.Any("error", err))
		return nil, err
	}

	message := domain.BuildAnalyticsMessage(cfg.Entity, cfg.Scope, snapshot, sanitized.Clone(), time.Now().UTC())
	if message == nil {
		slog.Warn("analytics connect message nil", slog.String("key", key))
		return nil, port.ErrAnalyticsNotFound
	}

	return &AnalyticsConnectOutput{
		Claims:  claims,
		Message: message,
		Request: sanitized.Clone(),
		Config:  cfg,
	}, nil
}

// HandleCommand processes websocket commands requesting analytics refreshes.
func (uc *AnalyticsUseCase) HandleCommand(ctx context.Context, key, token string, base domain.AnalyticsRequest, command domain.AnalyticsCommand) (*domain.Message, domain.AnalyticsRequest, error) {
	cfg, ok := uc.Endpoint(key)
	if !ok {
		return nil, base, port.ErrAnalyticsUnsupported
	}

	updated := mergeAnalyticsRequest(base, command)
	sanitized := cfg.SanitizeRequest(updated)

	path, err := cfg.BuildPath(sanitized.Identifier)
	if err != nil {
		return nil, base, err
	}

	trimmedToken := strings.TrimSpace(token)
	slog.Debug("analytics command fetch", slog.String("key", key), slog.Any("query", sanitized.Query))
	snapshot, err := uc.fetcher.Fetch(ctx, trimmedToken, path, sanitized.Query)
	if err != nil {
		return nil, base, err
	}

	message := domain.BuildAnalyticsMessage(cfg.Entity, cfg.Scope, snapshot, sanitized.Clone(), time.Now().UTC())
	if message == nil {
		return nil, base, port.ErrAnalyticsNotFound
	}

	return message, sanitized.Clone(), nil
}

// AnalyticsKey composes the canonical key for an analytics websocket based on scope and entity name.
func AnalyticsKey(scope, entity string) string {
	normalizedScope := normalizeAnalyticsScope(scope)
	normalizedEntity := normalizeAnalyticsEntity(normalizedScope, entity)
	if normalizedScope == "" || normalizedEntity == "" {
		return ""
	}
	return "analytics-" + normalizedScope + "-" + normalizedEntity
}

func normalizeAnalyticsScope(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "public", "pub":
		return "public"
	case "restaurant", "rest", "owner":
		return "restaurant"
	case "admin", "administrator", "adm":
		return "admin"
	case "auth":
		return "admin"
	default:
		return strings.TrimSpace(strings.ToLower(raw))
	}
}

func normalizeAnalyticsEntity(scope, raw string) string {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	trimmed = strings.ReplaceAll(trimmed, "_", "-")

	aliases := map[string]map[string]string{
		"public": {
			"user":         "users",
			"users":        "users",
			"public-user":  "users",
			"public-users": "users",
			"dish":         "dishes",
			"dishes":       "dishes",
			"menu":         "menus",
			"menus":        "menus",
		},
		"restaurant": {
			"user":             "users",
			"users":            "users",
			"restaurant":       "users",
			"restaurant-user":  "users",
			"restaurant-users": "users",
		},
		"admin": {
			"auth":               "auth",
			"auth-user":          "auth",
			"auth-users":         "auth",
			"user":               "auth",
			"users":              "auth",
			"restaurant":         "restaurants",
			"restaurants":        "restaurants",
			"section":            "sections",
			"sections":           "sections",
			"table":              "tables",
			"tables":             "tables",
			"image":              "images",
			"images":             "images",
			"object":             "objects",
			"objects":            "objects",
			"subscription":       "subscriptions",
			"subscriptions":      "subscriptions",
			"subscription-plan":  "subscription-plans",
			"subscription-plans": "subscription-plans",
			"subscriptionplan":   "subscription-plans",
			"subscriptionplans":  "subscription-plans",
			"reservation":        "reservations",
			"reservations":       "reservations",
			"review":             "reviews",
			"reviews":            "reviews",
			"payment":            "payments",
			"payments":           "payments",
		},
	}

	if scopeAliases, ok := aliases[scope]; ok {
		if resolved, ok := scopeAliases[trimmed]; ok {
			return resolved
		}
	}
	return trimmed
}

func mergeAnalyticsRequest(base domain.AnalyticsRequest, command domain.AnalyticsCommand) domain.AnalyticsRequest {
	merged := base.Clone()
	if strings.TrimSpace(command.Identifier) != "" {
		merged.Identifier = strings.TrimSpace(command.Identifier)
	}
	if merged.Query == nil {
		merged.Query = map[string]string{}
	}
	for key, value := range command.Query {
		trimmedKey := strings.TrimSpace(key)
		trimmedValue := strings.TrimSpace(value)
		if trimmedKey == "" {
			continue
		}
		if trimmedValue == "" {
			delete(merged.Query, trimmedKey)
			continue
		}
		merged.Query[trimmedKey] = trimmedValue
	}
	return merged
}

func defaultAnalyticsEndpoints() map[string]AnalyticsEndpointConfig {
	entries := []AnalyticsEndpointConfig{
		{
			Key:          "analytics-public-users",
			Scope:        "public",
			Entity:       "analytics-public-users",
			PathTemplate: "/api/v1/public/users/analytics",
			QueryParams:  []string{"startDate"},
		},
		{
			Key:          "analytics-public-dishes",
			Scope:        "public",
			Entity:       "analytics-public-dishes",
			PathTemplate: "/api/v1/public/dishes/analytics",
			QueryParams:  []string{"startDate"},
		},
		{
			Key:          "analytics-public-menus",
			Scope:        "public",
			Entity:       "analytics-public-menus",
			PathTemplate: "/api/v1/public/menus/analytics",
			QueryParams:  []string{"startDate"},
		},
		{
			Key:                "analytics-restaurant-users",
			Scope:              "restaurant",
			Entity:             "analytics-restaurant-users",
			PathTemplate:       "/api/v1/restaurant/users/analytics/restaurant/%s",
			RequiresIdentifier: true,
			IdentifierParam:    "restaurantId",
			QueryParams:        []string{"startDate"},
			RequireToken:       true,
		},
		{
			Key:          "analytics-admin-auth",
			Scope:        "admin",
			Entity:       "analytics-admin-auth",
			PathTemplate: "/api/v1/auth/analytics",
			QueryParams:  []string{"startDate"},
			RequireToken: true,
		},
		{
			Key:          "analytics-admin-restaurants",
			Scope:        "admin",
			Entity:       "analytics-admin-restaurants",
			PathTemplate: "/api/v1/admin/restaurant/analytics",
			QueryParams:  []string{"startDate"},
			RequireToken: true,
		},
		{
			Key:          "analytics-admin-sections",
			Scope:        "admin",
			Entity:       "analytics-admin-sections",
			PathTemplate: "/api/v1/admin/section/analytics",
			QueryParams:  []string{"restaurantId", "startDate"},
			RequireToken: true,
		},
		{
			Key:          "analytics-admin-tables",
			Scope:        "admin",
			Entity:       "analytics-admin-tables",
			PathTemplate: "/api/v1/admin/table/analytics",
			QueryParams:  []string{"sectionId", "restaurantId", "startDate"},
			RequireToken: true,
		},
		{
			Key:          "analytics-admin-images",
			Scope:        "admin",
			Entity:       "analytics-admin-images",
			PathTemplate: "/api/v1/admin/image/analytics",
			QueryParams:  []string{"startDate"},
			RequireToken: true,
		},
		{
			Key:          "analytics-admin-objects",
			Scope:        "admin",
			Entity:       "analytics-admin-objects",
			PathTemplate: "/api/v1/admin/object/analytics",
			QueryParams:  []string{"startDate"},
			RequireToken: true,
		},
		{
			Key:          "analytics-admin-subscriptions",
			Scope:        "admin",
			Entity:       "analytics-admin-subscriptions",
			PathTemplate: "/api/v1/admin/subscriptions/analytics",
			QueryParams:  []string{"startDate"},
			RequireToken: true,
		},
		{
			Key:          "analytics-admin-subscription-plans",
			Scope:        "admin",
			Entity:       "analytics-admin-subscription-plans",
			PathTemplate: "/api/v1/admin/subscription-plans/analytics",
			QueryParams:  []string{"startDate"},
			RequireToken: true,
		},
		{
			Key:          "analytics-admin-reservations",
			Scope:        "admin",
			Entity:       "analytics-admin-reservations",
			PathTemplate: "/api/v1/admin/reservations/analytics",
			QueryParams:  []string{"restaurantId", "startDate"},
			RequireToken: true,
		},
		{
			Key:          "analytics-admin-reviews",
			Scope:        "admin",
			Entity:       "analytics-admin-reviews",
			PathTemplate: "/api/v1/admin/review/analytics",
			QueryParams:  []string{"restaurantId", "startDate"},
			RequireToken: true,
		},
		{
			Key:          "analytics-admin-payments",
			Scope:        "admin",
			Entity:       "analytics-admin-payments",
			PathTemplate: "/api/v1/admin/payments/analytics",
			QueryParams:  []string{"restaurantId", "startDate"},
			RequireToken: true,
		},
	}

	registry := make(map[string]AnalyticsEndpointConfig, len(entries))
	for _, entry := range entries {
		registry[entry.Key] = entry
	}
	return registry
}

type analyticsSessionEntry struct {
	key     string
	token   string
	request domain.AnalyticsRequest
}

// RegisterSession stores the analytics request associated with an active websocket session.
func (uc *AnalyticsUseCase) RegisterSession(sessionID, key, token string, request domain.AnalyticsRequest) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return
	}
	cfg, ok := uc.Endpoint(key)
	if !ok {
		return
	}
	sanitized := cfg.SanitizeRequest(request).Clone()
	uc.mu.Lock()
	uc.sessions[sessionID] = &analyticsSessionEntry{
		key:     cfg.Key,
		token:   strings.TrimSpace(token),
		request: sanitized,
	}
	uc.mu.Unlock()
}

// UpdateSession updates the request/token stored for a websocket session.
func (uc *AnalyticsUseCase) UpdateSession(sessionID, key, token string, request domain.AnalyticsRequest) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return
	}
	cfg, ok := uc.Endpoint(key)
	if !ok {
		return
	}
	sanitized := cfg.SanitizeRequest(request).Clone()
	uc.mu.Lock()
	if entry, ok := uc.sessions[sessionID]; ok {
		entry.key = cfg.Key
		entry.token = strings.TrimSpace(token)
		entry.request = sanitized
	} else {
		uc.sessions[sessionID] = &analyticsSessionEntry{
			key:     cfg.Key,
			token:   strings.TrimSpace(token),
			request: sanitized,
		}
	}
	uc.mu.Unlock()
}

// UnregisterSession removes the stored state for a websocket session.
func (uc *AnalyticsUseCase) UnregisterSession(sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return
	}
	uc.mu.Lock()
	delete(uc.sessions, sessionID)
	uc.mu.Unlock()
}

// RefreshByEntity refreshes analytics dashboards that depend on the provided entity changes.
func (uc *AnalyticsUseCase) RefreshByEntity(ctx context.Context, entity string, broadcaster *BroadcastUseCase) {
	if broadcaster == nil {
		return
	}
	normalized := strings.ToLower(strings.TrimSpace(entity))
	if normalized == "" {
		return
	}
	keys := analyticsDependencies[normalized]
	for _, key := range keys {
		uc.refreshByKey(ctx, key, broadcaster)
	}
}

// RefreshAll refreshes all tracked analytics sessions.
func (uc *AnalyticsUseCase) RefreshAll(ctx context.Context, broadcaster *BroadcastUseCase) {
	if broadcaster == nil {
		return
	}
	uc.mu.RLock()
	snapshot := make(map[string]*analyticsSessionEntry, len(uc.sessions))
	for sessionID, entry := range uc.sessions {
		snapshot[sessionID] = &analyticsSessionEntry{
			key:     entry.key,
			token:   entry.token,
			request: entry.request.Clone(),
		}
	}
	uc.mu.RUnlock()

	for sessionID, entry := range snapshot {
		uc.refreshSession(ctx, sessionID, entry, broadcaster)
	}
}

func (uc *AnalyticsUseCase) refreshByKey(ctx context.Context, key string, broadcaster *BroadcastUseCase) {
	uc.mu.RLock()
	snapshot := make(map[string]*analyticsSessionEntry)
	for sessionID, entry := range uc.sessions {
		if strings.EqualFold(entry.key, key) {
			snapshot[sessionID] = &analyticsSessionEntry{
				key:     entry.key,
				token:   entry.token,
				request: entry.request.Clone(),
			}
		}
	}
	uc.mu.RUnlock()

	if len(snapshot) == 0 {
		return
	}

	for sessionID, entry := range snapshot {
		uc.refreshSession(ctx, sessionID, entry, broadcaster)
	}
}

func (uc *AnalyticsUseCase) refreshSession(ctx context.Context, sessionID string, entry *analyticsSessionEntry, broadcaster *BroadcastUseCase) {
	cfg, ok := uc.Endpoint(entry.key)
	if !ok {
		return
	}

	sanitized := cfg.SanitizeRequest(entry.request)
	path, err := cfg.BuildPath(sanitized.Identifier)
	if err != nil {
		slog.Warn("analytics refresh path build failed", slog.String("key", cfg.Key), slog.String("sessionId", sessionID), slog.Any("error", err))
		return
	}

	snapshot, err := uc.fetcher.Fetch(ctx, entry.token, path, sanitized.Query)
	if err != nil {
		slog.Warn("analytics refresh fetch failed", slog.String("key", cfg.Key), slog.String("sessionId", sessionID), slog.Any("error", err))
		uc.emitAnalyticsError(ctx, broadcaster, cfg, sessionID, sanitized, err)
		return
	}

	message := domain.BuildAnalyticsMessage(cfg.Entity, cfg.Scope, snapshot, sanitized.Clone(), time.Now().UTC())
	if message == nil {
		return
	}
	if message.Metadata == nil {
		message.Metadata = map[string]string{}
	}
	message.Metadata["sessionId"] = sessionID
	message.Metadata["analyticsKey"] = cfg.Key

	broadcaster.Execute(ctx, message)

	uc.mu.Lock()
	if stored, ok := uc.sessions[sessionID]; ok {
		stored.request = sanitized.Clone()
		stored.key = cfg.Key
	}
	uc.mu.Unlock()
}

func (uc *AnalyticsUseCase) emitAnalyticsError(ctx context.Context, broadcaster *BroadcastUseCase, cfg AnalyticsEndpointConfig, sessionID string, request domain.AnalyticsRequest, err error) {
	if broadcaster == nil {
		return
	}

	metadata := map[string]string{
		"scope":        cfg.Scope,
		"analyticsKey": cfg.Key,
		"sessionId":    sessionID,
		"action":       "refresh",
		"reason":       err.Error(),
	}
	if trimmed := strings.TrimSpace(request.Identifier); trimmed != "" {
		metadata["identifier"] = trimmed
	}

	message := &domain.Message{
		Topic:    domain.ErrorTopic(cfg.Entity),
		Entity:   cfg.Entity,
		Action:   domain.ActionError,
		Metadata: metadata,
		Data: map[string]string{
			"error": err.Error(),
		},
		Timestamp: time.Now().UTC(),
	}
	broadcaster.Execute(ctx, message)
}

var analyticsDependencies = map[string][]string{
	"restaurants": {
		"analytics-admin-restaurants",
		"analytics-admin-sections",
		"analytics-admin-tables",
		"analytics-admin-payments",
	},
	"sections": {
		"analytics-admin-sections",
		"analytics-admin-tables",
	},
	"tables": {
		"analytics-admin-tables",
		"analytics-admin-payments",
	},
	"images": {
		"analytics-admin-images",
	},
	"objects": {
		"analytics-admin-objects",
		"analytics-admin-sections",
	},
	"subscriptions": {
		"analytics-admin-subscriptions",
		"analytics-admin-subscription-plans",
		"analytics-admin-payments",
	},
	"subscription-plans": {
		"analytics-admin-subscription-plans",
	},
	"reservations": {
		"analytics-admin-reservations",
		"analytics-restaurant-users",
	},
	"reviews": {
		"analytics-admin-reviews",
	},
	"payments": {
		"analytics-admin-payments",
	},
	"auth-users": {
		"analytics-admin-auth",
		"analytics-public-users",
		"analytics-restaurant-users",
	},
	"menus": {
		"analytics-public-menus",
	},
	"dishes": {
		"analytics-public-dishes",
	},
	"restaurants-public": {
		"analytics-public-users",
		"analytics-public-menus",
		"analytics-public-dishes",
	},
}
