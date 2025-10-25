package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path"
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

type entityEndpoint struct {
	listPath   string
	detailPath string
}

var entityEndpoints = map[string]entityEndpoint{
	"restaurants":        {listPath: "/api/v1/restaurant", detailPath: "/api/v1/restaurant"},
	"tables":             {listPath: "/api/v1/table", detailPath: "/api/v1/table"},
	"reservations":       {listPath: "/api/v1/reservations", detailPath: "/api/v1/reservations"},
	"reviews":            {listPath: "/api/v1/review", detailPath: "/api/v1/review"},
	"sections":           {listPath: "/api/v1/section", detailPath: "/api/v1/section"},
	"objects":            {listPath: "/api/v1/object", detailPath: "/api/v1/object"},
	"menus":              {listPath: "/api/v1/menus", detailPath: "/api/v1/menus"},
	"dishes":             {listPath: "/api/v1/dishes", detailPath: "/api/v1/dishes"},
	"images":             {listPath: "/api/v1/image", detailPath: "/api/v1/image"},
	"section-objects":    {listPath: "/api/v1/section-object", detailPath: "/api/v1/section-object"},
	"payments":           {listPath: "/api/v1/payments", detailPath: "/api/v1/payments"},
	"subscriptions":      {listPath: "/api/v1/subscriptions", detailPath: "/api/v1/subscriptions"},
	"subscription-plans": {listPath: "/api/v1/subscription-plans", detailPath: "/api/v1/subscription-plans"},
	"auth-users":         {listPath: "/api/v1/auth/admin/users", detailPath: "/api/v1/auth/admin/users"},
}

func NewSectionSnapshotHTTPClient(baseURL string, timeout time.Duration, client *http.Client) *SectionSnapshotHTTPClient {
	return &SectionSnapshotHTTPClient{rest: NewRESTClient(baseURL, timeout, client), timeout: timeoutOrDefault(timeout)}
}

func (c *SectionSnapshotHTTPClient) FetchEntityList(ctx context.Context, token, entity, sectionID string, query domain.PagedQuery) (*domain.SectionSnapshot, error) {
	endpoint, ok := entityEndpoints[strings.ToLower(strings.TrimSpace(entity))]
	if !ok || endpoint.listPath == "" {
		slog.Warn("snapshot list entity unsupported", slog.String("entity", entity))
		return nil, port.ErrSnapshotUnsupported
	}

	sectionID = strings.TrimSpace(sectionID)
	if sectionID == "" {
		return nil, port.ErrSnapshotNotFound
	}
	slog.Info("snapshot list fetch start", slog.String("entity", entity), slog.String("sectionId", sectionID))

	if c.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	return c.performListRequest(ctx, token, endpoint.listPath, sectionID, query)
}

func (c *SectionSnapshotHTTPClient) FetchEntityDetail(ctx context.Context, token, entity, resourceID string) (*domain.SectionSnapshot, error) {
	endpoint, ok := entityEndpoints[strings.ToLower(strings.TrimSpace(entity))]
	if !ok || endpoint.detailPath == "" {
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

	return c.performDetailRequest(ctx, token, endpoint.detailPath, resource)
}

func (c *SectionSnapshotHTTPClient) performListRequest(ctx context.Context, token, basePath, sectionID string, query domain.PagedQuery) (*domain.SectionSnapshot, error) {
	req, err := c.rest.NewRequest(ctx, http.MethodGet, basePath, nil)
	if err != nil {
		slog.Error("snapshot request build failed", slog.String("sectionId", sectionID), slog.String("path", basePath), slog.Any("error", err))
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	if trimmed := strings.TrimSpace(token); trimmed != "" {
		req.Header.Set("Authorization", "Bearer "+trimmed)
	}

	req.URL.RawQuery = query.ToURLValues(sectionID).Encode()
	slog.Debug("snapshot request", slog.String("url", req.URL.String()))

	res, err := c.rest.Do(req)
	if err != nil {
		slog.Error("snapshot request error", slog.String("sectionId", sectionID), slog.String("path", basePath), slog.Any("error", err))
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

func (c *SectionSnapshotHTTPClient) performDetailRequest(ctx context.Context, token, basePath, resource string) (*domain.SectionSnapshot, error) {
	endpoint := path.Join(basePath, resource)
	req, err := c.rest.NewRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		slog.Error("snapshot detail request build failed", slog.String("resourceId", resource), slog.String("path", basePath), slog.Any("error", err))
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	if trimmed := strings.TrimSpace(token); trimmed != "" {
		req.Header.Set("Authorization", "Bearer "+trimmed)
	}

	slog.Debug("snapshot detail request", slog.String("url", req.URL.String()))

	res, err := c.rest.Do(req)
	if err != nil {
		slog.Error("snapshot detail request error", slog.String("resourceId", resource), slog.String("path", basePath), slog.Any("error", err))
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

var _ port.SectionSnapshotFetcher = (*SectionSnapshotHTTPClient)(nil)
