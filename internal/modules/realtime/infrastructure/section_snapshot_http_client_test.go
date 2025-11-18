package infrastructure

import (
	"testing"

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

	endpoint := entityEndpoint{
		filterAliases: map[string]string{
			"restaurantid": "restaurantId",
		},
	}

	values := buildQueryValues(query, "", endpoint)

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

func TestEntityEndpointMapFilterKey_FallsBackToTrimmedKey(t *testing.T) {
	endpoint := entityEndpoint{filterAliases: map[string]string{"status": "status"}}

	if got := endpoint.mapFilterKey(" status "); got != "status" {
		t.Fatalf("expected fallback to same key, got %s", got)
	}
	if got := endpoint.mapFilterKey("restaurantId"); got != "restaurantId" {
		t.Fatalf("expected original key, got %s", got)
	}
}
