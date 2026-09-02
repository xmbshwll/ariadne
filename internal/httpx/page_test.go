package httpx_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"context"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xmbshwll/ariadne/internal/httpx"
)

func TestPageFetcherBuildsPageRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, httpx.BrowserUserAgent, r.Header.Get("User-Agent"))
		_, _ = w.Write([]byte("page"))
	}))
	defer server.Close()

	body, err := httpx.PageFetcher{
		Client:       server.Client(),
		UserAgent:    httpx.BrowserUserAgent,
		BuildError:   "build page request",
		ExecuteError: "execute page request",
		ReadError:    "read page response",
	}.Fetch(context.Background(), server.URL)

	require.NoError(t, err)
	assert.Equal(t, []byte("page"), body)
}

func TestFetchPageAppliesTimeoutWhenCallerHasNoDeadline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(50 * time.Millisecond)
		_, _ = w.Write([]byte("late"))
	}))
	defer server.Close()

	_, err := httpx.FetchPage(context.Background(), httpx.PageRequest{
		BytesRequest: httpx.BytesRequest{
			RequestSpec: httpx.RequestSpec{
				Client:       server.Client(),
				URL:          server.URL,
				BuildError:   "build page request",
				ExecuteError: "execute page request",
			},
			ReadError: "read page response",
		},
		Timeout: time.Nanosecond,
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}
