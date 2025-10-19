package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"mesaYaWs/internal/realtime/application/port"
	"mesaYaWs/internal/realtime/domain"
)

// SectionSnapshotHTTPClient implements SectionSnapshotFetcher using the REST API described in swagger.json.
type SectionSnapshotHTTPClient struct {
	baseURL string
	client  *http.Client
}

func NewSectionSnapshotHTTPClient(baseURL string, client *http.Client) *SectionSnapshotHTTPClient {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed == "" {
		trimmed = "http://localhost:3000"
	}
	trimmed = strings.TrimRight(trimmed, "/")
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &SectionSnapshotHTTPClient{baseURL: trimmed, client: client}
}

func (c *SectionSnapshotHTTPClient) FetchSection(ctx context.Context, token, sectionID string) (*domain.SectionSnapshot, error) {
	sectionID = strings.TrimSpace(sectionID)
	if sectionID == "" {
		return nil, port.ErrSnapshotNotFound
	}

	escapedID := url.PathEscape(sectionID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/api/section/%s", c.baseURL, escapedID), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	if trimmed := strings.TrimSpace(token); trimmed != "" {
		req.Header.Set("Authorization", "Bearer "+trimmed)
	}

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusOK:
		return decodeSectionSnapshot(res.Body)
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, port.ErrSnapshotForbidden
	case http.StatusNotFound:
		return nil, port.ErrSnapshotNotFound
	default:
		body, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		return nil, fmt.Errorf("unexpected snapshot response %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}
}

func decodeSectionSnapshot(body io.Reader) (*domain.SectionSnapshot, error) {
	var payload interface{}
	if err := json.NewDecoder(body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode snapshot: %w", err)
	}

	section := extractSectionMap(payload)
	if section == nil {
		return nil, fmt.Errorf("snapshot payload missing section data")
	}
	return &domain.SectionSnapshot{Section: section}, nil
}

func extractSectionMap(payload interface{}) map[string]any {
	switch typed := payload.(type) {
	case map[string]interface{}:
		if section, ok := typed["data"].(map[string]interface{}); ok {
			return section
		}
		if section, ok := typed["section"].(map[string]interface{}); ok {
			return section
		}
		return typed
	default:
		return nil
	}
}

var _ port.SectionSnapshotFetcher = (*SectionSnapshotHTTPClient)(nil)
