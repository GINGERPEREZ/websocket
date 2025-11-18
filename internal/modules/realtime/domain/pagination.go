package domain

import (
	"net/url"
	"sort"
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
	Filters   map[string]string
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

	if len(normalized.Filters) > 0 {
		normalized.Filters = sanitizeFilters(normalized.Filters)
	}

	return normalized
}

// CanonicalKey builds a stable cache key for the combination of paging parameters.
func (q PagedQuery) CanonicalKey() string {
	normalized := q.Normalize("")
	search := strings.ToLower(strings.TrimSpace(normalized.Search))
	sortBy := strings.ToLower(strings.TrimSpace(normalized.SortBy))
	sortOrder := strings.ToUpper(strings.TrimSpace(normalized.SortOrder))
	filtersKey := canonicalFiltersKey(normalized.Filters)

	var builder strings.Builder
	builder.Grow(len(search) + len(sortBy) + len(sortOrder) + len(filtersKey) + 32)
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
	if filtersKey != "" {
		builder.WriteString("&filters=")
		builder.WriteString(filtersKey)
	}

	return builder.String()
}

// Metadata converts the query into the metadata map used by websocket messages.
func (q PagedQuery) Metadata(sectionID string) map[string]string {
	normalized := q.Normalize("")
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
	for key, value := range normalized.Filters {
		metadata[key] = value
	}
	return metadata
}

// ToURLValues returns normalized URL query parameters ready for REST calls.
func (q PagedQuery) ToURLValues(defaultSearch string) url.Values {
	normalized := q.Normalize(strings.TrimSpace(defaultSearch))
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
	for key, value := range normalized.Filters {
		values.Set(key, value)
	}
	return values
}

func sanitizeFilters(filters map[string]string) map[string]string {
	if len(filters) == 0 {
		return nil
	}
	sanitized := make(map[string]string, len(filters))
	for key, value := range filters {
		trimmedKey := strings.TrimSpace(key)
		trimmedValue := strings.TrimSpace(value)
		if trimmedKey == "" || trimmedValue == "" {
			continue
		}
		sanitized[strings.ToLower(trimmedKey)] = trimmedValue
	}
	if len(sanitized) == 0 {
		return nil
	}
	return sanitized
}

func canonicalFiltersKey(filters map[string]string) string {
	if len(filters) == 0 {
		return ""
	}
	keys := make([]string, 0, len(filters))
	for key := range filters {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var builder strings.Builder
	for index, key := range keys {
		if index > 0 {
			builder.WriteString(";")
		}
		builder.WriteString(key)
		builder.WriteString("=")
		builder.WriteString(filters[key])
	}
	return builder.String()
}
