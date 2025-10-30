package usecase

import "testing"

func TestNormalizeAnalyticsDependencyEntity(t *testing.T) {
	cases := map[string]string{
		"Restaurant":       "restaurants",
		"section":          "sections",
		"Tables":           "tables",
		"image":            "images",
		"OBJECT":           "objects",
		"Section_Object":   "section-objects",
		"SectionObject":    "section-objects",
		"subscription":     "subscriptions",
		"SubscriptionPlan": "subscription-plans",
		"Reservation":      "reservations",
		"Review":           "reviews",
		"Payment":          "payments",
		"auth":             "auth-users",
		"Auth_Users":       "auth-users",
		"Menu":             "menus",
		"DISH":             "dishes",
	}

	for input, expected := range cases {
		if got := normalizeAnalyticsDependencyEntity(input); got != expected {
			t.Fatalf("normalizeAnalyticsDependencyEntity(%q) = %q, expected %q", input, got, expected)
		}
	}
}
