package domain

import (
	"testing"
	"time"

	reservations "mesaYaWs/internal/modules/reservations/domain"
	restaurants "mesaYaWs/internal/modules/restaurants/domain"
	tables "mesaYaWs/internal/modules/tables/domain"
)

func TestBuildDetailMessageIncludesDaysOpenMetadata(t *testing.T) {
	open, _ := time.Parse("15:04", "09:00")
	close, _ := time.Parse("15:04", "18:00")
	snapshot := &SectionSnapshot{
		Payload: map[string]any{"id": "rest-1"},
		Restaurant: &restaurants.Restaurant{
			ID:     "rest-1",
			Name:   "  Fancy Place  ",
			Status: restaurants.RestaurantStatusActive,
			Schedule: restaurants.Schedule{
				Open:  open,
				Close: close,
			},
			DaysOpen:     []restaurants.DayOfWeek{restaurants.Monday, restaurants.Saturday},
			Subscription: 3,
		},
		TableList: &tables.TableList{
			Items: []tables.Table{
				{ID: "table-1", State: tables.TableStateAvailable},
				{ID: "table-2", State: tables.TableStateReserved},
				{ID: "table-3", State: tables.TableStateCleaning},
			},
		},
		Table: &tables.Table{ID: "table-2", State: tables.TableStateReserved, Number: 12, Capacity: 4},
		ReservationList: &reservations.ReservationList{
			Items: []reservations.Reservation{
				{ID: "res-1", Status: reservations.ReservationStatusPending},
				{ID: "res-2", Status: reservations.ReservationStatusConfirmed},
				{ID: "res-3", Status: reservations.ReservationStatusCancelled},
				{ID: "res-4", Status: reservations.ReservationStatusNoShow},
			},
		},
		Reservation: &reservations.Reservation{
			ID:              "res-2",
			TableID:         "table-2",
			Status:          reservations.ReservationStatusConfirmed,
			Guests:          5,
			ReservationDate: "2025-10-20",
			ReservationTime: "20:30",
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
		RestaurantList: &restaurants.RestaurantList{
			Items: []restaurants.Restaurant{{ID: "1"}, {ID: "2"}},
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
