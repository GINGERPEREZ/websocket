package domain

import (
	"strconv"
	"strings"
	"time"

	"mesaYaWs/internal/shared/normalization"
)

type metadataExtractor func(map[string]any) Metadata

var listExtractors = map[string]metadataExtractor{
	"restaurants":  restaurantListExtractor,
	"tables":       tableListExtractor,
	"reservations": reservationListExtractor,
}

var detailExtractors = map[string]metadataExtractor{
	"restaurants":  restaurantDetailExtractor,
	"tables":       tableDetailExtractor,
	"reservations": reservationDetailExtractor,
}

// ListMetadataFor inspects the normalized payload and returns metadata specific to the given entity.
func ListMetadataFor(entity string, payload any) Metadata {
	container := normalization.MapFromPayload(payload)
	meta := baseListMetadata(container)
	if extractor, ok := listExtractors[strings.ToLower(strings.TrimSpace(entity))]; ok {
		meta = merge(meta, extractor(container))
	}
	return meta
}

// DetailMetadataFor inspects the normalized payload and returns metadata specific to the given entity.
func DetailMetadataFor(entity string, payload any) Metadata {
	container := normalization.MapFromPayload(payload)
	if extractor, ok := detailExtractors[strings.ToLower(strings.TrimSpace(entity))]; ok {
		return extractor(container)
	}
	return nil
}

func baseListMetadata(container map[string]any) Metadata {
	if len(container) == 0 {
		return nil
	}
	items := extractItemMaps(container, "items")
	if len(items) == 0 {
		return nil
	}
	meta := Metadata{"itemsCount": strconv.Itoa(len(items))}
	if total := normalization.AsInt(container["total"]); total > 0 {
		meta["total"] = strconv.Itoa(total)
	}
	return meta
}

func restaurantListExtractor(_ map[string]any) Metadata {
	return nil
}

func restaurantDetailExtractor(container map[string]any) Metadata {
	entity := extractMap(container, "restaurant")
	if len(entity) == 0 {
		entity = container
	}
	meta := Metadata{}
	if name := normalization.AsString(entity["name"]); name != "" {
		meta["restaurantName"] = name
	}
	if status := normalization.AsString(entity["status"]); status != "" {
		meta["restaurantStatus"] = strings.ToUpper(status)
	}
	openTime := normalization.AsString(entity["openTime"])
	closeTime := normalization.AsString(entity["closeTime"])
	if openTime != "" {
		meta["openTime"] = openTime
	}
	if closeTime != "" {
		meta["closeTime"] = closeTime
	}
	if openTime != "" && closeTime != "" {
		if duration := scheduleDuration(openTime, closeTime); duration > 0 {
			meta["openDurationMinutes"] = strconv.Itoa(duration)
		}
	}
	if subscription := normalization.AsInt(entity["subscriptionId"]); subscription > 0 {
		meta["subscriptionId"] = strconv.Itoa(subscription)
	}
	if days := extractStringSlice(entity["daysOpen"]); len(days) > 0 {
		meta["daysOpen"] = strings.Join(days, ",")
	}

	meta = merge(meta, tableListExtractor(container))
	meta = merge(meta, tableDetailExtractor(container))
	meta = merge(meta, reservationListExtractor(container))
	meta = merge(meta, reservationDetailExtractor(container))
	return meta
}

func tableListExtractor(container map[string]any) Metadata {
	items := extractItemMaps(container, "tables", "items")
	if len(items) == 0 {
		return nil
	}
	summary := struct {
		available int
		reserved  int
		seated    int
		blocked   int
		cleaning  int
	}{}
	for _, item := range items {
		state := stateValue(item)
		switch state {
		case "AVAILABLE":
			summary.available++
		case "RESERVED":
			summary.reserved++
		case "SEATED":
			summary.seated++
		case "BLOCKED":
			summary.blocked++
		case "CLEANING":
			summary.cleaning++
		}
	}
	meta := Metadata{"tablesCount": strconv.Itoa(len(items))}
	if summary.available > 0 {
		meta["tablesAvailable"] = strconv.Itoa(summary.available)
	}
	if summary.reserved > 0 {
		meta["tablesReserved"] = strconv.Itoa(summary.reserved)
	}
	if summary.seated > 0 {
		meta["tablesSeated"] = strconv.Itoa(summary.seated)
	}
	if summary.blocked > 0 {
		meta["tablesBlocked"] = strconv.Itoa(summary.blocked)
	}
	if summary.cleaning > 0 {
		meta["tablesCleaning"] = strconv.Itoa(summary.cleaning)
	}
	return meta
}

func tableDetailExtractor(container map[string]any) Metadata {
	entity := extractMap(container, "table")
	if len(entity) == 0 {
		return nil
	}
	meta := Metadata{}
	if id := normalization.AsString(entity["id"]); id != "" {
		meta["tableId"] = id
	}
	if state := stateValue(entity); state != "" {
		meta["tableState"] = state
	}
	if number := normalization.AsInt(entity["number"]); number > 0 {
		meta["tableNumber"] = strconv.Itoa(number)
	}
	if capacity := normalization.AsInt(entity["capacity"]); capacity > 0 {
		meta["tableCapacity"] = strconv.Itoa(capacity)
	}
	return meta
}

