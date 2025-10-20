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
