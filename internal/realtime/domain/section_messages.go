package domain

import (
	"strconv"
	"strings"
	"time"
)

// BuildListMessage composes a realtime message for list operations using typed metadata when available.
func BuildListMessage(entity, sectionID string, snapshot *SectionSnapshot, query PagedQuery, at time.Time) *Message {
	if snapshot == nil {
		return nil
	}

	metadata := query.Metadata(sectionID)
	if metadata == nil {
		metadata = map[string]string{}
	}
	if snapshot.RestaurantList != nil {
		metadata["itemsCount"] = strconv.Itoa(len(snapshot.RestaurantList.Items))
		if snapshot.RestaurantList.Total > 0 {
			metadata["total"] = strconv.Itoa(snapshot.RestaurantList.Total)
		}
	}

	entityName := strings.TrimSpace(entity)
	return &Message{
		Topic:      entityName + ".list",
		Entity:     entityName,
		Action:     "list",
		ResourceID: strings.TrimSpace(sectionID),
		Metadata:   metadata,
		Data:       snapshot.Payload,
		Timestamp:  at.UTC(),
	}
}

// BuildDetailMessage composes a realtime message for detail operations reusing typed restaurant data.
func BuildDetailMessage(entity, sectionID, restaurantID string, snapshot *SectionSnapshot, at time.Time) *Message {
	if snapshot == nil {
		return nil
	}

	trimmedSection := strings.TrimSpace(sectionID)
	trimmedRestaurant := strings.TrimSpace(restaurantID)
	metadata := map[string]string{
		"sectionId": trimmedSection,
	}
	if trimmedRestaurant != "" {
		metadata["restaurantId"] = trimmedRestaurant
	}

	if snapshot.Restaurant != nil {
		if name := strings.TrimSpace(snapshot.Restaurant.Name); name != "" {
			metadata["restaurantName"] = name
		}
		if status := strings.TrimSpace(string(snapshot.Restaurant.Status)); status != "" {
			metadata["restaurantStatus"] = status
		}
		schedule := snapshot.Restaurant.Schedule
		if !schedule.Open.IsZero() {
			metadata["openTime"] = schedule.Open.Format("15:04")
		}
		if !schedule.Close.IsZero() {
			metadata["closeTime"] = schedule.Close.Format("15:04")
		}
		if schedule.HasBothTimes() {
			metadata["openDurationMinutes"] = strconv.Itoa(int(schedule.Duration().Minutes()))
		}
		if snapshot.Restaurant.Subscription > 0 {
			metadata["subscriptionId"] = strconv.Itoa(snapshot.Restaurant.Subscription)
		}
		if len(snapshot.Restaurant.DaysOpen) > 0 {
			names := make([]string, 0, len(snapshot.Restaurant.DaysOpen))
			for _, day := range snapshot.Restaurant.DaysOpen {
				if trimmed := strings.TrimSpace(string(day)); trimmed != "" {
					names = append(names, trimmed)
				}
			}
			if len(names) > 0 {
				metadata["daysOpen"] = strings.Join(names, ",")
			}
		}
	}

	entityName := strings.TrimSpace(entity)
	return &Message{
		Topic:      entityName + ".detail",
		Entity:     entityName,
		Action:     "detail",
		ResourceID: trimmedRestaurant,
		Metadata:   metadata,
		Data:       snapshot.Payload,
		Timestamp:  at.UTC(),
	}
}
