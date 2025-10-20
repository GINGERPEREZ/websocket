package domain

import "testing"

func TestBuildReservationList(t *testing.T) {
	payload := map[string]any{
		"items": []any{
			map[string]any{"id": "res-1", "tableId": "table-1", "status": "pending", "numberOfGuests": 4},
			map[string]any{"id": "res-2", "tableId": "table-1", "state": "confirmed", "numberOfGuests": 2},
		},
	}

	list, ok := BuildReservationList(payload)
	if !ok {
		t.Fatal("expected reservation list")
	}
	if len(list.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(list.Items))
	}
	if list.Items[0].Status != ReservationStatusPending {
		t.Fatalf("unexpected status for first reservation: %s", list.Items[0].Status)
	}
	if list.Items[1].Status != ReservationStatusConfirmed {
		t.Fatalf("unexpected status for second reservation: %s", list.Items[1].Status)
	}
}
