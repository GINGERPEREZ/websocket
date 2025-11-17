package domain

// ListEntityCommand represents the generic payload for paginated list commands.
type ListEntityCommand struct {
	Page      int               `json:"page"`
	Limit     int               `json:"limit"`
	Search    string            `json:"search"`
	SortBy    string            `json:"sortBy"`
	SortOrder string            `json:"sortOrder"`
	Filters   map[string]string `json:"filters,omitempty"`
}

// GetEntityCommand represents the generic payload for retrieving a resource by identifier.
type GetEntityCommand struct {
	ID string `json:"id"`
}

// AnalyticsCommand represents the payload for requesting analytics data via websocket commands.
type AnalyticsCommand struct {
	Identifier string            `json:"identifier,omitempty"`
	Query      map[string]string `json:"query,omitempty"`
}
