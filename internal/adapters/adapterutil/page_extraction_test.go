package adapterutil

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	errPageExtractionNotFound  = errors.New("page extraction not found")
	errPageExtractionMalformed = errors.New("page extraction malformed")
)

func TestPageFetcherBuildsPageRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, BrowserUserAgent, r.Header.Get("User-Agent"))
		_, _ = w.Write([]byte("page"))
	}))
	defer server.Close()

	body, err := PageFetcher{
		Client:       server.Client(),
		UserAgent:    BrowserUserAgent,
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

	_, err := FetchPage(context.Background(), PageRequest{
		BytesRequest: BytesRequest{
			RequestSpec: RequestSpec{
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

func TestFirstRegexpGroup(t *testing.T) {
	pattern := regexp.MustCompile(`<script>(.*?)</script>`)

	payload, err := FirstRegexpGroup([]byte(`<script>payload</script>`), pattern, errPageExtractionNotFound)
	require.NoError(t, err)
	assert.Equal(t, []byte("payload"), payload)

	_, err = FirstRegexpGroup([]byte(`<html></html>`), pattern, errPageExtractionNotFound)
	require.ErrorIs(t, err, errPageExtractionNotFound)
}

func TestDecodeJSONBlock(t *testing.T) {
	pattern := regexp.MustCompile(`<script>(.*?)</script>`)

	payload, err := DecodeJSONBlock[struct {
		Name string `json:"name"`
	}](
		[]byte(`<script>{"name":"ariadne"}</script>`),
		pattern,
		errPageExtractionNotFound,
		"decode page json",
		errPageExtractionMalformed,
	)
	require.NoError(t, err)
	assert.Equal(t, "ariadne", payload.Name)

	_, err = DecodeJSONBlock[struct{}](
		[]byte(`<script>{</script>`),
		pattern,
		errPageExtractionNotFound,
		"decode page json",
		errPageExtractionMalformed,
	)
	require.ErrorIs(t, err, errPageExtractionMalformed)
}
