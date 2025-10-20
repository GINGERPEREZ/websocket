package domain

import (
	"strings"

	"mesaYaWs/internal/shared/normalization"
)

// DayOfWeek encapsulates the allowed opening days using uppercase english names as per REST responses.
type DayOfWeek string

const (
	Monday    DayOfWeek = "MONDAY"
	Tuesday   DayOfWeek = "TUESDAY"
	Wednesday DayOfWeek = "WEDNESDAY"
	Thursday  DayOfWeek = "THURSDAY"
	Friday    DayOfWeek = "FRIDAY"
	Saturday  DayOfWeek = "SATURDAY"
	Sunday    DayOfWeek = "SUNDAY"
)

var allowedDays = map[string]DayOfWeek{
	string(Monday):    Monday,
	string(Tuesday):   Tuesday,
	string(Wednesday): Wednesday,
	string(Thursday):  Thursday,
	string(Friday):    Friday,
	string(Saturday):  Saturday,
	string(Sunday):    Sunday,
}

// NormalizeDaysOpen converts arbitrary slice payloads into a canonical day-of-week list.
func NormalizeDaysOpen(value any) []DayOfWeek {
	items := normalization.AsInterfaceSlice(value)
	if len(items) == 0 {
		return nil
	}

	normalized := make([]DayOfWeek, 0, len(items))
	for _, item := range items {
		day := normalizeDay(item)
		if day != "" {
			normalized = append(normalized, day)
		}
	}
	return normalized
}

func normalizeDay(value any) DayOfWeek {
	switch typed := value.(type) {
	case string:
		key := strings.ToUpper(strings.TrimSpace(typed))
		return allowedDays[key]
	default:
		return ""
	}
}