func reservationListExtractor(container map[string]any) Metadata {
	items := extractItemMaps(container, "reservations", "items")
	if len(items) == 0 {
		return nil
	}
	summary := struct {
		pending   int
		confirmed int
		seated    int
		completed int
		cancelled int
		noShow    int
	}{}
	for _, item := range items {
		status := reservationStatusValue(item)
		switch status {
		case "PENDING":
			summary.pending++
		case "CONFIRMED":
			summary.confirmed++
		case "SEATED":
			summary.seated++
		case "COMPLETED":
			summary.completed++
		case "CANCELLED":
			summary.cancelled++
		case "NO_SHOW":
			summary.noShow++
		}
	}
	meta := Metadata{"reservationsCount": strconv.Itoa(len(items))}
	if summary.pending > 0 {
		meta["reservationsPending"] = strconv.Itoa(summary.pending)
	}
	if summary.confirmed > 0 {
		meta["reservationsConfirmed"] = strconv.Itoa(summary.confirmed)
	}
	if summary.seated > 0 {
		meta["reservationsSeated"] = strconv.Itoa(summary.seated)
	}
	if summary.completed > 0 {
		meta["reservationsCompleted"] = strconv.Itoa(summary.completed)
	}
	if summary.cancelled > 0 {
		meta["reservationsCancelled"] = strconv.Itoa(summary.cancelled)
	}
	if summary.noShow > 0 {
		meta["reservationsNoShow"] = strconv.Itoa(summary.noShow)
	}
	return meta
}

func reservationDetailExtractor(container map[string]any) Metadata {
	entity := extractMap(container, "reservation")
	if len(entity) == 0 {
		return nil
	}
	meta := Metadata{}
	if id := normalization.AsString(entity["id"]); id != "" {
		meta["reservationId"] = id
	}
	if status := reservationStatusValue(entity); status != "" {
		meta["reservationStatus"] = status
	}
	if guests := normalization.AsInt(entity["numberOfGuests"]); guests > 0 {
		meta["reservationGuests"] = strconv.Itoa(guests)
	}
	if date := normalization.AsString(entity["reservationDate"]); date != "" {
		meta["reservationDate"] = date
	}
	if timeStr := normalization.AsString(entity["reservationTime"]); timeStr != "" {
		meta["reservationTime"] = timeStr
	}
	if tableID := normalization.AsString(entity["tableId"]); tableID != "" {
		meta["reservationTableId"] = tableID
	}
	return meta
}

func extractItemMaps(container map[string]any, keys ...string) []map[string]any {
	for _, key := range keys {
		if raw, ok := container[key]; ok {
			return coerceToMapSlice(raw)
		}
	}
	return nil
}

func coerceToMapSlice(value any) []map[string]any {
	items := normalization.AsInterfaceSlice(value)
	if len(items) == 0 {
		return nil
	}
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		if mapped := normalization.MapFromPayload(item); len(mapped) > 0 {
			result = append(result, mapped)
			continue
		}
		if mapped, ok := item.(map[string]any); ok {
			result = append(result, mapped)
		}
	}
	return result
}

func extractMap(container map[string]any, keys ...string) map[string]any {
	if len(container) == 0 {
		return nil
	}
	for _, key := range keys {
		if raw, ok := container[key]; ok {
			if mapped := normalization.MapFromPayload(raw); len(mapped) > 0 {
				return mapped
			}
			if mapped, ok := raw.(map[string]any); ok {
				return mapped
			}
		}
	}
	return nil
}

func extractStringSlice(value any) []string {
	items := normalization.AsInterfaceSlice(value)
	if len(items) == 0 {
		return nil
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.(string); ok {
			trimmed := strings.ToUpper(strings.TrimSpace(s))
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
	}
	return result
}

func stateValue(raw map[string]any) string {
	if raw == nil {
		return ""
	}
	if state := strings.ToUpper(strings.TrimSpace(normalization.AsString(raw["state"]))); state != "" {
		return state
	}
	if status := strings.ToUpper(strings.TrimSpace(normalization.AsString(raw["status"]))); status != "" {
		return status
	}
	return ""
}

func reservationStatusValue(raw map[string]any) string {
	if raw == nil {
		return ""
	}
	if status := strings.ToUpper(strings.TrimSpace(normalization.AsString(raw["status"]))); status != "" {
		return status
	}
	if state := strings.ToUpper(strings.TrimSpace(normalization.AsString(raw["state"]))); state != "" {
		return state
	}
	return ""
}

func scheduleDuration(openTime, closeTime string) int {
	open, errOpen := time.Parse("15:04", strings.TrimSpace(openTime))
	close, errClose := time.Parse("15:04", strings.TrimSpace(closeTime))
	if errOpen != nil || errClose != nil {
		return 0
	}
	if !close.After(open) {
		return 0
	}
	return int(close.Sub(open).Minutes())
}
