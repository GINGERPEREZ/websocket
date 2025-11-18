package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"mesaYaWs/internal/modules/realtime/application/port"
	"mesaYaWs/internal/modules/realtime/domain"
	"mesaYaWs/internal/shared/normalization"
)

// SectionSnapshotHTTPClient implements SectionSnapshotFetcher using the REST API described in swagger.json.
type SectionSnapshotHTTPClient struct {
	rest    *RESTClient
	timeout time.Duration
}

type pathBuilder func(string) (string, error)

type endpointVariant struct {
	listPathBuilder   pathBuilder
	detailPathBuilder pathBuilder
	sectionQueryKey   string
	filterAliases     map[string]string
}

type entityEndpoint struct {
	defaultVariant endpointVariant
	ownerVariant   *endpointVariant
}

func (e entityEndpoint) resolveVariant(audience port.SnapshotAudience) endpointVariant {
	variant := e.defaultVariant
	if audience == port.SnapshotAudienceOwner && e.ownerVariant != nil {
		variant = mergeVariants(variant, *e.ownerVariant)
	}
	return variant
}

func mergeVariants(base, override endpointVariant) endpointVariant {
	result := base
	if override.listPathBuilder != nil {
		result.listPathBuilder = override.listPathBuilder
	}
	if override.detailPathBuilder != nil {
		result.detailPathBuilder = override.detailPathBuilder
	}
	if strings.TrimSpace(override.sectionQueryKey) != "" {
		result.sectionQueryKey = override.sectionQueryKey
	}
	if override.filterAliases != nil {
		result.filterAliases = override.filterAliases
	}
	return result
}

