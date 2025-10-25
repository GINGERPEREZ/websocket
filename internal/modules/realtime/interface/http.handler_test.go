package transport

import "testing"

func TestNormalizeEntity(t *testing.T) {
	cases := map[string]string{
		"":              "",
		"restaurant":    "restaurants",
		"restaurants":   "restaurants",
		" Restaurant ":  "restaurants",
		"table":         "tables",
		"tables":        "tables",
		"reservation":   "reservations",
		"reservations":  "reservations",
		" users ":       "users",
		"custom-entity": "custom-entity",
	}

	for input, expected := range cases {
		actual := normalizeEntity(input)
		if actual != expected {
			t.Fatalf("normalizeEntity(%q) expected %q got %q", input, expected, actual)
		}
	}
}
