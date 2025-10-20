package domain

import (
	"net/url"
	"strconv"
	"strings"
)

// PagedQuery encapsulates paging, filtering, and sorting preferences shared by list commands.
type PagedQuery struct {
	Page      int
	Limit     int
	Search    string
	SortBy    string
	SortOrder string
}

// Normalize returns a sanitized copy applying defaults and bounds. defaultSearch is used when
// no explicit search term is provided.
func (q PagedQuery) Normalize(defaultSearch string) PagedQuery {
	normalized := q
	if normalized.Page <= 0 {
		normalized.Page = 1
	}
	if normalized.Limit <= 0 {
		normalized.Limit = 20
	}
	if normalized.Limit > 100 {
		normalized.Limit = 100
	}

	normalized.Search = strings.TrimSpace(normalized.Search)
	if normalized.Search == "" {
		normalized.Search = strings.TrimSpace(defaultSearch)
	}

	normalized.SortBy = strings.TrimSpace(normalized.SortBy)
	normalized.SortOrder = strings.ToUpper(strings.TrimSpace(normalized.SortOrder))

	return normalized
}

// CanonicalKey builds a stable cache key for the combination of paging parameters.
func (q PagedQuery) CanonicalKey() string {
	normalized := q.Normalize("")
	search := strings.ToLower(strings.TrimSpace(normalized.Search))
	sortBy := strings.ToLower(strings.TrimSpace(normalized.SortBy))
	sortOrder := strings.ToUpper(strings.TrimSpace(normalized.SortOrder))

	var builder strings.Builder
	builder.Grow(len(search) + len(sortBy) + len(sortOrder) + 32)
	builder.WriteString("page=")
	builder.WriteString(strconv.Itoa(normalized.Page))
	builder.WriteString("&limit=")
	builder.WriteString(strconv.Itoa(normalized.Limit))
	builder.WriteString("&search=")
	builder.WriteString(search)
	builder.WriteString("&sortBy=")
	builder.WriteString(sortBy)
	builder.WriteString("&sortOrder=")
	builder.WriteString(sortOrder)

	return builder.String()
}

// Metadata converts the query into the metadata map used by websocket messages.
func (q PagedQuery) Metadata(sectionID string) map[string]string {
	normalized := q.Normalize(sectionID)
	metadata := map[string]string{
		"sectionId": strings.TrimSpace(sectionID),
		"page":      strconv.Itoa(normalized.Page),
		"limit":     strconv.Itoa(normalized.Limit),
	}
	if normalized.Search != "" {
		metadata["search"] = normalized.Search
	}
	if normalized.SortBy != "" {
		metadata["sortBy"] = normalized.SortBy
	}
	if normalized.SortOrder != "" {
		metadata["sortOrder"] = normalized.SortOrder
	}
	return metadata
}

// ToURLValues returns normalized URL query parameters ready for REST calls.
func (q PagedQuery) ToURLValues(defaultSearch string) url.Values {
	normalized := q.Normalize(defaultSearch)
	values := url.Values{}
	values.Set("page", strconv.Itoa(normalized.Page))
	values.Set("limit", strconv.Itoa(normalized.Limit))
	if normalized.Search != "" {
		values.Set("q", normalized.Search)
	}
	if normalized.SortBy != "" {
		values.Set("sortBy", normalized.SortBy)
	}
	if normalized.SortOrder != "" {
		values.Set("sortOrder", normalized.SortOrder)
	}
	return values
}