var entityEndpoints = map[string]entityEndpoint{
	"restaurants": {
		defaultVariant: endpointVariant{
			listPathBuilder:   staticPathBuilder("/api/v1/admin/restaurants"),
			detailPathBuilder: resourcePathBuilder("/api/v1/admin/restaurants"),
		},
		ownerVariant: &endpointVariant{
			listPathBuilder:   staticPathBuilder("/api/v1/restaurant/me"),
			detailPathBuilder: resourcePathBuilder("/api/v1/restaurant"),
		},
	},
	"tables": {
		defaultVariant: endpointVariant{
			listPathBuilder:   staticPathBuilder("/api/v1/admin/tables"),
			detailPathBuilder: resourcePathBuilder("/api/v1/admin/tables"),
			filterAliases: map[string]string{
				"restaurantid": "restaurantId",
				"sectionid":    "sectionId",
			},
		},
		ownerVariant: &endpointVariant{
			listPathBuilder:   requiredValuePathBuilder("/api/v1/restaurant/tables/section/%s"),
			detailPathBuilder: resourcePathBuilder("/api/v1/restaurant/tables"),
			filterAliases: map[string]string{
				"restaurantid": "restaurantId",
				"sectionid":    "sectionId",
			},
		},
	},
	"reservations": {
		defaultVariant: endpointVariant{
			listPathBuilder:   staticPathBuilder("/api/v1/admin/reservations"),
			detailPathBuilder: resourcePathBuilder("/api/v1/admin/reservations"),
			filterAliases: map[string]string{
				"status":       "status",
				"restaurantid": "restaurantId",
				"date":         "date",
			},
		},
		ownerVariant: &endpointVariant{
			listPathBuilder:   staticPathBuilder("/api/v1/restaurant/reservations"),
			detailPathBuilder: resourcePathBuilder("/api/v1/restaurant/reservations"),
			filterAliases: map[string]string{
				"status":       "status",
				"restaurantid": "restaurantId",
				"date":         "date",
			},
		},
	},
	"reviews": {
		defaultVariant: endpointVariant{
			listPathBuilder:   staticPathBuilder("/api/v1/admin/reviews"),
			detailPathBuilder: resourcePathBuilder("/api/v1/admin/reviews"),
		},
	},
	"sections": {
		defaultVariant: endpointVariant{
			listPathBuilder:   staticPathBuilder("/api/v1/admin/sections"),
			detailPathBuilder: resourcePathBuilder("/api/v1/admin/sections"),
			filterAliases: map[string]string{
				"restaurantid": "restaurantId",
			},
		},
		ownerVariant: &endpointVariant{
			listPathBuilder:   requiredValuePathBuilder("/api/v1/restaurant/sections/restaurant/%s"),
			detailPathBuilder: resourcePathBuilder("/api/v1/restaurant/sections"),
			filterAliases: map[string]string{
				"restaurantid": "restaurantId",
			},
		},
	},
	"objects": {
		defaultVariant: endpointVariant{
			listPathBuilder:   staticPathBuilder("/api/v1/admin/objects"),
			detailPathBuilder: resourcePathBuilder("/api/v1/admin/objects"),
		},
	},
	"menus": {
		defaultVariant: endpointVariant{
			listPathBuilder:   staticPathBuilder("/api/v1/admin/menus"),
			detailPathBuilder: resourcePathBuilder("/api/v1/admin/menus"),
			filterAliases: map[string]string{
				"restaurantid": "restaurantId",
			},
		},
	},
	"dishes": {
		defaultVariant: endpointVariant{
			listPathBuilder:   staticPathBuilder("/api/v1/admin/dishes"),
			detailPathBuilder: resourcePathBuilder("/api/v1/admin/dishes"),
			filterAliases: map[string]string{
				"restaurantid": "restaurantId",
			},
		},
	},
	"images": {
		defaultVariant: endpointVariant{
			listPathBuilder:   staticPathBuilder("/api/v1/admin/images"),
			detailPathBuilder: resourcePathBuilder("/api/v1/admin/images"),
			filterAliases: map[string]string{
				"entityid": "entityId",
			},
		},
	},
	"owners": {
		defaultVariant: endpointVariant{
			listPathBuilder:   staticPathBuilder("/api/v1/auth/admin/users"),
			detailPathBuilder: resourcePathBuilder("/api/v1/auth/admin/users"),
			filterAliases: map[string]string{
				"status":       "status",
				"role":         "role",
				"restaurantid": "restaurantId",
				"active":       "active",
			},
		},
	},
	"section-objects": {
		defaultVariant: endpointVariant{
			listPathBuilder:   staticPathBuilder("/api/v1/admin/section-objects"),
			detailPathBuilder: resourcePathBuilder("/api/v1/admin/section-objects"),
		},
	},
	"payments": {
		defaultVariant: endpointVariant{
			listPathBuilder:   staticPathBuilder("/api/v1/admin/payments"),
			detailPathBuilder: resourcePathBuilder("/api/v1/admin/payments"),
			filterAliases: map[string]string{
				"status":        "status",
				"type":          "type",
				"reservationid": "reservationId",
				"restaurantid":  "restaurantId",
				"startdate":     "startDate",
				"enddate":       "endDate",
				"minamount":     "minAmount",
				"maxamount":     "maxAmount",
			},
		},
		ownerVariant: &endpointVariant{
			listPathBuilder:   requiredValuePathBuilder("/api/v1/restaurant/payments/restaurant/%s"),
			detailPathBuilder: resourcePathBuilder("/api/v1/restaurant/payments"),
		},
	},
	"subscriptions": {
		defaultVariant: endpointVariant{
			listPathBuilder:   staticPathBuilder("/api/v1/admin/subscriptions"),
			detailPathBuilder: resourcePathBuilder("/api/v1/admin/subscriptions"),
		},
		ownerVariant: &endpointVariant{
			listPathBuilder: requiredValuePathBuilder("/api/v1/restaurant/subscriptions/restaurant/%s"),
		},
	},
	"subscription-plans": {
		defaultVariant: endpointVariant{
			listPathBuilder:   staticPathBuilder("/api/v1/admin/subscription-plans"),
			detailPathBuilder: resourcePathBuilder("/api/v1/admin/subscription-plans"),
		},
	},
	"auth-users": {
		defaultVariant: endpointVariant{
			listPathBuilder:   staticPathBuilder("/api/v1/auth/admin/users"),
			detailPathBuilder: resourcePathBuilder("/api/v1/auth/admin/users"),
			filterAliases: map[string]string{
				"status":       "status",
				"role":         "role",
				"restaurantid": "restaurantId",
				"active":       "active",
			},
		},
		ownerVariant: &endpointVariant{
			filterAliases: map[string]string{
				"status":       "status",
				"role":         "role",
				"restaurantid": "restaurantId",
				"active":       "active",
			},
		},
	},
	"owner-upgrade": {
		defaultVariant: endpointVariant{
			listPathBuilder:   staticPathBuilder("/api/v1/owners/upgrade-requests"),
			detailPathBuilder: resourcePathBuilder("/api/v1/owners/upgrade-requests"),
			filterAliases: map[string]string{
				"status": "status",
				"userid": "userId",
			},
		},
	},
}

