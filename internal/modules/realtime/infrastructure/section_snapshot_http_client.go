package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"mesaYaWs/internal/modules/realtime/application/port"
	"mesaYaWs/internal/modules/realtime/domain"
	reservations "mesaYaWs/internal/modules/reservations/domain"
	restaurants "mesaYaWs/internal/modules/restaurants/domain"
	tables "mesaYaWs/internal/modules/tables/domain"
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

func (c *SectionSnapshotHTTPClient) FetchSection(ctx context.Context, token, sectionID string, query domain.PagedQuery) (*domain.SectionSnapshot, error) {
	return c.FetchEntityList(ctx, token, "restaurants", sectionID, query)
}

func (c *SectionSnapshotHTTPClient) FetchRestaurant(ctx context.Context, token, restaurantID string) (*domain.SectionSnapshot, error) {
	return c.FetchEntityDetail(ctx, token, "restaurants", restaurantID)
}

func (c *SectionSnapshotHTTPClient) FetchTable(ctx context.Context, token, tableID string) (*domain.SectionSnapshot, error) {
	return c.FetchEntityDetail(ctx, token, "tables", tableID)
}

func (c *SectionSnapshotHTTPClient) FetchReservation(ctx context.Context, token, reservationID string) (*domain.SectionSnapshot, error) {
	return c.FetchEntityDetail(ctx, token, "reservations", reservationID)
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
	snapshot := &domain.SectionSnapshot{Payload: normalized}

	if list, ok := restaurants.BuildRestaurantList(normalized); ok {
		snapshot.MergeListMetadata(restaurantListMetadata(list))
	}
	if restaurant, ok := restaurants.BuildRestaurantDetail(normalized); ok {
		snapshot.MergeDetailMetadata(restaurantDetailMetadata(restaurant))
	}
	if tableList, ok := tables.BuildTableList(normalized); ok {
		meta := tableListMetadata(tableList)
		snapshot.MergeListMetadata(meta)
		snapshot.MergeDetailMetadata(meta)
	}
	if table, ok := tables.BuildTableDetail(normalized); ok {
		snapshot.MergeDetailMetadata(tableDetailMetadata(table))
	}
	if reservationList, ok := reservations.BuildReservationList(normalized); ok {
		meta := reservationListMetadata(reservationList)
		snapshot.MergeListMetadata(meta)
		snapshot.MergeDetailMetadata(meta)
	}
	if reservation, ok := reservations.BuildReservationDetail(normalized); ok {
		snapshot.MergeDetailMetadata(reservationDetailMetadata(reservation))
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

func restaurantListMetadata(list *restaurants.RestaurantList) domain.Metadata {
	if list == nil {
		return nil
	}
	meta := domain.Metadata{}
	if count := len(list.Items); count > 0 {
		meta["itemsCount"] = strconv.Itoa(count)
	}
	if list.Total > 0 {
		meta["total"] = strconv.Itoa(list.Total)
	}
	return meta
}

func restaurantDetailMetadata(restaurant *restaurants.Restaurant) domain.Metadata {
	if restaurant == nil {
		return nil
	}
	meta := domain.Metadata{}
	if name := strings.TrimSpace(restaurant.Name); name != "" {
		meta["restaurantName"] = name
	}
	if status := strings.TrimSpace(string(restaurant.Status)); status != "" {
		meta["restaurantStatus"] = status
	}
	schedule := restaurant.Schedule
	if !schedule.Open.IsZero() {
		meta["openTime"] = schedule.Open.Format("15:04")
	}
	if !schedule.Close.IsZero() {
		meta["closeTime"] = schedule.Close.Format("15:04")
	}
	if schedule.HasBothTimes() {
		meta["openDurationMinutes"] = strconv.Itoa(int(schedule.Duration().Minutes()))
	}
	if restaurant.Subscription > 0 {
		meta["subscriptionId"] = strconv.Itoa(restaurant.Subscription)
	}
	if len(restaurant.DaysOpen) > 0 {
		names := make([]string, 0, len(restaurant.DaysOpen))
		for _, day := range restaurant.DaysOpen {
			if trimmed := strings.TrimSpace(string(day)); trimmed != "" {
				names = append(names, trimmed)
			}
		}
		if len(names) > 0 {
			meta["daysOpen"] = strings.Join(names, ",")
		}
	}
	return meta
}

func tableListMetadata(list *tables.TableList) domain.Metadata {
	if list == nil || len(list.Items) == 0 {
		return nil
	}
	summary := struct {
		available int
		reserved  int
		seated    int
		blocked   int
		cleaning  int
	}{}
	for _, table := range list.Items {
		switch table.State {
		case tables.TableStateAvailable:
			summary.available++
		case tables.TableStateReserved:
			summary.reserved++
		case tables.TableStateSeated:
			summary.seated++
		case tables.TableStateBlocked:
			summary.blocked++
		case tables.TableStateCleaning:
			summary.cleaning++
		}
	}
	meta := domain.Metadata{"tablesCount": strconv.Itoa(len(list.Items))}
	if summary.available > 0 {
		meta["tablesAvailable"] = strconv.Itoa(summary.available)
	}
	if summary.reserved > 0 {
		meta["tablesReserved"] = strconv.Itoa(summary.reserved)
	}
	if summary.seated > 0 {
		meta["tablesSeated"] = strconv.Itoa(summary.seated)
	}
	if summary.blocked > 0 {
		meta["tablesBlocked"] = strconv.Itoa(summary.blocked)
	}
	if summary.cleaning > 0 {
		meta["tablesCleaning"] = strconv.Itoa(summary.cleaning)
	}
	return meta
}

func tableDetailMetadata(table *tables.Table) domain.Metadata {
	if table == nil {
		return nil
	}
	meta := domain.Metadata{}
	if id := strings.TrimSpace(table.ID); id != "" {
		meta["tableId"] = id
	}
	if state := strings.TrimSpace(string(table.State)); state != "" {
		meta["tableState"] = state
	}
	if table.Number > 0 {
		meta["tableNumber"] = strconv.Itoa(table.Number)
	}
	if table.Capacity > 0 {
		meta["tableCapacity"] = strconv.Itoa(table.Capacity)
	}
	return meta
}

func reservationListMetadata(list *reservations.ReservationList) domain.Metadata {
	if list == nil || len(list.Items) == 0 {
		return nil
	}
	summary := struct {
		pending   int
		confirmed int
		seated    int
		completed int
		cancelled int
		noShow    int
	}{}
	for _, reservation := range list.Items {
		switch reservation.Status {
		case reservations.ReservationStatusPending:
			summary.pending++
		case reservations.ReservationStatusConfirmed:
			summary.confirmed++
		case reservations.ReservationStatusSeated:
			summary.seated++
		case reservations.ReservationStatusCompleted:
			summary.completed++
		case reservations.ReservationStatusCancelled:
			summary.cancelled++
		case reservations.ReservationStatusNoShow:
			summary.noShow++
		}
	}
	meta := domain.Metadata{"reservationsCount": strconv.Itoa(len(list.Items))}
	if summary.pending > 0 {
		meta["reservationsPending"] = strconv.Itoa(summary.pending)
	}
	if summary.confirmed > 0 {
		meta["reservationsConfirmed"] = strconv.Itoa(summary.confirmed)
	}
	if summary.seated > 0 {
		meta["reservationsSeated"] = strconv.Itoa(summary.seated)
	}
	if summary.completed > 0 {
		meta["reservationsCompleted"] = strconv.Itoa(summary.completed)
	}
	if summary.cancelled > 0 {
		meta["reservationsCancelled"] = strconv.Itoa(summary.cancelled)
	}
	if summary.noShow > 0 {
		meta["reservationsNoShow"] = strconv.Itoa(summary.noShow)
	}
	return meta
}

func reservationDetailMetadata(reservation *reservations.Reservation) domain.Metadata {
	if reservation == nil {
		return nil
	}
	meta := domain.Metadata{}
	if id := strings.TrimSpace(reservation.ID); id != "" {
		meta["reservationId"] = id
	}
	if status := strings.TrimSpace(string(reservation.Status)); status != "" {
		meta["reservationStatus"] = status
	}
	if reservation.Guests > 0 {
		meta["reservationGuests"] = strconv.Itoa(reservation.Guests)
	}
	if date := strings.TrimSpace(reservation.ReservationDate); date != "" {
		meta["reservationDate"] = date
	}
	if timeStr := strings.TrimSpace(reservation.ReservationTime); timeStr != "" {
		meta["reservationTime"] = timeStr
	}
	if tableID := strings.TrimSpace(reservation.TableID); tableID != "" {
		meta["reservationTableId"] = tableID
	}
	return meta
}

var _ port.SectionSnapshotFetcher = (*SectionSnapshotHTTPClient)(nil)
