package domain

import "strings"

// asString trims and returns the string representation of value when possible.
func asString(value any) string {
	if s, ok := value.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

// asInt coerces numeric values supported by the REST layer into Go ints.
func asInt(value any) int {
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

// asStringSlice trims each entry from an arbitrary slice preserving non-empty values.
// asInterfaceSlice normalizes different collection types into a []any.
func asInterfaceSlice(value any) []any {
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
