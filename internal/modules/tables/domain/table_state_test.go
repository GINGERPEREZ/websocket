package domain

import "testing"

func TestNormalizeTableState(t *testing.T) {
	cases := []struct {
		name     string
		input    any
		expected TableState
	}{
		{name: "available", input: "available", expected: TableStateAvailable},
		{name: "cleaning", input: " CLEANING ", expected: TableStateCleaning},
		{name: "unknown passthrough", input: "maintenance", expected: TableState("MAINTENANCE")},
		{name: "non string", input: 42, expected: TableStateUnknown},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := NormalizeTableState(tc.input)
			if result != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}
