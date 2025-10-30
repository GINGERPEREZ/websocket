package infrastructure

import (
	"fmt"
	"net/url"
	"strings"

	"mesaYaWs/internal/modules/realtime/application/port"
)

type pathBuilder func(string) (string, error)

type entityEndpoint struct {
	listPathBuilder   pathBuilder
	detailPathBuilder pathBuilder
	sectionQueryKey   string
}

var entityEndpoints = map[string]entityEndpoint{
	"restaurants": {
		listPathBuilder:   staticPathBuilder("/api/v1/public/restaurant"),
		detailPathBuilder: resourcePathBuilder("/api/v1/public/restaurant"),
	},
	"tables": {
		listPathBuilder:   requiredValuePathBuilder("/api/v1/public/table/section/%s"),
		detailPathBuilder: resourcePathBuilder("/api/v1/public/table"),
	},
	"reservations": {
		listPathBuilder:   requiredValuePathBuilder("/api/v1/public/reservations/restaurant/%s"),
		detailPathBuilder: resourcePathBuilder("/api/v1/public/reservations"),
	},
	"reviews": {
		listPathBuilder:   requiredValuePathBuilder("/api/v1/public/review/restaurant/%s"),
		detailPathBuilder: resourcePathBuilder("/api/v1/public/review"),
	},
	"sections": {
		listPathBuilder:   requiredValuePathBuilder("/api/v1/public/section/restaurant/%s"),
		detailPathBuilder: resourcePathBuilder("/api/v1/public/section"),
	},
	"objects": {
		listPathBuilder:   staticPathBuilder("/api/v1/public/object"),
		detailPathBuilder: resourcePathBuilder("/api/v1/public/object"),
	},
	"menus": {
		listPathBuilder:   staticPathBuilder("/api/v1/public/menus"),
		detailPathBuilder: resourcePathBuilder("/api/v1/public/menus"),
	},
	"dishes": {
		listPathBuilder:   staticPathBuilder("/api/v1/public/dishes"),
		detailPathBuilder: resourcePathBuilder("/api/v1/public/dishes"),
	},
	"images": {
		listPathBuilder:   staticPathBuilder("/api/v1/image"),
		detailPathBuilder: resourcePathBuilder("/api/v1/image"),
	},
	"section-objects": {
		listPathBuilder:   staticPathBuilder("/api/v1/admin/section-objects"),
		detailPathBuilder: resourcePathBuilder("/api/v1/admin/section-objects"),
	},
	"payments": {
		listPathBuilder:   requiredValuePathBuilder("/api/v1/restaurant/payments/restaurant/%s"),
		detailPathBuilder: resourcePathBuilder("/api/v1/restaurant/payments"),
	},
	"subscriptions": {
		listPathBuilder:   requiredValuePathBuilder("/api/v1/restaurant/subscriptions/restaurant/%s"),
		detailPathBuilder: resourcePathBuilder("/api/v1/admin/subscriptions"),
	},
	"subscription-plans": {
		listPathBuilder:   staticPathBuilder("/api/v1/public/subscription-plans"),
		detailPathBuilder: resourcePathBuilder("/api/v1/public/subscription-plans"),
	},
	"auth-users": {
		listPathBuilder:   staticPathBuilder("/api/v1/public/users"),
		detailPathBuilder: resourcePathBuilder("/api/v1/public/users"),
	},
}

func staticPathBuilder(path string) pathBuilder {
	trimmed := strings.TrimSpace(path)
	return func(string) (string, error) {
		if trimmed == "" {
			return "", fmt.Errorf("missing path configuration")
		}
		return trimmed, nil
	}
}

func requiredValuePathBuilder(format string) pathBuilder {
	trimmed := strings.TrimSpace(format)
	return func(value string) (string, error) {
		identifier := strings.TrimSpace(value)
		if identifier == "" {
			return "", port.ErrSnapshotNotFound
		}
		return fmt.Sprintf(trimmed, url.PathEscape(identifier)), nil
	}
}

func resourcePathBuilder(base string) pathBuilder {
	trimmed := strings.TrimSpace(base)
	return func(value string) (string, error) {
		identifier := strings.TrimSpace(value)
		if identifier == "" {
			return "", port.ErrSnapshotNotFound
		}
		return strings.TrimRight(trimmed, "/") + "/" + url.PathEscape(identifier), nil
	}
}
