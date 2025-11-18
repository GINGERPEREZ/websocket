package usecase

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"mesaYaWs/internal/modules/realtime/application/port"
	"mesaYaWs/internal/modules/realtime/domain"
)

type mockSnapshotFetcher struct {
	listFn   func(ctx context.Context, token, entity, sectionID string, query domain.PagedQuery) (*domain.SectionSnapshot, error)
	detailFn func(ctx context.Context, token, entity, resourceID string) (*domain.SectionSnapshot, error)
}

func (m *mockSnapshotFetcher) FetchEntityList(ctx context.Context, token, entity, sectionID string, query domain.PagedQuery) (*domain.SectionSnapshot, error) {
	if m.listFn == nil {
		return nil, nil
	}
	return m.listFn(ctx, token, entity, sectionID, query)
}

func (m *mockSnapshotFetcher) FetchEntityDetail(ctx context.Context, token, entity, resourceID string) (*domain.SectionSnapshot, error) {
	if m.detailFn == nil {
		return nil, nil
	}
	return m.detailFn(ctx, token, entity, resourceID)
}

func TestConnectSectionUseCase_HandleListCommandSuccess(t *testing.T) {
	t.Parallel()

	inputQuery := newPagedQuery(0, 0, "  sushi  ", "  name  ", " desc ", map[string]string{"status": "pending"})
	normalizedQuery := inputQuery.Normalize("")
	snapshot := &domain.SectionSnapshot{Payload: map[string]string{"ok": "yes"}}

	fetcher := &mockSnapshotFetcher{listFn: func(ctx context.Context, token, entity, sectionID string, query domain.PagedQuery) (*domain.SectionSnapshot, error) {
		if token != "token" {
			t.Fatalf("unexpected token: %s", token)
		}
		if entity != "restaurants" {
			t.Fatalf("unexpected entity: %s", entity)
		}
		if sectionID != "section-1" {
			t.Fatalf("unexpected section id: %s", sectionID)
		}
		if !reflect.DeepEqual(query, normalizedQuery) {
			t.Fatalf("unexpected query: %#v", query)
		}
		return snapshot, nil
	}}

	uc := &ConnectSectionUseCase{SnapshotFetcher: fetcher, cache: newSnapshotCache()}

	msg, err := uc.handleListCommand(context.Background(), "token", "section-1", "restaurants", inputQuery)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg == nil {
		t.Fatal("expected message, got nil")
	}
	if msg.Topic != domain.ListTopic("restaurants") {
		t.Fatalf("unexpected topic: %s", msg.Topic)
	}
	if msg.Entity != "restaurants" {
		t.Fatalf("unexpected entity: %s", msg.Entity)
	}
	if msg.ResourceID != "section-1" {
		t.Fatalf("unexpected resource id: %s", msg.ResourceID)
	}
	data, ok := msg.Data.(map[string]string)
	if !ok {
		t.Fatalf("expected payload map[string]string, got %T", msg.Data)
	}
	if data["ok"] != "yes" {
		t.Fatalf("unexpected payload value: %#v", data)
	}
	if got := msg.Metadata["page"]; got != "1" {
		t.Fatalf("metadata page mismatch: %s", got)
	}
	if got := msg.Metadata["limit"]; got != "20" {
		t.Fatalf("metadata limit mismatch: %s", got)
	}
	if got := msg.Metadata["search"]; got != "sushi" {
		t.Fatalf("metadata search mismatch: %s", got)
	}
	if got := msg.Metadata["sortBy"]; got != "name" {
		t.Fatalf("metadata sortBy mismatch: %s", got)
	}
	if got := msg.Metadata["sortOrder"]; got != "DESC" {
		t.Fatalf("metadata sortOrder mismatch: %s", got)
	}
	if got := msg.Metadata["status"]; got != "pending" {
		t.Fatalf("metadata filter mismatch: %s", got)
	}
	if got := msg.Metadata["origin"]; got != "request" {
		t.Fatalf("metadata origin mismatch: %s", got)
	}
}

func TestConnectSectionUseCase_HandleListCommandError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("fetch failed")
	fetcher := &mockSnapshotFetcher{listFn: func(ctx context.Context, token, entity, sectionID string, query domain.PagedQuery) (*domain.SectionSnapshot, error) {
		return nil, expectedErr
	}}
	uc := &ConnectSectionUseCase{SnapshotFetcher: fetcher, cache: newSnapshotCache()}

	_, err := uc.handleListCommand(context.Background(), "token", "section", "entity", domain.PagedQuery{})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}

func TestConnectSectionUseCase_HandleListCommandSnapshotMissing(t *testing.T) {
	t.Parallel()

	fetcher := &mockSnapshotFetcher{listFn: func(ctx context.Context, token, entity, sectionID string, query domain.PagedQuery) (*domain.SectionSnapshot, error) {
		return nil, nil
	}}
	uc := &ConnectSectionUseCase{SnapshotFetcher: fetcher, cache: newSnapshotCache()}

	_, err := uc.handleListCommand(context.Background(), "token", "section", "entity", domain.PagedQuery{})
	if !errors.Is(err, port.ErrSnapshotNotFound) {
		t.Fatalf("expected ErrSnapshotNotFound, got %v", err)
	}
}

