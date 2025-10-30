package infrastructure

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"

	"mesaYaWs/internal/modules/realtime/domain"
	"mesaYaWs/internal/shared/normalization"
)

func decodeSectionSnapshot(body io.Reader) (*domain.SectionSnapshot, error) {
	var payload interface{}
	if err := json.NewDecoder(body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode snapshot: %w", err)
	}
	slog.Debug("snapshot payload decoded", slog.String("type", fmt.Sprintf("%T", payload)))

	normalized := normalizeSnapshotPayload(payload)
	return &domain.SectionSnapshot{Payload: normalized}, nil
}

func normalizeSnapshotPayload(payload interface{}) map[string]any {
	switch typed := payload.(type) {
	case map[string]interface{}:
		return normalization.MapFromPayload(typed)
	case []interface{}:
		return map[string]any{"items": typed}
	default:
		return map[string]any{"value": typed}
	}
}
