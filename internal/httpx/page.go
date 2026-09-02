package httpx

import (
	"context"
	"net/http"
	"time"
)

// PageRequest fetches one HTML page rather than an API payload.
type PageRequest struct {
	BytesRequest
	Timeout time.Duration
}

// PageFetcher reads HTML pages with a browser user agent, which is what the
// services without a public search API need.
type PageFetcher struct {
	Client         *http.Client
	UserAgent      string
	BuildError     string
	ExecuteError   string
	StatusError    StatusErrorFunc
	ErrorBodyLimit int64
	ReadError      string
	MaxBodyBytes   int64
	TooLargeError  error
	TooLarge       TooLargeErrorFunc
	Timeout        time.Duration
}

// Fetch reads one page with the fetcher's settings.
func (f PageFetcher) Fetch(ctx context.Context, requestURL string) ([]byte, error) {
	return FetchPage(ctx, PageRequest{
		BytesRequest: BytesRequest{
			RequestSpec: RequestSpec{
				Client:         f.Client,
				URL:            requestURL,
				UserAgent:      f.UserAgent,
				BuildError:     f.BuildError,
				ExecuteError:   f.ExecuteError,
				StatusError:    f.StatusError,
				ErrorBodyLimit: f.ErrorBodyLimit,
			},
			ReadError:     f.ReadError,
			MaxBodyBytes:  f.MaxBodyBytes,
			TooLargeError: f.TooLargeError,
			TooLarge:      f.TooLarge,
		},
		Timeout: f.Timeout,
	})
}

// FetchPage applies its own timeout, shortened to whatever the caller's context
// already allows.
func FetchPage(ctx context.Context, spec PageRequest) ([]byte, error) {
	if spec.Timeout <= 0 {
		return FetchBytes(ctx, spec.BytesRequest)
	}

	timeout := spec.Timeout
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining < timeout {
			timeout = remaining
		}
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return FetchBytes(ctx, spec.BytesRequest)
}
