package domain

import "strings"

// Restaurant represents the restaurant aggregate root exposed over realtime.
type Restaurant struct {
	ID            string
	Name          string
	Description   string
	Location      string
	Schedule      Schedule
	DaysOpen      []DayOfWeek
	Status        RestaurantStatus
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
		DaysOpen:      NormalizeDaysOpen(raw["daysOpen"]),
		Status:        NormalizeRestaurantStatus(raw["status"]),
		TotalCapacity: asInt(raw["totalCapacity"]),
		Subscription:  asInt(raw["subscriptionId"]),
		ImageID:       asInt(raw["imageId"]),
	}
	if schedule, ok := BuildSchedule(raw["openTime"], raw["closeTime"]); ok {
		resto.Schedule = schedule
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

