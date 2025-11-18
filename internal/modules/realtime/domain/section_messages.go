package domain

import (
	"strings"
	"time"
)

// BuildListMessage composes a realtime message for list operations using typed metadata when available.
func BuildListMessage(entity, sectionID string, snapshot *SectionSnapshot, query PagedQuery, at time.Time, extras Metadata) *Message {
	if snapshot == nil {
		return nil
	}

	metadata := query.Metadata(sectionID)
	metadata = mergeInto(metadata, snapshot.ListMetadata)
	metadata = mergeInto(metadata, extras)

	entityName := strings.TrimSpace(entity)
	return &Message{
		Topic:      ListTopic(entityName),
		Entity:     entityName,
		Action:     ActionList,
		ResourceID: strings.TrimSpace(sectionID),
		Metadata:   metadata,
		Data:       snapshot.Payload,
		Timestamp:  at.UTC(),
	}
}

// BuildDetailMessage composes a realtime message for detail operations reusing typed restaurant data.
func BuildDetailMessage(entity, sectionID, restaurantID string, snapshot *SectionSnapshot, at time.Time, extras Metadata) *Message {
	if snapshot == nil {
		return nil
	}

	trimmedSection := strings.TrimSpace(sectionID)
	trimmedResource := strings.TrimSpace(restaurantID)
	metadata := map[string]string{
		"sectionId": trimmedSection,
	}
	entityName := strings.TrimSpace(entity)
	resourceKey := detailResourceKey(entityName)
	if trimmedResource != "" && resourceKey != "" {
		if resourceKey == "sectionId" {
			metadata["resourceSectionId"] = trimmedResource
		} else {
			metadata[resourceKey] = trimmedResource
		}
	}

	metadata = mergeInto(metadata, snapshot.DetailMetadata)
	metadata = mergeInto(metadata, extras)

	return &Message{
		Topic:      DetailTopic(entityName),
		Entity:     entityName,
		Action:     ActionDetail,
		ResourceID: trimmedResource,
		Metadata:   metadata,
		Data:       snapshot.Payload,
		Timestamp:  at.UTC(),
	}
}

func detailResourceKey(entity string) string {
	switch strings.ToLower(strings.TrimSpace(entity)) {
	case "tables":
		return "tableId"
	case "reservations":
		return "reservationId"
	case "reviews":
		return "reviewId"
	case "sections":
		return "sectionId"
	case "objects":
		return "objectId"
	case "menus":
		return "menuId"
	case "dishes":
		return "dishId"
	case "images":
		return "imageId"
	case "section-objects":
		return "sectionObjectId"
	case "payments":
		return "paymentId"
	case "subscriptions":
		return "subscriptionId"
	case "subscription-plans":
		return "subscriptionPlanId"
	case "auth-users":
		return "userId"
	default:
		return "restaurantId"
	}
}

func mergeInto(target map[string]string, extras Metadata) map[string]string {
	if len(extras) == 0 {
		return target
	}
	if target == nil {
		target = map[string]string{}
	}
	for key, value := range extras {
		trimmedKey := strings.TrimSpace(key)
		trimmedValue := strings.TrimSpace(value)
		if trimmedKey == "" || trimmedValue == "" {
			continue
		}
		target[trimmedKey] = trimmedValue
	}
	return target
}
