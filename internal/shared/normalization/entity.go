package normalization

import "strings"

// entityAliases maps various entity name formats to their canonical form.
// This centralizes all entity normalization logic that was previously
// duplicated across multiple files.
var entityAliases = map[string]string{
	// Empty/default
	"":        "",
	"-":       "",
	"default": "",

	// Restaurants
	"restaurant":  "restaurants",
	"restaurants": "restaurants",

	// Tables
	"table":  "tables",
	"tables": "tables",

	// Reservations
	"reservation":  "reservations",
	"reservations": "reservations",

	// Sections
	"section":  "sections",
	"sections": "sections",

	// Reviews
	"review":  "reviews",
	"reviews": "reviews",

	// Section Objects (multiple formats)
	"sectionobject":     "section-objects",
	"sectionobjects":    "section-objects",
	"section-object":    "section-objects",
	"section-objects":   "section-objects",
	"section_object":    "section-objects",
	"section_objects":   "section-objects",

	// Objects
	"object":  "objects",
	"objects": "objects",

	// Menus
	"menu":  "menus",
	"menus": "menus",

	// Dishes
	"dish":   "dishes",
	"dishes": "dishes",

	// Images
	"image":  "images",
	"images": "images",

	// Payments
	"payment":  "payments",
	"payments": "payments",

	// Subscriptions
	"subscription":  "subscriptions",
	"subscriptions": "subscriptions",

	// Subscription Plans
	"subscription-plan":   "subscription-plans",
	"subscription-plans":  "subscription-plans",
	"subscription_plan":   "subscription-plans",
	"subscription_plans":  "subscription-plans",
	"subscriptionplan":    "subscription-plans",
	"subscriptionplans":   "subscription-plans",

	// Users/Auth (multiple formats)
	"user":       "users",
	"users":      "users",
	"auth":       "users",
	"auth-user":  "users",
	"auth-users": "users",
	"auth_user":  "users",
	"auth_users": "users",
	"authuser":   "users",
	"authusers":  "users",
	"owner":      "users",
	"owners":     "users",

	// Owner Upgrades
	"owner-upgrade":   "owner-upgrades",
	"owner-upgrades":  "owner-upgrades",
	"owner_upgrade":   "owner-upgrades",
	"owner_upgrades":  "owner-upgrades",
	"ownerupgrade":    "owner-upgrades",
	"ownerupgrades":   "owner-upgrades",
}

// NormalizeEntity converts various entity name formats to their canonical form.
// This function handles singular/plural forms, different separators (-, _),
// and common aliases.
//
// Example:
//
//	NormalizeEntity("restaurant") => "restaurants"
//	NormalizeEntity("SECTION_OBJECT") => "section-objects"
//	NormalizeEntity("auth-user") => "users"
func NormalizeEntity(raw string) string {
	// Normalize the input: lowercase, trim, replace underscores with hyphens
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	normalized := strings.ReplaceAll(trimmed, "_", "-")

	// Check for known aliases
	if canonical, found := entityAliases[normalized]; found {
		return canonical
	}

	// Also check without underscore replacement for backwards compatibility
	if canonical, found := entityAliases[trimmed]; found {
		return canonical
	}

	// Return the normalized form if no alias found
	return normalized
}

// IsValidEntity checks if the given entity name is a known entity type.
func IsValidEntity(raw string) bool {
	normalized := NormalizeEntity(raw)
	if normalized == "" {
		return false
	}

	// Check if the normalized form is a known canonical entity
	validEntities := map[string]bool{
		"restaurants":        true,
		"tables":             true,
		"reservations":       true,
		"sections":           true,
		"reviews":            true,
		"section-objects":    true,
		"objects":            true,
		"menus":              true,
		"dishes":             true,
		"images":             true,
		"payments":           true,
		"subscriptions":      true,
		"subscription-plans": true,
		"users":              true,
		"owner-upgrades":     true,
	}

	return validEntities[normalized]
}

// GetAllValidEntities returns a list of all valid canonical entity names.
func GetAllValidEntities() []string {
	return []string{
		"restaurants",
		"tables",
		"reservations",
		"sections",
		"reviews",
		"section-objects",
		"objects",
		"menus",
		"dishes",
		"images",
		"payments",
		"subscriptions",
		"subscription-plans",
		"users",
		"owner-upgrades",
	}
}
