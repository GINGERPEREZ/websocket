package domain

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
