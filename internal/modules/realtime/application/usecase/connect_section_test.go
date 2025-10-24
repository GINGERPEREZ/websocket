package usecase

import (
	"context"
	"errors"
	"testing"

	"mesaYaWs/internal/modules/realtime/application/port"
	"mesaYaWs/internal/modules/realtime/domain"
)

func TestConnectSectionUseCase_HandleListCommandSuccess(t *testing.T) {
	t.Parallel()

	uc := &ConnectSectionUseCase{}
	inputQuery := newPagedQuery(0, 0, "  sushi  ", "  name  ", " desc ")
	normalizedQuery := domain.PagedQuery{Page: 3, Limit: 15, Search: "normalized-search", SortBy: "createdAt", SortOrder: "ASC"}
	snapshot := &domain.SectionSnapshot{Payload: map[string]string{"ok": "yes"}}

	fetchFn := func(ctx context.Context, token, sectionID string, query domain.PagedQuery) (*domain.SectionSnapshot, domain.PagedQuery, error) {
		if token != "token" {
			t.Fatalf("unexpected token: %s", token)
		}
		if sectionID != "section-1" {
			t.Fatalf("unexpected section id: %s", sectionID)
		}
		if query != inputQuery {
			t.Fatalf("unexpected query: %#v", query)
		}
		return snapshot, normalizedQuery, nil
	}

	msg, err := uc.handleListCommand(context.Background(), "token", "section-1", " restaurants ", inputQuery, fetchFn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg == nil {
		t.Fatal("expected message, got nil")
	}
	if msg.Topic != "restaurants.list" {
		t.Fatalf("unexpected topic: %s", msg.Topic)
	}
	if msg.Entity != "restaurants" {
		t.Fatalf("unexpected entity: %s", msg.Entity)
	}
	if msg.ResourceID != "section-1" {
		t.Fatalf("unexpected resource id: %s", msg.ResourceID)
	}
	if msg.Data != snapshot.Payload {
		t.Fatalf("unexpected payload: %#v", msg.Data)
	}
	if got := msg.Metadata["page"]; got != "3" {
		t.Fatalf("metadata page mismatch: %s", got)
	}
	if got := msg.Metadata["sortBy"]; got != "createdAt" {
		t.Fatalf("metadata sortBy mismatch: %s", got)
	}
}

func TestConnectSectionUseCase_HandleListCommandError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("fetch failed")
	uc := &ConnectSectionUseCase{}

	fetchFn := func(ctx context.Context, token, sectionID string, query domain.PagedQuery) (*domain.SectionSnapshot, domain.PagedQuery, error) {
		return nil, domain.PagedQuery{}, expectedErr
	}

	_, err := uc.handleListCommand(context.Background(), "token", "section", "entity", domain.PagedQuery{}, fetchFn)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}

func TestConnectSectionUseCase_HandleListCommandSnapshotMissing(t *testing.T) {
	t.Parallel()

	uc := &ConnectSectionUseCase{}

	fetchFn := func(ctx context.Context, token, sectionID string, query domain.PagedQuery) (*domain.SectionSnapshot, domain.PagedQuery, error) {
		return nil, domain.PagedQuery{}, nil
	}

	_, err := uc.handleListCommand(context.Background(), "token", "section", "entity", domain.PagedQuery{}, fetchFn)
	if !errors.Is(err, port.ErrSnapshotNotFound) {
		t.Fatalf("expected ErrSnapshotNotFound, got %v", err)
	}
}

func TestConnectSectionUseCase_HandleDetailCommandSuccess(t *testing.T) {
	t.Parallel()

	uc := &ConnectSectionUseCase{}
	snapshot := &domain.SectionSnapshot{Payload: map[string]string{"key": "value"}}

	fetchFn := func(ctx context.Context, token, sectionID, resourceID string) (*domain.SectionSnapshot, error) {
		if token != "token" {
			t.Fatalf("unexpected token: %s", token)
		}
		if sectionID != "section-42" {
			t.Fatalf("unexpected section id: %s", sectionID)
		}
		if resourceID != "resource-9" {
			t.Fatalf("unexpected resource id: %s", resourceID)
		}
		return snapshot, nil
	}

	msg, err := uc.handleDetailCommand(context.Background(), "token", " section-42 ", " tables ", " resource-9 ", fetchFn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg == nil {
		t.Fatal("expected message, got nil")
	}
	if msg.Topic != "tables.detail" {
		t.Fatalf("unexpected topic: %s", msg.Topic)
	}
	if msg.Metadata["tableId"] != "resource-9" {
		t.Fatalf("expected tableId metadata, got: %v", msg.Metadata)
	}
	if msg.ResourceID != "resource-9" {
		t.Fatalf("unexpected resource id: %s", msg.ResourceID)
	}
	if msg.Metadata["sectionId"] != "section-42" {
		t.Fatalf("expected sectionId metadata, got: %v", msg.Metadata)
	}
	if msg.Data != snapshot.Payload {
		t.Fatalf("unexpected payload: %#v", msg.Data)
	}
}

func TestConnectSectionUseCase_HandleDetailCommandError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("boom")
	uc := &ConnectSectionUseCase{}

	fetchFn := func(ctx context.Context, token, sectionID, resourceID string) (*domain.SectionSnapshot, error) {
		return nil, expectedErr
	}

	_, err := uc.handleDetailCommand(context.Background(), "token", "section", "entity", "resource", fetchFn)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}

func TestConnectSectionUseCase_HandleDetailCommandMissingResource(t *testing.T) {
	t.Parallel()

	uc := &ConnectSectionUseCase{}

	fetchFn := func(ctx context.Context, token, sectionID, resourceID string) (*domain.SectionSnapshot, error) {
		t.Fatal("fetchFn should not be called when resource is empty")
		return nil, nil
	}

	_, err := uc.handleDetailCommand(context.Background(), "token", "section", "entity", "   ", fetchFn)
	if !errors.Is(err, port.ErrSnapshotNotFound) {
		t.Fatalf("expected ErrSnapshotNotFound, got %v", err)
	}
}
