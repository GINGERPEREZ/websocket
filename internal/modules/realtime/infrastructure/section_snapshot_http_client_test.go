package infrastructure

import (
	"testing"

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
}
