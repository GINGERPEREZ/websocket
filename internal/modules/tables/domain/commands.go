package domain

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
