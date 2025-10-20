package domain

import (
	"testing"
)

func TestNormalizeDaysOpen(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected []DayOfWeek
	}{
		{
			name:  "mixed casing and spacing",
			input: []any{"monday", "  Tuesday", "SATURDAY"},
			expected: []DayOfWeek{
				Monday,
				Tuesday,
				Saturday,
			},
		},
		{
			name:     "invalid entries filtered",
			input:    []any{"", "holiday", 123, nil},
			expected: nil,
		},
		{
			name:     "non slice input returns nil",
			input:    "monday",
			expected: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := NormalizeDaysOpen(test.input)
			if len(result) != len(test.expected) {
				t.Fatalf("expected %d items, got %d", len(test.expected), len(result))
			}
			for i := range result {
				if result[i] != test.expected[i] {
					t.Fatalf("expected %v at position %d, got %v", test.expected[i], i, result[i])
				}
			}
		})
	}
}
