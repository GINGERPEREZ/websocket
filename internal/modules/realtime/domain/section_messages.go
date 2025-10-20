package domain

import (
	"strconv"
	"strings"
	"time"

	reservations "mesaYaWs/internal/modules/reservations/domain"
	tables "mesaYaWs/internal/modules/tables/domain"
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
	if snapshot.TableList != nil && len(snapshot.TableList.Items) > 0 {
		metadata["tablesCount"] = strconv.Itoa(len(snapshot.TableList.Items))
	}
	if snapshot.ReservationList != nil && len(snapshot.ReservationList.Items) > 0 {
		metadata["reservationsCount"] = strconv.Itoa(len(snapshot.ReservationList.Items))
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
	trimmedResource := strings.TrimSpace(restaurantID)
	metadata := map[string]string{
		"sectionId": trimmedSection,
	}
	entityName := strings.TrimSpace(entity)
	resourceKey := detailResourceKey(entityName)
	if trimmedResource != "" && resourceKey != "" {
		metadata[resourceKey] = trimmedResource
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
	if snapshot.TableList != nil && len(snapshot.TableList.Items) > 0 {
		summary := summarizeTableStates(snapshot.TableList.Items)
		metadata["tablesCount"] = strconv.Itoa(len(snapshot.TableList.Items))
		if summary.available > 0 {
			metadata["tablesAvailable"] = strconv.Itoa(summary.available)
		}
		if summary.reserved > 0 {
			metadata["tablesReserved"] = strconv.Itoa(summary.reserved)
		}
		if summary.seated > 0 {
			metadata["tablesSeated"] = strconv.Itoa(summary.seated)
		}
		if summary.blocked > 0 {
			metadata["tablesBlocked"] = strconv.Itoa(summary.blocked)
		}
		if summary.cleaning > 0 {
			metadata["tablesCleaning"] = strconv.Itoa(summary.cleaning)
		}
	}
	if snapshot.Table != nil {
		if id := strings.TrimSpace(snapshot.Table.ID); id != "" {
			metadata["tableId"] = id
		}
		if state := strings.TrimSpace(string(snapshot.Table.State)); state != "" {
			metadata["tableState"] = state
		}
		if snapshot.Table.Number > 0 {
			metadata["tableNumber"] = strconv.Itoa(snapshot.Table.Number)
		}
		if snapshot.Table.Capacity > 0 {
			metadata["tableCapacity"] = strconv.Itoa(snapshot.Table.Capacity)
		}
	}
	if snapshot.ReservationList != nil && len(snapshot.ReservationList.Items) > 0 {
		summary := summarizeReservationStatuses(snapshot.ReservationList.Items)
		metadata["reservationsCount"] = strconv.Itoa(len(snapshot.ReservationList.Items))
		if summary.pending > 0 {
			metadata["reservationsPending"] = strconv.Itoa(summary.pending)
		}
		if summary.confirmed > 0 {
			metadata["reservationsConfirmed"] = strconv.Itoa(summary.confirmed)
		}
		if summary.seated > 0 {
			metadata["reservationsSeated"] = strconv.Itoa(summary.seated)
		}
		if summary.completed > 0 {
			metadata["reservationsCompleted"] = strconv.Itoa(summary.completed)
		}
		if summary.cancelled > 0 {
			metadata["reservationsCancelled"] = strconv.Itoa(summary.cancelled)
		}
		if summary.noShow > 0 {
			metadata["reservationsNoShow"] = strconv.Itoa(summary.noShow)
		}
	}
	if snapshot.Reservation != nil {
		if id := strings.TrimSpace(snapshot.Reservation.ID); id != "" {
			metadata["reservationId"] = id
		}
		if status := strings.TrimSpace(string(snapshot.Reservation.Status)); status != "" {
			metadata["reservationStatus"] = status
		}
		if snapshot.Reservation.Guests > 0 {
			metadata["reservationGuests"] = strconv.Itoa(snapshot.Reservation.Guests)
		}
		if date := strings.TrimSpace(snapshot.Reservation.ReservationDate); date != "" {
			metadata["reservationDate"] = date
		}
		if timeStr := strings.TrimSpace(snapshot.Reservation.ReservationTime); timeStr != "" {
			metadata["reservationTime"] = timeStr
		}
		if tableID := strings.TrimSpace(snapshot.Reservation.TableID); tableID != "" {
			metadata["reservationTableId"] = tableID
		}
	}

	return &Message{
		Topic:      entityName + ".detail",
		Entity:     entityName,
		Action:     "detail",
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
	default:
		return "restaurantId"
	}
}

type tableStateSummary struct {
	available int
	reserved  int
	seated    int
	blocked   int
	cleaning  int
}

func summarizeTableStates(items []tables.Table) tableStateSummary {
	summary := tableStateSummary{}
	for _, table := range items {
		switch table.State {
		case tables.TableStateAvailable:
			summary.available++
		case tables.TableStateReserved:
			summary.reserved++
		case tables.TableStateSeated:
			summary.seated++
		case tables.TableStateBlocked:
			summary.blocked++
		case tables.TableStateCleaning:
			summary.cleaning++
		}
	}
	return summary
}

type reservationStatusSummary struct {
	pending   int
	confirmed int
	seated    int
	completed int
	cancelled int
	noShow    int
}

func summarizeReservationStatuses(items []reservations.Reservation) reservationStatusSummary {
	summary := reservationStatusSummary{}
	for _, reservation := range items {
		switch reservation.Status {
		case reservations.ReservationStatusPending:
			summary.pending++
		case reservations.ReservationStatusConfirmed:
			summary.confirmed++
		case reservations.ReservationStatusSeated:
			summary.seated++
		case reservations.ReservationStatusCompleted:
			summary.completed++
		case reservations.ReservationStatusCancelled:
			summary.cancelled++
		case reservations.ReservationStatusNoShow:
			summary.noShow++
		}
	}
	return summary
}
