package infrastructure

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"mesaYaWs/internal/modules/realtime/application/port"
	"mesaYaWs/internal/modules/realtime/domain"
)

func TestBuildQueryValues_UsesFilterAliases(t *testing.T) {
	query := domain.PagedQuery{
		Page:      2,
		Limit:     50,
		Search:    "sushi",
		SortBy:    "createdAt",
		SortOrder: "DESC",
		Filters: map[string]string{
			"restaurantid": "rest-1",
			"status":       "paid",
		},
	}

	variant := endpointVariant{
		filterAliases: map[string]string{
			"restaurantid": "restaurantId",
		},
	}

	values := buildQueryValues(query, "", variant)

	if got := values.Get("page"); got != "2" {
		t.Fatalf("expected page=2, got %s", got)
	}
	if got := values.Get("limit"); got != "50" {
		t.Fatalf("expected limit=50, got %s", got)
	}
	if got := values.Get("q"); got != "sushi" {
		t.Fatalf("expected search term, got %s", got)
	}
	if got := values.Get("sortBy"); got != "createdAt" {
		t.Fatalf("expected sortBy=createdAt, got %s", got)
	}
	if got := values.Get("sortOrder"); got != "DESC" {
		t.Fatalf("expected sortOrder=DESC, got %s", got)
	}
	if got := values.Get("restaurantId"); got != "rest-1" {
		t.Fatalf("expected restaurantId alias, got %s", got)
	}
	if got := values.Get("status"); got != "paid" {
		t.Fatalf("expected status filter, got %s", got)
	}
}

func TestEndpointVariantMapFilterKey_FallsBackToTrimmedKey(t *testing.T) {
	variant := endpointVariant{filterAliases: map[string]string{"status": "status"}}

	if got := variant.mapFilterKey(" status "); got != "status" {
		t.Fatalf("expected fallback to same key, got %s", got)
	}
	if got := variant.mapFilterKey("restaurantId"); got != "restaurantId" {
		t.Fatalf("expected original key, got %s", got)
	}
}

func TestEntityEndpointsResolveOwnerVariants(t *testing.T) {
	t.Parallel()

	t.Run("sections", func(t *testing.T) {
		sections := entityEndpoints["sections"].resolveVariant(port.SnapshotAudienceOwner)
		listPath, err := sections.listPathBuilder("  restaurant-123  ")
		if err != nil {
			t.Fatalf("sections list builder failed: %v", err)
		}
		if listPath != "/api/v1/restaurant/sections/restaurant/restaurant-123" {
			t.Fatalf("unexpected sections list path: %s", listPath)
		}
		detailPath, err := sections.detailPathBuilder("  section-9  ")
		if err != nil {
			t.Fatalf("sections detail builder failed: %v", err)
		}
		if detailPath != "/api/v1/restaurant/sections/section-9" {
			t.Fatalf("unexpected sections detail path: %s", detailPath)
		}
	})

	t.Run("tables", func(t *testing.T) {
		tables := entityEndpoints["tables"].resolveVariant(port.SnapshotAudienceOwner)
		listPath, err := tables.listPathBuilder("  section-321  ")
		if err != nil {
			t.Fatalf("tables list builder failed: %v", err)
		}
		if listPath != "/api/v1/restaurant/tables/section/section-321" {
			t.Fatalf("unexpected tables list path: %s", listPath)
		}
		detailPath, err := tables.detailPathBuilder(" table-99 ")
		if err != nil {
			t.Fatalf("tables detail builder failed: %v", err)
		}
		if detailPath != "/api/v1/restaurant/tables/table-99" {
			t.Fatalf("unexpected tables detail path: %s", detailPath)
		}
	})

	t.Run("payments", func(t *testing.T) {
		payments := entityEndpoints["payments"].resolveVariant(port.SnapshotAudienceOwner)
		paymentList, err := payments.listPathBuilder("restaurant-456")
		if err != nil {
			t.Fatalf("payments list builder failed: %v", err)
		}
		if paymentList != "/api/v1/restaurant/payments/restaurant/restaurant-456" {
			t.Fatalf("unexpected payments list path: %s", paymentList)
		}
		paymentDetail, err := payments.detailPathBuilder(" pay-7 ")
		if err != nil {
			t.Fatalf("payments detail builder failed: %v", err)
		}
		if paymentDetail != "/api/v1/restaurant/payments/pay-7" {
			t.Fatalf("unexpected payments detail path: %s", paymentDetail)
		}
	})

	t.Run("subscriptions", func(t *testing.T) {
		subscriptions := entityEndpoints["subscriptions"].resolveVariant(port.SnapshotAudienceOwner)
		subList, err := subscriptions.listPathBuilder(" restaurant-789 ")
		if err != nil {
			t.Fatalf("subscriptions list builder failed: %v", err)
		}
		if subList != "/api/v1/restaurant/subscriptions/restaurant/restaurant-789" {
			t.Fatalf("unexpected subscriptions list path: %s", subList)
		}
		subDetail, err := subscriptions.detailPathBuilder(" sub-1 ")
		if err != nil {
			t.Fatalf("subscriptions detail builder failed: %v", err)
		}
		if subDetail != "/api/v1/admin/subscriptions/sub-1" {
			t.Fatalf("unexpected subscriptions detail path: %s", subDetail)
		}
	})

	t.Run("restaurants", func(t *testing.T) {
		restaurants := entityEndpoints["restaurants"].resolveVariant(port.SnapshotAudienceOwner)
		listPath, err := restaurants.listPathBuilder("ignored")
		if err != nil {
			t.Fatalf("restaurants list builder failed: %v", err)
		}
		if listPath != "/api/v1/restaurant/me" {
			t.Fatalf("unexpected restaurants list path: %s", listPath)
		}
		detailPath, err := restaurants.detailPathBuilder(" rest-42 ")
		if err != nil {
			t.Fatalf("restaurants detail builder failed: %v", err)
		}
		if detailPath != "/api/v1/restaurant/rest-42" {
			t.Fatalf("unexpected restaurants detail path: %s", detailPath)
		}
	})

	t.Run("schedules", func(t *testing.T) {
		schedules := entityEndpoints["schedules"].resolveVariant(port.SnapshotAudienceOwner)
		listPath, err := schedules.listPathBuilder(" rest-555 ")
		if err != nil {
			t.Fatalf("schedules list builder failed: %v", err)
		}
		if listPath != "/api/v1/restaurant/schedules/restaurant/rest-555" {
			t.Fatalf("unexpected schedules list path: %s", listPath)
		}
		detailPath, err := schedules.detailPathBuilder(" sched-1 ")
		if err != nil {
			t.Fatalf("schedules detail builder failed: %v", err)
		}
		if detailPath != "/api/v1/admin/schedules/sched-1" {
			t.Fatalf("unexpected schedules detail path: %s", detailPath)
		}
	})
}

