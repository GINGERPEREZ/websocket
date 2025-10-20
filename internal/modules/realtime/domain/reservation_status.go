package domain

import "strings"

// ReservationStatus represents the lifecycle of a reservation as exposed by the REST API.
type ReservationStatus string

const (
	ReservationStatusUnknown   ReservationStatus = ""
	ReservationStatusPending   ReservationStatus = "PENDING"
	ReservationStatusConfirmed ReservationStatus = "CONFIRMED"
	ReservationStatusSeated    ReservationStatus = "SEATED"
	ReservationStatusCompleted ReservationStatus = "COMPLETED"
	ReservationStatusCancelled ReservationStatus = "CANCELLED"
	ReservationStatusNoShow    ReservationStatus = "NO_SHOW"
)

var allowedReservationStatuses = map[string]ReservationStatus{
	string(ReservationStatusPending):   ReservationStatusPending,
	string(ReservationStatusConfirmed): ReservationStatusConfirmed,
	string(ReservationStatusSeated):    ReservationStatusSeated,
	string(ReservationStatusCompleted): ReservationStatusCompleted,
	string(ReservationStatusCancelled): ReservationStatusCancelled,
	string(ReservationStatusNoShow):    ReservationStatusNoShow,
}

// NormalizeReservationStatus returns the canonical ReservationStatus for the given input.
// Unknown statuses are uppercased and returned as-is to avoid data loss.
func NormalizeReservationStatus(value any) ReservationStatus {
	s, ok := value.(string)
	if !ok {
		return ReservationStatusUnknown
	}
	trimmed := strings.ToUpper(strings.TrimSpace(s))
	if trimmed == "" {
		return ReservationStatusUnknown
	}
	if status, ok := allowedReservationStatuses[trimmed]; ok {
		return status
	}
	return ReservationStatus(trimmed)
}