func staticPathBuilder(path string) pathBuilder {
	trimmed := strings.TrimSpace(path)
	return func(string) (string, error) {
		if trimmed == "" {
			return "", fmt.Errorf("missing path configuration")
		}
		return trimmed, nil
	}
}

func requiredValuePathBuilder(format string) pathBuilder {
	trimmed := strings.TrimSpace(format)
	return func(value string) (string, error) {
		identifier := strings.TrimSpace(value)
		if identifier == "" {
			return "", port.ErrSnapshotNotFound
		}
		return fmt.Sprintf(trimmed, url.PathEscape(identifier)), nil
	}
}

func resourcePathBuilder(base string) pathBuilder {
	trimmed := strings.TrimSpace(base)
	return func(value string) (string, error) {
		identifier := strings.TrimSpace(value)
		if identifier == "" {
			return "", port.ErrSnapshotNotFound
		}
		return strings.TrimRight(trimmed, "/") + "/" + url.PathEscape(identifier), nil
	}
}

func NewSectionSnapshotHTTPClient(baseURL string, timeout time.Duration, client *http.Client) *SectionSnapshotHTTPClient {
	return &SectionSnapshotHTTPClient{rest: NewRESTClient(baseURL, timeout, client), timeout: timeoutOrDefault(timeout)}
}

func (c *SectionSnapshotHTTPClient) FetchEntityList(ctx context.Context, token, entity string, snapshotCtx port.SnapshotContext, query domain.PagedQuery) (*domain.SectionSnapshot, error) {
	endpoint, ok := entityEndpoints[strings.ToLower(strings.TrimSpace(entity))]
	variant := endpoint.resolveVariant(snapshotCtx.Audience)
	if !ok || variant.listPathBuilder == nil {
		slog.Warn("snapshot list entity unsupported", slog.String("entity", entity))
		return nil, port.ErrSnapshotUnsupported
	}

	sectionID := strings.TrimSpace(snapshotCtx.SectionID)
	if sectionID == "" {
		return nil, port.ErrSnapshotNotFound
	}
	slog.Info("snapshot list fetch start", slog.String("entity", entity), slog.String("sectionId", sectionID))

	if c.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	listPath, err := variant.listPathBuilder(sectionID)
	if err != nil {
		slog.Warn("snapshot list path build failed", slog.String("entity", entity), slog.String("sectionId", sectionID), slog.Any("error", err))
		return nil, err
	}

	extraQuery := map[string]string{}
	if key := strings.TrimSpace(variant.sectionQueryKey); key != "" {
		extraQuery[key] = sectionID
	}

	return c.performListRequest(ctx, token, listPath, sectionID, query, extraQuery, variant)
}

func (c *SectionSnapshotHTTPClient) FetchEntityDetail(ctx context.Context, token, entity string, snapshotCtx port.SnapshotContext, resourceID string) (*domain.SectionSnapshot, error) {
	endpoint, ok := entityEndpoints[strings.ToLower(strings.TrimSpace(entity))]
	variant := endpoint.resolveVariant(snapshotCtx.Audience)
	if !ok || variant.detailPathBuilder == nil {
		slog.Warn("snapshot detail entity unsupported", slog.String("entity", entity))
		return nil, port.ErrSnapshotUnsupported
	}

	resource := strings.TrimSpace(resourceID)
	if resource == "" {
		return nil, port.ErrSnapshotNotFound
	}

	slog.Info("snapshot detail fetch start", slog.String("entity", entity), slog.String("resourceId", resource))

	if c.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	detailPath, err := variant.detailPathBuilder(resource)
	if err != nil {
		slog.Warn("snapshot detail path build failed", slog.String("entity", entity), slog.String("resourceId", resource), slog.Any("error", err))
		return nil, err
	}

	return c.performDetailRequest(ctx, token, detailPath)
}

