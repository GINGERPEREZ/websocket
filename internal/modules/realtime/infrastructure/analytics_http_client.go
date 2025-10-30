package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"mesaYaWs/internal/modules/realtime/application/port"
	"mesaYaWs/internal/modules/realtime/domain"
	"mesaYaWs/internal/shared/normalization"
)

// AnalyticsHTTPClient implements AnalyticsFetcher by calling the REST analytics endpoints.
type AnalyticsHTTPClient struct {
	rest    *RESTClient
	timeout time.Duration
}

// NewAnalyticsHTTPClient creates a new analytics REST client.
func NewAnalyticsHTTPClient(baseURL string, timeout time.Duration, client *http.Client) *AnalyticsHTTPClient {
	return &AnalyticsHTTPClient{rest: NewRESTClient(baseURL, timeout, client), timeout: timeoutOrDefault(timeout)}
}

// Fetch invokes the configured analytics endpoint and decodes the response payload.
func (c *AnalyticsHTTPClient) Fetch(ctx context.Context, token, path string, query map[string]string) (*domain.AnalyticsSnapshot, error) {
	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		return nil, fmt.Errorf("analytics fetch missing path")
	}
	if !strings.HasPrefix(trimmedPath, "/") {
		trimmedPath = "/" + trimmedPath
	}

	if c.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	req, err := c.rest.NewRequest(ctx, http.MethodGet, trimmedPath, nil)
	if err != nil {
		slog.Error("analytics request build failed", slog.String("path", trimmedPath), slog.Any("error", err))
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	if trimmedToken := strings.TrimSpace(token); trimmedToken != "" {
		req.Header.Set("Authorization", "Bearer "+trimmedToken)
	}

	values := url.Values{}
	for key, value := range query {
		trimmedKey := strings.TrimSpace(key)
		trimmedValue := strings.TrimSpace(value)
		if trimmedKey == "" || trimmedValue == "" {
			continue
		}
		values.Set(trimmedKey, trimmedValue)
	}
	if len(values) > 0 {
		req.URL.RawQuery = values.Encode()
	}

	slog.Debug("analytics request", slog.String("url", req.URL.String()))

	res, err := c.rest.Do(req)
	if err != nil {
		slog.Error("analytics request error", slog.String("path", trimmedPath), slog.Any("error", err))
		return nil, fmt.Errorf("analytics request failed: %w", err)
	}
	defer res.Body.Close()

	slog.Debug("analytics response", slog.Int("status", res.StatusCode), slog.String("url", req.URL.String()))

	switch res.StatusCode {
	case http.StatusOK:
		return decodeAnalyticsSnapshot(res.Body)
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, port.ErrAnalyticsForbidden
	case http.StatusNotFound:
		return nil, port.ErrAnalyticsNotFound
	default:
		body, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		slog.Error("analytics fetch unexpected status", slog.Int("status", res.StatusCode), slog.String("url", req.URL.String()), slog.String("body", strings.TrimSpace(string(body))))
		return nil, fmt.Errorf("unexpected analytics response %d", res.StatusCode)
	}
}

func decodeAnalyticsSnapshot(body io.Reader) (*domain.AnalyticsSnapshot, error) {
	var payload any
	if err := json.NewDecoder(body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode analytics: %w", err)
	}
	normalized := normalizeAnalyticsPayload(payload)
	return &domain.AnalyticsSnapshot{Payload: normalized}, nil
}

func normalizeAnalyticsPayload(payload any) any {
	switch typed := payload.(type) {
	case map[string]any:
		return normalization.MapFromPayload(typed)
	case []any:
		return map[string]any{"items": typed}
	default:
		return map[string]any{"value": typed}
	}
}

var _ port.AnalyticsFetcher = (*AnalyticsHTTPClient)(nil)
