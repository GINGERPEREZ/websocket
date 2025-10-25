package domain

import "strings"

// Metadata encapsulates the key-value pairs attached to realtime snapshots.
// Values are stored as strings to simplify transport across websocket boundaries.
type Metadata map[string]string

// merge copies non-empty entries from the source metadata into the destination map.
func merge(into Metadata, values Metadata) Metadata {
	if len(values) == 0 {
		return into
	}
	if into == nil {
		into = Metadata{}
	}
	for key, value := range values {
		trimmedKey := strings.TrimSpace(key)
		trimmedValue := strings.TrimSpace(value)
		if trimmedKey == "" || trimmedValue == "" {
			continue
		}
		into[trimmedKey] = trimmedValue
	}
	return into
}

// SectionSnapshot holds the full state returned by the REST API.
// Payload remains untyped so the adapter can forward whatever structure Nest emits.
// Entity-specific metadata extracted from the payload is captured in the list/detail maps.
type SectionSnapshot struct {
	Payload        any
	ListMetadata   Metadata
	DetailMetadata Metadata
}

// MergeListMetadata enriches the snapshot with additional information for list broadcasts.
func (s *SectionSnapshot) MergeListMetadata(values Metadata) {
	s.ListMetadata = merge(s.ListMetadata, values)
}

// MergeDetailMetadata enriches the snapshot with additional information for detail broadcasts.
func (s *SectionSnapshot) MergeDetailMetadata(values Metadata) {
	s.DetailMetadata = merge(s.DetailMetadata, values)
}
