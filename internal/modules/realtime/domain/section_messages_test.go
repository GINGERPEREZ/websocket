package domain

import (
	"testing"
	"time"
)

func TestBuildDetailMessageIncludesDaysOpenMetadata(t *testing.T) {
	open, _ := time.Parse("15:04", "09:00")
	close, _ := time.Parse("15:04", "18:00")
	snapshot := &SectionSnapshot{
		Payload: map[string]any{"id": "rest-1"},
		DetailMetadata: Metadata{
			"restaurantName":        "Fancy Place",
			"restaurantStatus":      "ACTIVE",
			"openTime":              open.Format("15:04"),
			"closeTime":             close.Format("15:04"),
			"openDurationMinutes":   "540",
			"subscriptionId":        "3",
			"daysOpen":              "MONDAY,SATURDAY",
			"tablesCount":           "3",
			"tablesAvailable":       "1",
			"tablesReserved":        "1",
			"tablesCleaning":        "1",
			"tableId":               "table-2",
			"tableState":            "RESERVED",
			"tableNumber":           "12",
			"tableCapacity":         "4",
			"reservationsCount":     "4",
			"reservationsPending":   "1",
			"reservationsConfirmed": "1",
			"reservationsCancelled": "1",
			"reservationsNoShow":    "1",
			"reservationId":         "res-2",
			"reservationStatus":     "CONFIRMED",
			"reservationGuests":     "5",
			"reservationDate":       "2025-10-20",
			"reservationTime":       "20:30",
			"reservationTableId":    "table-2",
		},
	}

	at := time.Date(2025, time.October, 19, 15, 4, 0, 0, time.UTC)
	msg := BuildDetailMessage(" restaurant ", "section-1", "rest-1", snapshot, at)
	if msg == nil {
		t.Fatal("expected message, got nil")
	}

	if msg.Topic != DetailTopic("restaurant") {
		t.Fatalf("unexpected topic: %s", msg.Topic)
	}
	if msg.Entity != "restaurant" {
		t.Fatalf("unexpected entity: %s", msg.Entity)
	}
	if msg.ResourceID != "rest-1" {
		t.Fatalf("unexpected resource id: %s", msg.ResourceID)
	}
	if !msg.Timestamp.Equal(at.UTC()) {
		t.Fatalf("timestamp mismatch want=%s got=%s", at.UTC(), msg.Timestamp)
	}

	md := msg.Metadata
	if md == nil {
		t.Fatal("expected metadata map")
	}
	if md["sectionId"] != "section-1" {
		t.Fatalf("sectionId metadata mismatch: %s", md["sectionId"])
	}
	if md["daysOpen"] != "MONDAY,SATURDAY" {
		t.Fatalf("daysOpen metadata mismatch: %s", md["daysOpen"])
	}
	if md["restaurantStatus"] != "ACTIVE" {
		t.Fatalf("restaurantStatus metadata mismatch: %s", md["restaurantStatus"])
	}
	if md["openTime"] != "09:00" {
		t.Fatalf("openTime metadata mismatch: %s", md["openTime"])
	}
	if md["closeTime"] != "18:00" {
		t.Fatalf("closeTime metadata mismatch: %s", md["closeTime"])
	}
	if md["openDurationMinutes"] != "540" {
		t.Fatalf("duration metadata mismatch: %s", md["openDurationMinutes"])
	}
	if md["subscriptionId"] != "3" {
		t.Fatalf("subscriptionId metadata mismatch: %s", md["subscriptionId"])
	}
	if md["tablesCount"] != "3" {
		t.Fatalf("tablesCount metadata mismatch: %s", md["tablesCount"])
	}
	if md["tablesAvailable"] != "1" {
		t.Fatalf("tablesAvailable metadata mismatch: %s", md["tablesAvailable"])
	}
	if md["tablesReserved"] != "1" {
		t.Fatalf("tablesReserved metadata mismatch: %s", md["tablesReserved"])
	}
	if md["tablesCleaning"] != "1" {
		t.Fatalf("tablesCleaning metadata mismatch: %s", md["tablesCleaning"])
	}
	if md["tableState"] != "RESERVED" {
		t.Fatalf("tableState metadata mismatch: %s", md["tableState"])
	}
	if md["tableNumber"] != "12" {
		t.Fatalf("tableNumber metadata mismatch: %s", md["tableNumber"])
	}
	if md["tableCapacity"] != "4" {
		t.Fatalf("tableCapacity metadata mismatch: %s", md["tableCapacity"])
	}
	if md["reservationsCount"] != "4" {
		t.Fatalf("reservationsCount metadata mismatch: %s", md["reservationsCount"])
	}
	if md["reservationsPending"] != "1" {
		t.Fatalf("reservationsPending metadata mismatch: %s", md["reservationsPending"])
	}
	if md["reservationsConfirmed"] != "1" {
		t.Fatalf("reservationsConfirmed metadata mismatch: %s", md["reservationsConfirmed"])
	}
	if md["reservationsCancelled"] != "1" {
		t.Fatalf("reservationsCancelled metadata mismatch: %s", md["reservationsCancelled"])
	}
	if md["reservationsNoShow"] != "1" {
		t.Fatalf("reservationsNoShow metadata mismatch: %s", md["reservationsNoShow"])
	}
	if md["reservationStatus"] != "CONFIRMED" {
		t.Fatalf("reservationStatus metadata mismatch: %s", md["reservationStatus"])
	}
	if md["reservationGuests"] != "5" {
		t.Fatalf("reservationGuests metadata mismatch: %s", md["reservationGuests"])
	}
	if md["reservationDate"] != "2025-10-20" {
		t.Fatalf("reservationDate metadata mismatch: %s", md["reservationDate"])
	}
	if md["reservationTime"] != "20:30" {
		t.Fatalf("reservationTime metadata mismatch: %s", md["reservationTime"])
	}
	if md["reservationTableId"] != "table-2" {
		t.Fatalf("reservationTableId metadata mismatch: %s", md["reservationTableId"])
	}
}

