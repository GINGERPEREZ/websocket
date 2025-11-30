package httputil

import (
	"context"
	"errors"
	"net/http"
)

// HTTPErrorInfo contains the HTTP status code and message for an error.
type HTTPErrorInfo struct {
	Status  int
	Message string
}

// ErrorMapping represents a single error to HTTP status/message mapping.
type ErrorMapping struct {
	Error   error
	Status  int
	Message string
}

// ErrorMapper maps domain errors to HTTP status codes and messages.
// It provides a centralized way to handle error mapping across handlers.
type ErrorMapper struct {
	mappings       []ErrorMapping
	defaultStatus  int
	defaultMessage string
}

// NewErrorMapper creates a new ErrorMapper with default settings.
func NewErrorMapper() *ErrorMapper {
	return &ErrorMapper{
		mappings:       make([]ErrorMapping, 0),
		defaultStatus:  http.StatusInternalServerError,
		defaultMessage: "internal server error",
	}
}

// WithMapping adds an error mapping to the mapper.
func (m *ErrorMapper) WithMapping(err error, status int, message string) *ErrorMapper {
	m.mappings = append(m.mappings, ErrorMapping{
		Error:   err,
		Status:  status,
		Message: message,
	})
	return m
}

// WithDefault sets the default status and message for unmatched errors.
func (m *ErrorMapper) WithDefault(status int, message string) *ErrorMapper {
	m.defaultStatus = status
	m.defaultMessage = message
	return m
}

// Map converts an error to HTTP status and message.
func (m *ErrorMapper) Map(err error) HTTPErrorInfo {
	if err == nil {
		return HTTPErrorInfo{Status: http.StatusOK, Message: ""}
	}

	// Check for context errors first
	if errors.Is(err, context.DeadlineExceeded) {
		return HTTPErrorInfo{Status: http.StatusGatewayTimeout, Message: "request timeout"}
	}
	if errors.Is(err, context.Canceled) {
		return HTTPErrorInfo{Status: http.StatusServiceUnavailable, Message: "request cancelled"}
	}

	// Check registered mappings
	for _, mapping := range m.mappings {
		if errors.Is(err, mapping.Error) {
			return HTTPErrorInfo{Status: mapping.Status, Message: mapping.Message}
		}
	}

	return HTTPErrorInfo{Status: m.defaultStatus, Message: m.defaultMessage}
}

// Common error categories for convenience
var (
	// BadRequestErrors maps common bad request errors
	BadRequestErrors = []ErrorMapping{
		// Add domain errors here as needed
	}

	// UnauthorizedErrors maps common unauthorized errors
	UnauthorizedErrors = []ErrorMapping{
		// Add auth errors here as needed
	}

	// ForbiddenErrors maps common forbidden errors
	ForbiddenErrors = []ErrorMapping{
		// Add permission errors here as needed
	}

	// NotFoundErrors maps common not found errors
	NotFoundErrors = []ErrorMapping{
		// Add not found errors here as needed
	}
)

// QuickMap is a convenience function for quick error mapping without creating a mapper.
func QuickMap(err error, mappings ...ErrorMapping) HTTPErrorInfo {
	if err == nil {
		return HTTPErrorInfo{Status: http.StatusOK, Message: ""}
	}

	// Check context errors
	if errors.Is(err, context.DeadlineExceeded) {
		return HTTPErrorInfo{Status: http.StatusGatewayTimeout, Message: "request timeout"}
	}
	if errors.Is(err, context.Canceled) {
		return HTTPErrorInfo{Status: http.StatusServiceUnavailable, Message: "request cancelled"}
	}

	// Check provided mappings
	for _, mapping := range mappings {
		if errors.Is(err, mapping.Error) {
			return HTTPErrorInfo{Status: mapping.Status, Message: mapping.Message}
		}
	}

	return HTTPErrorInfo{Status: http.StatusInternalServerError, Message: "internal server error"}
}
