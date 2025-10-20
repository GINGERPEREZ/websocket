package domain

// ListRestaurantsCommand represents the client payload for listing restaurants.
type ListRestaurantsCommand struct {
	Page      int    `json:"page"`
	Limit     int    `json:"limit"`
	Search    string `json:"search"`
	SortBy    string `json:"sortBy"`
	SortOrder string `json:"sortOrder"`
}

// GetRestaurantCommand represents the client payload for requesting a restaurant detail.
type GetRestaurantCommand struct {
	ID string `json:"id"`
}

// ListTablesCommand represents the client payload for listing tables in a section.
// Auxiliary commands for related bounded contexts live in their respective
// modules (tables, reservations, etc.).
