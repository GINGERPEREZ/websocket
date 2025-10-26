package domain

import (
    "strings"
    "time"
)

// AnalyticsRequest encapsulates the parameters used when fetching analytics data from the REST API.
type AnalyticsRequest struct {
    Identifier string            `json:"identifier,omitempty"`
    Query      map[string]string `json:"query,omitempty"`
}

// clone creates a deep copy of the analytics request, ensuring maps are duplicated.
func (r AnalyticsRequest) Clone() AnalyticsRequest {
    cloned := AnalyticsRequest{Identifier: strings.TrimSpace(r.Identifier)}
    if len(r.Query) == 0 {
        return cloned
    }
    cloned.Query = make(map[string]string, len(r.Query))
    for key, value := range r.Query {
        trimmedKey := strings.TrimSpace(key)
        trimmedValue := strings.TrimSpace(value)
        if trimmedKey == "" || trimmedValue == "" {
            continue
        }
        cloned.Query[trimmedKey] = trimmedValue
    }
    return cloned
}

// AnalyticsSnapshot represents the decoded payload returned by the analytics REST endpoints.
type AnalyticsSnapshot struct {
    Payload  any
    Metadata Metadata
}

// MergeMetadata enriches the analytics snapshot with additional metadata.
func (s *AnalyticsSnapshot) MergeMetadata(values Metadata) {
    s.Metadata = merge(s.Metadata, values)
}

// BuildAnalyticsMessage composes a realtime message for analytics fetch operations.
func BuildAnalyticsMessage(entity, scope string, snapshot *AnalyticsSnapshot, request AnalyticsRequest, at time.Time) *Message {
    if snapshot == nil {
        return nil
    }

    meta := Metadata{}
    trimmedScope := strings.TrimSpace(scope)
    if trimmedScope != "" {
        meta["scope"] = trimmedScope
    }

    identifier := strings.TrimSpace(request.Identifier)
    if identifier != "" {
        meta["identifier"] = identifier
    }

    for key, value := range request.Query {
        trimmedKey := strings.TrimSpace(key)
        trimmedValue := strings.TrimSpace(value)
        if trimmedKey == "" || trimmedValue == "" {
            continue
        }
        meta["query."+trimmedKey] = trimmedValue
    }

    meta = merge(meta, snapshot.Metadata)

    entityName := strings.TrimSpace(entity)
    return &Message{
        Topic:      SnapshotTopic(entityName),
        Entity:     entityName,
        Action:     ActionSnapshot,
        ResourceID: identifier,
        Metadata:   map[string]string(meta),
        Data:       snapshot.Payload,
        Timestamp:  at.UTC(),
    }
}
