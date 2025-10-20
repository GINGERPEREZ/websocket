package domain

import "strings"

// RestaurantStatus captures lifecycle states emitted by the REST backoffice service.
type RestaurantStatus string

const (
	RestaurantStatusUnknown  RestaurantStatus = ""
	RestaurantStatusActive   RestaurantStatus = "ACTIVE"
	RestaurantStatusInactive RestaurantStatus = "INACTIVE"
)

var allowedRestaurantStatuses = map[string]RestaurantStatus{
	string(RestaurantStatusActive):   RestaurantStatusActive,
	string(RestaurantStatusInactive): RestaurantStatusInactive,
}

// NormalizeRestaurantStatus converts arbitrary inputs into a canonical restaurant status while
// preserving unexpected custom values coming from upstream.
func NormalizeRestaurantStatus(value any) RestaurantStatus {
	s, ok := value.(string)
	if !ok {
		return RestaurantStatusUnknown
	}
	trimmed := strings.ToUpper(strings.TrimSpace(s))
	if trimmed == "" {
		return RestaurantStatusUnknown
	}
	if status, ok := allowedRestaurantStatuses[trimmed]; ok {
		return status
	}
	return RestaurantStatus(trimmed)
}