func TestSectionSnapshotHTTPClient_OwnerReservations(t *testing.T) {
	t.Parallel()

	requests := make(chan *http.Request, 2)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.URL.Path == "/api/v1/restaurant/reservations":
			if _, err := io.WriteString(w, `[{"id":"res-1"}]`); err != nil {
				t.Fatalf("write list response: %v", err)
			}
		case strings.HasPrefix(r.URL.Path, "/api/v1/restaurant/reservations/"):
			if _, err := io.WriteString(w, `{"id":"res-1"}`); err != nil {
				t.Fatalf("write detail response: %v", err)
			}
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewSectionSnapshotHTTPClient(server.URL, time.Second, server.Client())
	ctx := port.SnapshotContext{SectionID: "rest-123", Audience: port.SnapshotAudienceOwner}
	query := domain.PagedQuery{
		Filters: map[string]string{
			"ownerId": "should-be-ignored",
			"status":  "confirmed",
			"date":    "2024-05-01",
		},
	}

	snapshot, err := client.FetchEntityList(context.Background(), "token-abc", "reservations", ctx, query)
	if err != nil {
		t.Fatalf("list fetch failed: %v", err)
	}

	listReq := <-requests
	if got := listReq.URL.Path; got != "/api/v1/restaurant/reservations" {
		t.Fatalf("unexpected list path: %s", got)
	}
	if auth := listReq.Header.Get("Authorization"); auth != "Bearer token-abc" {
		t.Fatalf("missing auth header, got %q", auth)
	}
	listQuery := listReq.URL.Query()
	if got := listQuery.Get("restaurantId"); got != "rest-123" {
		t.Fatalf("restaurantId mismatch, got %q", got)
	}
	if _, exists := listQuery["ownerId"]; exists {
		t.Fatalf("ownerId should be removed, query: %#v", listQuery)
	}
	if got := listQuery.Get("status"); got != "confirmed" {
		t.Fatalf("status mismatch, got %q", got)
	}
	if got := listQuery.Get("date"); got != "2024-05-01" {
		t.Fatalf("date mismatch, got %q", got)
	}

	payload, ok := snapshot.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", snapshot.Payload)
	}
	items, ok := payload["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("unexpected items payload: %#v", payload["items"])
	}

	if _, err := client.FetchEntityDetail(context.Background(), "token-abc", "reservations", ctx, "res-1"); err != nil {
		t.Fatalf("detail fetch failed: %v", err)
	}

	detailReq := <-requests
	if got := detailReq.URL.Path; got != "/api/v1/restaurant/reservations/res-1" {
		t.Fatalf("unexpected detail path: %s", got)
	}
	if auth := detailReq.Header.Get("Authorization"); auth != "Bearer token-abc" {
		t.Fatalf("missing auth header on detail, got %q", auth)
	}
}