func (c *SectionSnapshotHTTPClient) performListRequest(ctx context.Context, token, path, sectionID string, query domain.PagedQuery, extras map[string]string, variant endpointVariant) (*domain.SectionSnapshot, error) {
	req, err := c.rest.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		slog.Error("snapshot request build failed", slog.String("sectionId", sectionID), slog.String("path", path), slog.Any("error", err))
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	if trimmed := strings.TrimSpace(token); trimmed != "" {
		req.Header.Set("Authorization", "Bearer "+trimmed)
	}

	values := buildQueryValues(query, "", variant)
	for key, value := range extras {
		trimmedKey := strings.TrimSpace(key)
		trimmedValue := strings.TrimSpace(value)
		if trimmedKey == "" || trimmedValue == "" {
			continue
		}
		values.Set(trimmedKey, trimmedValue)
	}
	req.URL.RawQuery = values.Encode()
	slog.Debug("snapshot request", slog.String("url", req.URL.String()))

	res, err := c.rest.Do(req)
	if err != nil {
		slog.Error("snapshot request error", slog.String("sectionId", sectionID), slog.String("path", path), slog.Any("error", err))
		return nil, fmt.Errorf("snapshot request failed: %w", err)
	}
	defer res.Body.Close()
	slog.Debug("snapshot response", slog.Int("status", res.StatusCode), slog.String("url", req.URL.String()))

	if res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden {
		return nil, port.ErrSnapshotForbidden
	}
	if res.StatusCode == http.StatusNotFound {
		return nil, port.ErrSnapshotNotFound
	}
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		slog.Error("snapshot fetch unexpected status", slog.Int("status", res.StatusCode), slog.String("url", req.URL.String()), slog.String("body", strings.TrimSpace(string(body))))
		return nil, fmt.Errorf("unexpected snapshot response %d", res.StatusCode)
	}

	return decodeSectionSnapshot(res.Body)
}

func (c *SectionSnapshotHTTPClient) performDetailRequest(ctx context.Context, token, endpoint string) (*domain.SectionSnapshot, error) {
	req, err := c.rest.NewRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		slog.Error("snapshot detail request build failed", slog.String("path", endpoint), slog.Any("error", err))
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	if trimmed := strings.TrimSpace(token); trimmed != "" {
		req.Header.Set("Authorization", "Bearer "+trimmed)
	}

	slog.Debug("snapshot detail request", slog.String("url", req.URL.String()))

	res, err := c.rest.Do(req)
	if err != nil {
		slog.Error("snapshot detail request error", slog.String("path", endpoint), slog.Any("error", err))
		return nil, fmt.Errorf("snapshot request failed: %w", err)
	}
	defer res.Body.Close()
	slog.Debug("snapshot detail response", slog.Int("status", res.StatusCode), slog.String("url", req.URL.String()))

	if res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden {
		return nil, port.ErrSnapshotForbidden
	}
	if res.StatusCode == http.StatusNotFound {
		return nil, port.ErrSnapshotNotFound
	}
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		slog.Error("snapshot detail unexpected status", slog.Int("status", res.StatusCode), slog.String("url", req.URL.String()), slog.String("body", strings.TrimSpace(string(body))))
		return nil, fmt.Errorf("unexpected snapshot response %d", res.StatusCode)
	}

	return decodeSectionSnapshot(res.Body)
}

func decodeSectionSnapshot(body io.Reader) (*domain.SectionSnapshot, error) {
	var payload interface{}
	if err := json.NewDecoder(body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode snapshot: %w", err)
	}
	slog.Debug("snapshot payload decoded", slog.String("type", fmt.Sprintf("%T", payload)))

	normalized := normalizeSnapshotPayload(payload)
	return &domain.SectionSnapshot{Payload: normalized}, nil
}

func normalizeSnapshotPayload(payload interface{}) map[string]any {
	switch typed := payload.(type) {
	case map[string]interface{}:
		return normalization.MapFromPayload(typed)
	case []interface{}:
		return map[string]any{"items": typed}
	default:
		return map[string]any{"value": typed}
	}
}

func (v endpointVariant) mapFilterKey(key string) string {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return ""
	}
	if len(v.filterAliases) == 0 {
		return trimmed
	}
	aliased, ok := v.filterAliases[strings.ToLower(trimmed)]
	if !ok {
		return trimmed
	}
	return strings.TrimSpace(aliased)
}

func buildQueryValues(query domain.PagedQuery, defaultSearch string, variant endpointVariant) url.Values {
	normalized := query.Normalize(strings.TrimSpace(defaultSearch))
	values := url.Values{}
	values.Set("page", strconv.Itoa(normalized.Page))
	values.Set("limit", strconv.Itoa(normalized.Limit))
	if normalized.Search != "" {
		values.Set("q", normalized.Search)
	}
	if normalized.SortBy != "" {
		values.Set("sortBy", normalized.SortBy)
	}
	if normalized.SortOrder != "" {
		values.Set("sortOrder", normalized.SortOrder)
	}
	for key, value := range normalized.Filters {
		mapped := variant.mapFilterKey(key)
		if mapped == "" {
			continue
		}
		values.Set(mapped, value)
	}
	return values
}

var _ port.SectionSnapshotFetcher = (*SectionSnapshotHTTPClient)(nil)
