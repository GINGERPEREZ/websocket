package domain

import (
	"strings"
	"time"
)

// Schedule represents the operating hours for a restaurant within a single day.
type Schedule struct {
	Open  time.Time
	Close time.Time
}

// IsZero returns true when both open and close times are unspecified.
func (s Schedule) IsZero() bool {
	return s.Open.IsZero() && s.Close.IsZero()
}

// HasBothTimes indicates whether the schedule has explicit open and close times.
func (s Schedule) HasBothTimes() bool {
	return !s.Open.IsZero() && !s.Close.IsZero()
}

// Duration returns the span between open and close when both values are present.
func (s Schedule) Duration() time.Duration {
	if !s.HasBothTimes() {
		return 0
	}
	return s.Close.Sub(s.Open)
}

// BuildSchedule constructs a schedule enforcing domain invariants.
//   - Accepts values in "HH:MM" format.
//   - Returns false when both values are missing or invalid.
//   - When both values are present, the close time must occur after the open time and
//     within 24 hours to avoid spilling into the next day.
func BuildSchedule(openRaw, closeRaw any) (Schedule, bool) {
	open, openOK := parseScheduleComponent(openRaw)
	close, closeOK := parseScheduleComponent(closeRaw)
	if !openOK && !closeOK {
		return Schedule{}, false
	}

	schedule := Schedule{Open: open, Close: close}
	if openOK && closeOK {
		if !close.After(open) {
			return Schedule{}, false
		}
		if close.Sub(open) > 24*time.Hour {
			return Schedule{}, false
		}
	}
	return schedule, true
}

func parseScheduleComponent(value any) (time.Time, bool) {
	s, ok := value.(string)
	if !ok {
		return time.Time{}, false
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse("15:04", s)
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}
