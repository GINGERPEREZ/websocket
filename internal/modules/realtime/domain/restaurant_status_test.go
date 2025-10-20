package domain

import "testing"

func TestNormalizeRestaurantStatus(t *testing.T) {
	cases := []struct {
		name     string
		input    any
		expected RestaurantStatus
	}{
		{name: "known active", input: "active", expected: RestaurantStatusActive},
		{name: "known inactive", input: " INACTIVE ", expected: RestaurantStatusInactive},
		{name: "unknown passthrough", input: "temporarily_closed", expected: RestaurantStatus("TEMPORARILY_CLOSED")},
		{name: "non string", input: 123, expected: RestaurantStatusUnknown},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := NormalizeRestaurantStatus(tc.input)
			if result != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}
