package domain

// ListEntityCommand represents the generic payload for paginated list commands.
type ListEntityCommand struct {
	Page      int    `json:"page"`
	Limit     int    `json:"limit"`
	Search    string `json:"search"`
	SortBy    string `json:"sortBy"`
	SortOrder string `json:"sortOrder"`
}

// GetEntityCommand represents the generic payload for retrieving a resource by identifier.
type GetEntityCommand struct {
	ID string `json:"id"`
}
