package normalization

import (
	"strconv"
	"strings"
)

// asString trims and returns the string representation of value when possible.
func AsString(value any) string {
	if s, ok := value.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

// asInt coerces numeric values supported by the REST layer into Go ints.
func AsInt(value any) int {
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case float32:
		return int(typed)
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	default:
		return 0
	}
}

// asFloat64 coerces numeric values (including numeric strings) into float64.
func AsFloat64(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int32:
		return float64(typed)
	case int64:
		return float64(typed)
	case string:
		if trimmed := strings.TrimSpace(typed); trimmed != "" {
			if parsed, err := strconv.ParseFloat(trimmed, 64); err == nil {
				return parsed
			}
		}
	}
	return 0
}

// asStringSlice trims each entry from an arbitrary slice preserving non-empty values.
// asInterfaceSlice normalizes different collection types into a []any.
func AsInterfaceSlice(value any) []any {
	switch typed := value.(type) {
	case []any:
		return typed
	case []map[string]any:
		items := make([]any, 0, len(typed))
		for _, entry := range typed {
			items = append(items, entry)
		}
		return items
	default:
		return nil
	}
}

// mapFromPayload attempts to unwrap common envelope structures (e.g. {"data": {...}})
// into a plain map for normalization routines.
func MapFromPayload(value any) map[string]any {
	if value == nil {
		return nil
	}
	if typed, ok := value.(map[string]any); ok {
		if data, ok := typed["data"].(map[string]any); ok {
			return data
		}
		return typed
	}
	return nil
}