func TestBuildListMessageIncludesCounts(t *testing.T) {
	snapshot := &SectionSnapshot{
		Payload: map[string]any{"items": []any{"a", "b"}},
		ListMetadata: Metadata{
			"itemsCount": "2",
			"total":      "10",
		},
	}
	query := PagedQuery{Page: 2, Limit: 50, Search: ""}
	at := time.Date(2025, time.October, 19, 15, 4, 0, 0, time.UTC)

	msg := BuildListMessage(" restaurant ", "section-1", snapshot, query, at)
	if msg == nil {
		t.Fatal("expected message, got nil")
	}

	if msg.Topic != ListTopic("restaurant") {
		t.Fatalf("unexpected topic: %s", msg.Topic)
	}
	if msg.Entity != "restaurant" {
		t.Fatalf("unexpected entity: %s", msg.Entity)
	}
	if msg.ResourceID != "section-1" {
		t.Fatalf("unexpected resource id: %s", msg.ResourceID)
	}

	md := msg.Metadata
	if md == nil {
		t.Fatal("expected metadata map")
	}
	if md["sectionId"] != "section-1" {
		t.Fatalf("sectionId metadata mismatch: %s", md["sectionId"])
	}
	if md["page"] != "2" {
		t.Fatalf("page metadata mismatch: %s", md["page"])
	}
	if md["limit"] != "50" {
		t.Fatalf("limit metadata mismatch: %s", md["limit"])
	}
	if md["itemsCount"] != "2" {
		t.Fatalf("itemsCount metadata mismatch: %s", md["itemsCount"])
	}
	if md["total"] != "10" {
		t.Fatalf("total metadata mismatch: %s", md["total"])
	}
}
