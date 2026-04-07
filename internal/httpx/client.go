package httpx

import (
	"net/http"
	"time"
)

const defaultTimeout = 15 * time.Second

// DefaultTimeout returns the built-in HTTP client timeout used by adapters.
func DefaultTimeout() time.Duration {
	return defaultTimeout
}

// NewClient returns the default HTTP client used by adapters.
func NewClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &http.Client{
		Timeout: timeout,
	}
}
