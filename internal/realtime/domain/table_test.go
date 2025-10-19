package domain

import "testing"

func TestBuildTableList(t *testing.T) {
    payload := map[string]any{
        "items": []any{
            map[string]any{"id": "table-1", "sectionId": "section-1", "number": 10, "capacity": 4, "state": "available"},
            map[string]any{"id": "table-2", "sectionId": "section-1", "number": 11, "capacity": 2, "status": "reserved"},
        },
        "total": 5,
    }

    list, ok := BuildTableList(payload)
    if !ok {
        t.Fatal("expected table list")
    }
    if list.Total != 5 {
        t.Fatalf("unexpected total: %d", list.Total)
    }
    if len(list.Items) != 2 {
        t.Fatalf("expected 2 items, got %d", len(list.Items))
    }
    if list.Items[0].State != TableStateAvailable {
        t.Fatalf("unexpected state for first table: %s", list.Items[0].State)
    }
    if list.Items[1].State != TableStateReserved {
        t.Fatalf("unexpected state for second table: %s", list.Items[1].State)
    }
}

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
