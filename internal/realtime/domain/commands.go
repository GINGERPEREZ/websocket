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
type ListTablesCommand struct {
	Page      int    `json:"page"`
	Limit     int    `json:"limit"`
	Search    string `json:"search"`
	SortBy    string `json:"sortBy"`
	SortOrder string `json:"sortOrder"`
}

// GetTableCommand represents the client payload for requesting a table detail.
type GetTableCommand struct {
	ID string `json:"id"`
}

// ListReservationsCommand represents the client payload for listing reservations.
type ListReservationsCommand struct {
	Page      int    `json:"page"`
	Limit     int    `json:"limit"`
	Search    string `json:"search"`
	SortBy    string `json:"sortBy"`
	SortOrder string `json:"sortOrder"`
}

// GetReservationCommand represents the client payload for requesting a reservation detail.
type GetReservationCommand struct {
	ID string `json:"id"`
}
