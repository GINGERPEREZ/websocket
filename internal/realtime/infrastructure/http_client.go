package infrastructure

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"
)

// RESTClient wraps http.Client with base URL handling to avoid duplicating boilerplate in adapters.
type RESTClient struct {
	baseURL string
	client  *http.Client
}

func NewRESTClient(baseURL string, timeout time.Duration, client *http.Client) *RESTClient {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed == "" {
		trimmed = "http://localhost:3000"
	}
	trimmed = strings.TrimRight(trimmed, "/")
	if client == nil {
		client = &http.Client{Timeout: timeoutOrDefault(timeout)}
	} else if timeout > 0 {
		client.Timeout = timeout
	}
	return &RESTClient{baseURL: trimmed, client: client}
}

func (c *RESTClient) NewRequest(ctx context.Context, method, endpoint string, body io.Reader) (*http.Request, error) {
	url := c.baseURL + "/" + strings.TrimLeft(endpoint, "/")
	return http.NewRequestWithContext(ctx, method, url, body)
}

func (c *RESTClient) Do(req *http.Request) (*http.Response, error) {
	return c.client.Do(req)
}

func timeoutOrDefault(value time.Duration) time.Duration {
	if value <= 0 {
		return 10 * time.Second
	}
	return value
}
