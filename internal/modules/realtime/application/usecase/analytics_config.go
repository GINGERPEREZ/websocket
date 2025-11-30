package usecase

import (
	"fmt"
	"net/url"
	"strings"

	"mesaYaWs/internal/modules/realtime/domain"
)

// AnalyticsEndpointConfig describes how to resolve a REST endpoint for analytics data.
type AnalyticsEndpointConfig struct {
	Key                string
	Scope              string
	Entity             string
	PathTemplate       string
	RequiresIdentifier bool
	IdentifierParam    string
	IdentifierAsQuery  bool
	QueryParams        []string
	RequireToken       bool
}

// BuildPath resolves the HTTP path for the configured endpoint.
func (cfg AnalyticsEndpointConfig) BuildPath(identifier string) (string, error) {
	path := strings.TrimSpace(cfg.PathTemplate)
	if path == "" {
		return "", fmt.Errorf("analytics endpoint %s missing path", cfg.Key)
	}
	if cfg.RequiresIdentifier {
		trimmed := strings.TrimSpace(identifier)
		if trimmed == "" {
			return "", ErrAnalyticsMissingIdentifier
		}
		if cfg.IdentifierAsQuery {
			return path, nil
		}
		return fmt.Sprintf(path, url.PathEscape(trimmed)), nil
	}
	return path, nil
}

// RequestFromValues builds an analytics request from HTTP query parameters.
func (cfg AnalyticsEndpointConfig) RequestFromValues(values url.Values) domain.AnalyticsRequest {
	req := domain.AnalyticsRequest{}
	if cfg.IdentifierParam != "" {
		req.Identifier = strings.TrimSpace(values.Get(cfg.IdentifierParam))
	}
	if len(cfg.QueryParams) > 0 {
		req.Query = make(map[string]string, len(cfg.QueryParams))
		for _, key := range cfg.QueryParams {
			trimmedKey := strings.TrimSpace(key)
			if trimmedKey == "" {
				continue
			}
			if value := strings.TrimSpace(values.Get(trimmedKey)); value != "" {
				req.Query[trimmedKey] = value
			}
		}
	}
	return cfg.SanitizeRequest(req)
}

// SanitizeRequest filters the provided request to the allowed parameters for the endpoint.
func (cfg AnalyticsEndpointConfig) SanitizeRequest(req domain.AnalyticsRequest) domain.AnalyticsRequest {
	sanitized := domain.AnalyticsRequest{Identifier: strings.TrimSpace(req.Identifier)}
	if len(cfg.QueryParams) == 0 {
		if len(req.Query) > 0 {
			sanitized.Query = map[string]string{}
		}
		return sanitized
	}

	query := make(map[string]string, len(cfg.QueryParams))
	for _, key := range cfg.QueryParams {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		value, ok := req.Query[trimmedKey]
		if !ok {
			continue
		}
		trimmedValue := strings.TrimSpace(value)
		if trimmedValue == "" {
			continue
		}
		query[trimmedKey] = trimmedValue
	}
	if len(query) > 0 {
		sanitized.Query = query
	}
	return sanitized
}