func TestConnectSectionUseCase_HandleListCommandServesCache(t *testing.T) {
	t.Parallel()

	cachedSnapshot := &domain.SectionSnapshot{Payload: map[string]string{"source": "cache"}}
	inputQuery := newPagedQuery(0, 0, " sushi ", "", "", nil)
	normalized := inputQuery.Normalize("")

	uc := &ConnectSectionUseCase{SnapshotFetcher: &mockSnapshotFetcher{listFn: func(ctx context.Context, token, entity, sectionID string, query domain.PagedQuery) (*domain.SectionSnapshot, error) {
		t.Fatal("fetcher should not be called when cache is warm")
		return nil, nil
	}}, cache: newSnapshotCache()}
	uc.cache.set("section-1", "restaurants", cacheKindList, normalized, "", "token", cachedSnapshot)

	msg, err := uc.handleListCommand(context.Background(), "token", "section-1", "restaurants", inputQuery)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg == nil {
		t.Fatal("expected message, got nil")
	}
	data, ok := msg.Data.(map[string]string)
	if !ok {
		t.Fatalf("expected payload map[string]string, got %T", msg.Data)
	}
	if data["source"] != "cache" {
		t.Fatalf("expected cached payload, got %v", data)
	}
}

func TestConnectSectionUseCase_HandleDetailCommandSuccess(t *testing.T) {
	t.Parallel()

	snapshot := &domain.SectionSnapshot{Payload: map[string]string{"key": "value"}}

	fetcher := &mockSnapshotFetcher{detailFn: func(ctx context.Context, token, entity, resourceID string) (*domain.SectionSnapshot, error) {
		if token != "token" {
			t.Fatalf("unexpected token: %s", token)
		}
		if strings.TrimSpace(entity) != "tables" {
			t.Fatalf("unexpected entity: %s", entity)
		}
		if strings.TrimSpace(resourceID) != "resource-9" {
			t.Fatalf("unexpected resource id: %q", resourceID)
		}
		return snapshot, nil
	}}

	uc := &ConnectSectionUseCase{SnapshotFetcher: fetcher, cache: newSnapshotCache()}

	msg, err := uc.handleDetailCommand(context.Background(), "token", " section-42 ", "tables", " resource-9 ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg == nil {
		t.Fatal("expected message, got nil")
	}
	if msg.Topic != domain.DetailTopic("tables") {
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
	data, ok := msg.Data.(map[string]string)
	if !ok {
		t.Fatalf("expected payload map[string]string, got %T", msg.Data)
	}
	if data["key"] != "value" {
		t.Fatalf("unexpected payload value: %#v", data)
	}
}

func TestConnectSectionUseCase_HandleDetailCommandError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("boom")
	fetcher := &mockSnapshotFetcher{detailFn: func(ctx context.Context, token, entity, resourceID string) (*domain.SectionSnapshot, error) {
		return nil, expectedErr
	}}
	uc := &ConnectSectionUseCase{SnapshotFetcher: fetcher, cache: newSnapshotCache()}

	_, err := uc.handleDetailCommand(context.Background(), "token", "section", "entity", "resource")
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}

func TestConnectSectionUseCase_HandleDetailCommandMissingResource(t *testing.T) {
	t.Parallel()

	fetcher := &mockSnapshotFetcher{detailFn: func(ctx context.Context, token, entity, resourceID string) (*domain.SectionSnapshot, error) {
		t.Fatal("detailFn should not be called when resource is empty")
		return nil, nil
	}}
	uc := &ConnectSectionUseCase{SnapshotFetcher: fetcher, cache: newSnapshotCache()}

	_, err := uc.handleDetailCommand(context.Background(), "token", "section", "entity", "   ")
	if !errors.Is(err, port.ErrSnapshotNotFound) {
		t.Fatalf("expected ErrSnapshotNotFound, got %v", err)
	}
}

func TestConnectSectionUseCase_HandleDetailCommandServesCache(t *testing.T) {
	t.Parallel()

	cachedSnapshot := &domain.SectionSnapshot{Payload: map[string]string{"source": "cache"}}
	uc := &ConnectSectionUseCase{SnapshotFetcher: &mockSnapshotFetcher{detailFn: func(ctx context.Context, token, entity, resourceID string) (*domain.SectionSnapshot, error) {
		t.Fatal("fetcher should not be called when cache is warm")
		return nil, nil
	}}, cache: newSnapshotCache()}
	uc.cache.set("section-1", "restaurants", cacheKindItem, domain.PagedQuery{}, "resource-9", "token", cachedSnapshot)

	msg, err := uc.handleDetailCommand(context.Background(), "token", "section-1", "restaurants", "resource-9")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg == nil {
		t.Fatal("expected message, got nil")
	}
	data, ok := msg.Data.(map[string]string)
	if !ok {
		t.Fatalf("expected payload map[string]string, got %T", msg.Data)
	}
	if data["source"] != "cache" {
		t.Fatalf("expected cached payload, got %v", data)
	}
}
