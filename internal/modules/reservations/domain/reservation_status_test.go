package domain

import "testing"

func TestNormalizeReservationStatus(t *testing.T) {
	cases := []struct {
		name     string
		input    any
		expected ReservationStatus
	}{
		{name: "pending", input: " pending ", expected: ReservationStatusPending},
		{name: "confirmed uppercase", input: "CONFIRMED", expected: ReservationStatusConfirmed},
		{name: "unknown passthrough", input: "delayed", expected: ReservationStatus("DELAYED")},
		{name: "non string", input: nil, expected: ReservationStatusUnknown},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := NormalizeReservationStatus(tc.input)
			if result != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}