// DefaultAnalyticsEndpoints returns the default registry of analytics endpoints.
func DefaultAnalyticsEndpoints() map[string]AnalyticsEndpointConfig {
	entries := []AnalyticsEndpointConfig{
		// Public endpoints
		{
			Key:          "analytics-public-users",
			Scope:        "public",
			Entity:       "analytics-public-users",
			PathTemplate: "/api/v1/users/analytics",
			QueryParams:  []string{"startDate"},
		},
		{
			Key:          "analytics-public-dishes",
			Scope:        "public",
			Entity:       "analytics-public-dishes",
			PathTemplate: "/api/v1/dishes/analytics",
			QueryParams:  []string{"startDate"},
		},
		{
			Key:          "analytics-public-menus",
			Scope:        "public",
			Entity:       "analytics-public-menus",
			PathTemplate: "/api/v1/menus/analytics",
			QueryParams:  []string{"startDate"},
		},
		// Restaurant scoped endpoints
		{
			Key:                "analytics-restaurant-users",
			Scope:              "restaurant",
			Entity:             "analytics-restaurant-users",
			PathTemplate:       "/api/v1/users/analytics",
			RequiresIdentifier: true,
			IdentifierParam:    "restaurantId",
			IdentifierAsQuery:  true,
			QueryParams:        []string{"startDate"},
			RequireToken:       true,
		},
		// Admin endpoints
		{
			Key:          "analytics-admin-users",
			Scope:        "admin",
			Entity:       "analytics-admin-users",
			PathTemplate: "/api/v1/users/analytics",
			QueryParams:  []string{"startDate"},
			RequireToken: true,
		},
		{
			Key:          "analytics-admin-restaurants",
			Scope:        "admin",
			Entity:       "analytics-admin-restaurants",
			PathTemplate: "/api/v1/restaurants/analytics",
			QueryParams:  []string{"startDate"},
			RequireToken: true,
		},
		{
			Key:          "analytics-admin-sections",
			Scope:        "admin",
			Entity:       "analytics-admin-sections",
			PathTemplate: "/api/v1/sections/analytics",
			QueryParams:  []string{"restaurantId", "startDate"},
			RequireToken: true,
		},
		{
			Key:          "analytics-admin-tables",
			Scope:        "admin",
			Entity:       "analytics-admin-tables",
			PathTemplate: "/api/v1/tables/analytics",
			QueryParams:  []string{"sectionId", "restaurantId", "startDate"},
			RequireToken: true,
		},
		{
			Key:          "analytics-admin-images",
			Scope:        "admin",
			Entity:       "analytics-admin-images",
			PathTemplate: "/api/v1/images/analytics",
			QueryParams:  []string{"startDate"},
			RequireToken: true,
		},
		{
			Key:          "analytics-admin-objects",
			Scope:        "admin",
			Entity:       "analytics-admin-objects",
			PathTemplate: "/api/v1/objects/analytics",
			QueryParams:  []string{"startDate"},
			RequireToken: true,
		},
		{
			Key:          "analytics-admin-subscriptions",
			Scope:        "admin",
			Entity:       "analytics-admin-subscriptions",
			PathTemplate: "/api/v1/subscriptions/analytics",
			QueryParams:  []string{"startDate"},
			RequireToken: true,
		},
		{
			Key:          "analytics-admin-subscription-plans",
			Scope:        "admin",
			Entity:       "analytics-admin-subscription-plans",
			PathTemplate: "/api/v1/subscription-plans/analytics",
			QueryParams:  []string{"startDate"},
			RequireToken: true,
		},
		{
			Key:          "analytics-admin-reservations",
			Scope:        "admin",
			Entity:       "analytics-admin-reservations",
			PathTemplate: "/api/v1/reservations/analytics",
			QueryParams:  []string{"restaurantId", "startDate"},
			RequireToken: true,
		},
		{
			Key:          "analytics-admin-reviews",
			Scope:        "admin",
			Entity:       "analytics-admin-reviews",
			PathTemplate: "/api/v1/reviews/analytics/stats",
			QueryParams:  []string{"restaurantId", "startDate"},
			RequireToken: true,
		},
		{
			Key:          "analytics-admin-payments",
			Scope:        "admin",
			Entity:       "analytics-admin-payments",
			PathTemplate: "/api/v1/payments/analytics",
			QueryParams:  []string{"restaurantId", "startDate"},
			RequireToken: true,
		},
		{
			Key:          "analytics-admin-auth",
			Scope:        "admin",
			Entity:       "analytics-admin-auth",
			PathTemplate: "/api/v1/auth/analytics",
			QueryParams:  []string{"startDate"},
			RequireToken: true,
		},
	}

	registry := make(map[string]AnalyticsEndpointConfig, len(entries))
	for _, entry := range entries {
		registry[entry.Key] = entry
	}
	return registry
}

// AnalyticsDependencies maps entity changes to analytics endpoints that need refreshing.
var AnalyticsDependencies = map[string][]string{
	"restaurants": {
		"analytics-admin-restaurants",
		"analytics-admin-sections",
		"analytics-admin-tables",
		"analytics-admin-payments",
	},
	"sections": {
		"analytics-admin-sections",
		"analytics-admin-tables",
	},
	"tables": {
		"analytics-admin-tables",
		"analytics-admin-payments",
	},
	"images": {
		"analytics-admin-images",
	},
	"objects": {
		"analytics-admin-objects",
		"analytics-admin-sections",
	},
	"section-objects": {
		"analytics-admin-objects",
		"analytics-admin-sections",
	},
	"subscriptions": {
		"analytics-admin-subscriptions",
		"analytics-admin-subscription-plans",
		"analytics-admin-payments",
	},
	"subscription-plans": {
		"analytics-admin-subscription-plans",
	},
	"reservations": {
		"analytics-admin-reservations",
		"analytics-restaurant-users",
	},
	"reviews": {
		"analytics-admin-reviews",
	},
	"payments": {
		"analytics-admin-payments",
	},
	"auth-users": {
		"analytics-admin-auth",
		"analytics-public-users",
		"analytics-restaurant-users",
	},
	"menus": {
		"analytics-public-menus",
	},
	"dishes": {
		"analytics-public-dishes",
	},
}
