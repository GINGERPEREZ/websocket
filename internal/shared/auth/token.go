package auth

import (
	"net/http"
	"strings"
)

// ExtractBearerToken extracts the JWT token from the Authorization header.
// It handles the "Bearer " prefix and returns an empty string if no token is present.
//
// Example:
//
//	token := ExtractBearerToken(request)
//	if token == "" {
//	    // Handle missing token
//	}
func ExtractBearerToken(r *http.Request) string {
	if r == nil {
		return ""
	}
	return ExtractBearerTokenFromHeader(r.Header.Get("Authorization"))
}

// ExtractBearerTokenFromHeader extracts the JWT token from an Authorization header value.
// It handles the "Bearer " prefix and returns an empty string if no token is present.
//
// Example:
//
//	token := ExtractBearerTokenFromHeader("Bearer eyJhbGciOiJIUzI1NiIs...")
func ExtractBearerTokenFromHeader(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}

	const bearerPrefix = "Bearer "
	if strings.HasPrefix(header, bearerPrefix) {
		return strings.TrimSpace(header[len(bearerPrefix):])
	}

	// Also handle lowercase "bearer" for flexibility
	const bearerPrefixLower = "bearer "
	if strings.HasPrefix(strings.ToLower(header), bearerPrefixLower) {
		return strings.TrimSpace(header[len(bearerPrefixLower):])
	}

	return ""
}

// ExtractTokenFromQuery extracts a token from a URL query parameter.
// Common parameter names: "token", "access_token", "jwt"
func ExtractTokenFromQuery(r *http.Request, paramName string) string {
	if r == nil || r.URL == nil {
		return ""
	}
	return strings.TrimSpace(r.URL.Query().Get(paramName))
}

// ExtractToken attempts to extract a token from multiple sources in order:
// 1. Authorization header (Bearer token)
// 2. Query parameter (configurable name, default "token")
//
// Returns the first non-empty token found.
func ExtractToken(r *http.Request, queryParam string) string {
	// Try Authorization header first
	if token := ExtractBearerToken(r); token != "" {
		return token
	}

	// Fall back to query parameter
	if queryParam == "" {
		queryParam = "token"
	}
	return ExtractTokenFromQuery(r, queryParam)
}
