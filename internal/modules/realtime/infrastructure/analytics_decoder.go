package infrastructure

import (
	"encoding/json"
	"fmt"
	"io"

	"mesaYaWs/internal/modules/realtime/domain"
	"mesaYaWs/internal/shared/normalization"
)

func decodeAnalyticsSnapshot(body io.Reader) (*domain.AnalyticsSnapshot, error) {
	var payload any
	if err := json.NewDecoder(body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode analytics: %w", err)
	}
	normalized := normalizeAnalyticsPayload(payload)
	return &domain.AnalyticsSnapshot{Payload: normalized}, nil
}

func normalizeAnalyticsPayload(payload any) any {
	switch typed := payload.(type) {
	case map[string]any:
		return normalization.MapFromPayload(typed)
	case []any:
		return map[string]any{"items": typed}
	default:
		return map[string]any{"value": typed}
	}
}
