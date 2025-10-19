package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path"
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

func (c *SectionSnapshotHTTPClient) FetchSection(ctx context.Context, token, sectionID string, query domain.PagedQuery) (*domain.SectionSnapshot, error) {
	sectionID = strings.TrimSpace(sectionID)
	if sectionID == "" {
		return nil, port.ErrSnapshotNotFound
	}
	log.Printf("snapshot-client: start section=%s", sectionID)

	req, err := c.rest.NewRequest(ctx, http.MethodGet, "/api/v1/restaurant", nil)
	if err != nil {
		log.Printf("snapshot-client: request build failed section=%s err=%v", sectionID, err)
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	if trimmed := strings.TrimSpace(token); trimmed != "" {
		req.Header.Set("Authorization", "Bearer "+trimmed)
	}

	req.URL.RawQuery = query.ToURLValues(sectionID).Encode()
	log.Printf("snapshot-client: requesting url=%s", req.URL.String())

	res, err := c.rest.Do(req)
	if err != nil {
		log.Printf("snapshot-client: request error section=%s err=%v", sectionID, err)
		return nil, fmt.Errorf("snapshot request failed: %w", err)
	}
	defer res.Body.Close()
	log.Printf("snapshot-client: response status=%d url=%s", res.StatusCode, req.URL.String())

	if res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden {
		return nil, port.ErrSnapshotForbidden
	}
	if res.StatusCode == http.StatusNotFound {
		return nil, port.ErrSnapshotNotFound
	}
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		log.Printf("snapshot fetch error status=%d url=%s body=%s", res.StatusCode, req.URL.String(), strings.TrimSpace(string(body)))
		return nil, fmt.Errorf("unexpected snapshot response %d", res.StatusCode)
	}

	return decodeSectionSnapshot(res.Body)
}

func (c *SectionSnapshotHTTPClient) FetchRestaurant(ctx context.Context, token, restaurantID string) (*domain.SectionSnapshot, error) {
	resource := strings.TrimSpace(restaurantID)
	if resource == "" {
		return nil, port.ErrSnapshotNotFound
	}
	log.Printf("snapshot-client: start restaurant id=%s", resource)

	endpoint := path.Join("/api/v1/restaurant", resource)
	req, err := c.rest.NewRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		log.Printf("snapshot-client: request build failed restaurant=%s err=%v", resource, err)
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	if trimmed := strings.TrimSpace(token); trimmed != "" {
		req.Header.Set("Authorization", "Bearer "+trimmed)
	}

	log.Printf("snapshot-client: requesting url=%s", req.URL.String())

	res, err := c.rest.Do(req)
	if err != nil {
		log.Printf("snapshot-client: request error restaurant=%s err=%v", resource, err)
		return nil, fmt.Errorf("snapshot request failed: %w", err)
	}
	defer res.Body.Close()
	log.Printf("snapshot-client: response status=%d url=%s", res.StatusCode, req.URL.String())

	if res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden {
		return nil, port.ErrSnapshotForbidden
	}
	if res.StatusCode == http.StatusNotFound {
		return nil, port.ErrSnapshotNotFound
	}
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		log.Printf("snapshot fetch error status=%d url=%s body=%s", res.StatusCode, req.URL.String(), strings.TrimSpace(string(body)))
		return nil, fmt.Errorf("unexpected snapshot response %d", res.StatusCode)
	}

	return decodeSectionSnapshot(res.Body)
}

func decodeSectionSnapshot(body io.Reader) (*domain.SectionSnapshot, error) {
	var payload interface{}
	if err := json.NewDecoder(body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode snapshot: %w", err)
	}
	log.Printf("snapshot-client: payload decoded type=%T", payload)

	normalized := normalizeSnapshotPayload(payload)
	snapshot := &domain.SectionSnapshot{Payload: normalized}
	if list, ok := domain.BuildRestaurantList(normalized); ok {
		snapshot.RestaurantList = list
	}
	if restaurant, ok := domain.BuildRestaurantDetail(normalized); ok {
		snapshot.Restaurant = restaurant
	}
	return snapshot, nil
}

func normalizeSnapshotPayload(payload interface{}) map[string]any {
	switch typed := payload.(type) {
	case map[string]interface{}:
		return typed
	case []interface{}:
		return map[string]any{"items": typed}
	default:
		return map[string]any{"value": typed}
	}
}

var _ port.SectionSnapshotFetcher = (*SectionSnapshotHTTPClient)(nil)
