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
		Restaurant: &Restaurant{
			ID:     "rest-1",
			Name:   "  Fancy Place  ",
			Status: RestaurantStatusActive,
			Schedule: Schedule{
				Open:  open,
				Close: close,
			},
			DaysOpen:     []DayOfWeek{Monday, Saturday},
			Subscription: 3,
		},
	}

	at := time.Date(2025, time.October, 19, 15, 4, 0, 0, time.UTC)
	msg := BuildDetailMessage(" restaurant ", "section-1", "rest-1", snapshot, at)
	if msg == nil {
		t.Fatal("expected message, got nil")
	}

	if msg.Topic != "restaurant.detail" {
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
}

func TestBuildListMessageIncludesCounts(t *testing.T) {
	snapshot := &SectionSnapshot{
		Payload: map[string]any{"items": []any{"a", "b"}},
		RestaurantList: &RestaurantList{
			Items: []Restaurant{{ID: "1"}, {ID: "2"}},
			Total: 10,
		},
	}
	query := PagedQuery{Page: 2, Limit: 50, Search: ""}
	at := time.Date(2025, time.October, 19, 15, 4, 0, 0, time.UTC)

	msg := BuildListMessage(" restaurant ", "section-1", snapshot, query, at)
	if msg == nil {
		t.Fatal("expected message, got nil")
	}

	if msg.Topic != "restaurant.list" {
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
