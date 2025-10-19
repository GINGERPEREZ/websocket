package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"mesaYaWs/internal/realtime/application/port"
	"mesaYaWs/internal/realtime/domain"
)

// SectionSnapshotHTTPClient implements SectionSnapshotFetcher using the REST API described in swagger.json.
type SectionSnapshotHTTPClient struct {
	rest *RESTClient
}

func NewSectionSnapshotHTTPClient(baseURL string, client *http.Client) *SectionSnapshotHTTPClient {
	return &SectionSnapshotHTTPClient{rest: NewRESTClient(baseURL, client)}
}

func (c *SectionSnapshotHTTPClient) FetchSection(ctx context.Context, token, sectionID string) (*domain.SectionSnapshot, error) {
	sectionID = strings.TrimSpace(sectionID)
	if sectionID == "" {
		return nil, port.ErrSnapshotNotFound
	}

	req, err := c.rest.NewRequest(ctx, http.MethodGet, "/api/v1/restaurant", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	if trimmed := strings.TrimSpace(token); trimmed != "" {
		req.Header.Set("Authorization", "Bearer "+trimmed)
	}

	query := req.URL.Query()
	if _, exists := query["page"]; !exists {
		query.Set("page", "1")
	}
	if _, exists := query["limit"]; !exists {
		query.Set("limit", "20")
	}
	if sectionID != "" {
		query.Set("q", sectionID)
	}
	req.URL.RawQuery = query.Encode()

	res, err := c.rest.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden {
		return nil, port.ErrSnapshotForbidden
	}
	if res.StatusCode == http.StatusNotFound {
		return nil, port.ErrSnapshotNotFound
	}
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		return nil, fmt.Errorf("unexpected snapshot response %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	return decodeSectionSnapshot(res.Body)
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
