package httpx

import (
	"net/http"
	"time"
)

// NewClient returns the default HTTP client used by adapters.
func NewClient() *http.Client {
	return &http.Client{
		Timeout: 15 * time.Second,
	}
}
