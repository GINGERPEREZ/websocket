package port

import (
    "context"
    "errors"

    "mesaYaWs/internal/modules/realtime/domain"
)

var (
    // ErrAnalyticsForbidden indicates the REST API rejected the analytics request due to authorization.
    ErrAnalyticsForbidden = errors.New("analytics fetch forbidden")
    // ErrAnalyticsNotFound indicates the requested analytics resource was not found.
    ErrAnalyticsNotFound = errors.New("analytics fetch not found")
    // ErrAnalyticsUnsupported is returned when an analytics endpoint has not been configured.
    ErrAnalyticsUnsupported = errors.New("analytics endpoint unsupported")
)

// AnalyticsFetcher retrieves analytics payloads from the REST API.
type AnalyticsFetcher interface {
    Fetch(ctx context.Context, token, path string, query map[string]string) (*domain.AnalyticsSnapshot, error)
}
