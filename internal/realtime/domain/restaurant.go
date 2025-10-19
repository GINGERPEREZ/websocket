package domain

import (
	"strings"
	"time"
)

// Restaurant represents the restaurant aggregate root exposed over realtime.
type Restaurant struct {
	ID            string
	Name          string
	Description   string
	Location      string
	OpenTime      time.Time
	CloseTime     time.Time
	DaysOpen      []string
	TotalCapacity int
	Subscription  int
	ImageID       int
}

// RestaurantList wraps a collection of restaurants with paging metadata.
type RestaurantList struct {
	Items []Restaurant
	Total int
}

// NormalizeRestaurant builds a Restaurant from a map payload coming from the REST service.
// If required fields are missing it returns false.
func NormalizeRestaurant(raw map[string]any) (Restaurant, bool) {
	id, _ := raw["id"].(string)
	name, _ := raw["name"].(string)
	if strings.TrimSpace(id) == "" || strings.TrimSpace(name) == "" {
		return Restaurant{}, false
	}
	resto := Restaurant{
		ID:            strings.TrimSpace(id),
		Name:          strings.TrimSpace(name),
		Description:   asString(raw["description"]),
		Location:      asString(raw["location"]),
		DaysOpen:      asStringSlice(raw["daysOpen"]),
		TotalCapacity: asInt(raw["totalCapacity"]),
		Subscription:  asInt(raw["subscriptionId"]),
		ImageID:       asInt(raw["imageId"]),
	}
	if open := parseTime(raw["openTime"]); !open.IsZero() {
		resto.OpenTime = open
	}
	if close := parseTime(raw["closeTime"]); !close.IsZero() {
		resto.CloseTime = close
	}
	return resto, true
}

// BuildRestaurantList tries to project an arbitrary payload into a RestaurantList.
func BuildRestaurantList(payload any) (*RestaurantList, bool) {
	container, ok := payload.(map[string]any)
	if !ok {
		return nil, false
	}
	rawItems := container["items"]
	itemsSlice := asInterfaceSlice(rawItems)
	if len(itemsSlice) == 0 {
		return nil, false
	}
	result := RestaurantList{Items: make([]Restaurant, 0, len(itemsSlice))}
	for _, item := range itemsSlice {
		if rawMap, ok := item.(map[string]any); ok {
			if resto, ok := NormalizeRestaurant(rawMap); ok {
				result.Items = append(result.Items, resto)
			}
		}
	}
	if total := asInt(container["total"]); total > 0 {
		result.Total = total
	} else {
		result.Total = len(result.Items)
	}
	if len(result.Items) == 0 {
		return nil, false
	}
	return &result, true
}

// BuildRestaurantDetail tries to project an arbitrary payload into a Restaurant entity.
func BuildRestaurantDetail(payload any) (*Restaurant, bool) {
	raw := mapFromPayload(payload)
	if len(raw) == 0 {
		return nil, false
	}
	resto, ok := NormalizeRestaurant(raw)
	if !ok {
		return nil, false
	}
	return &resto, true
}

func asString(value any) string {
	if s, ok := value.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func asInt(value any) int {
	switch typed := value.(type) {
	case float64:
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

func asStringSlice(value any) []string {
	var result []string
	switch typed := value.(type) {
	case []interface{}:
		for _, item := range typed {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				result = append(result, strings.TrimSpace(s))
			}
		}
	case []string:
		for _, s := range typed {
			if strings.TrimSpace(s) != "" {
				result = append(result, strings.TrimSpace(s))
			}
		}
	}
	return result
}

func asInterfaceSlice(value any) []interface{} {
	switch typed := value.(type) {
	case []interface{}:
		return typed
	case []map[string]any:
		items := make([]interface{}, 0, len(typed))
		for _, entry := range typed {
			items = append(items, entry)
		}
		return items
	default:
		return nil
	}
}

func parseTime(value any) time.Time {
	s, ok := value.(string)
	if !ok {
		return time.Time{}
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	parsed, err := time.Parse("15:04", s)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func mapFromPayload(value any) map[string]any {
	if value == nil {
		return nil
	}
	if typed, ok := value.(map[string]any); ok {
		// If the payload uses a data envelope we unwrap it.
		if data, ok := typed["data"].(map[string]any); ok {
			return data
		}
		return typed
	}
	return nil
}
