package infrastructure

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"mesaYaWs/internal/modules/realtime/application/port"
	"mesaYaWs/internal/modules/realtime/domain"
)

// SectionSnapshotHTTPClient implements SectionSnapshotFetcher using the REST API described in swagger.json.
type SectionSnapshotHTTPClient struct {
	rest    *RESTClient
	timeout time.Duration
}

func NewSectionSnapshotHTTPClient(baseURL string, timeout time.Duration, client *http.Client) *SectionSnapshotHTTPClient {
	return &SectionSnapshotHTTPClient{rest: NewRESTClient(baseURL, timeout, client), timeout: timeoutOrDefault(timeout)}
}

func (c *SectionSnapshotHTTPClient) FetchEntityList(ctx context.Context, token, entity, sectionID string, query domain.PagedQuery) (*domain.SectionSnapshot, error) {
	endpoint, ok := entityEndpoints[strings.ToLower(strings.TrimSpace(entity))]
	if !ok || endpoint.listPathBuilder == nil {
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

	listPath, err := endpoint.listPathBuilder(sectionID)
	if err != nil {
		slog.Warn("snapshot list path build failed", slog.String("entity", entity), slog.String("sectionId", sectionID), slog.Any("error", err))
		return nil, err
	}

	extraQuery := map[string]string{}
	if key := strings.TrimSpace(endpoint.sectionQueryKey); key != "" {
		extraQuery[key] = sectionID
	}

	return c.performListRequest(ctx, token, listPath, sectionID, query, extraQuery)
}

func (c *SectionSnapshotHTTPClient) FetchEntityDetail(ctx context.Context, token, entity, resourceID string) (*domain.SectionSnapshot, error) {
	endpoint, ok := entityEndpoints[strings.ToLower(strings.TrimSpace(entity))]
	if !ok || endpoint.detailPathBuilder == nil {
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

	detailPath, err := endpoint.detailPathBuilder(resource)
	if err != nil {
		slog.Warn("snapshot detail path build failed", slog.String("entity", entity), slog.String("resourceId", resource), slog.Any("error", err))
		return nil, err
	}

	return c.performDetailRequest(ctx, token, detailPath)
}

func (c *SectionSnapshotHTTPClient) performListRequest(ctx context.Context, token, path, sectionID string, query domain.PagedQuery, extras map[string]string) (*domain.SectionSnapshot, error) {
	req, err := c.rest.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		slog.Error("snapshot request build failed", slog.String("sectionId", sectionID), slog.String("path", path), slog.Any("error", err))
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	if trimmed := strings.TrimSpace(token); trimmed != "" {
		req.Header.Set("Authorization", "Bearer "+trimmed)
	}

	values := query.ToURLValues(sectionID)
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

var _ port.SectionSnapshotFetcher = (*SectionSnapshotHTTPClient)(nil)
