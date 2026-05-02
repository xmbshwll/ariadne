package adapterutil

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	errHTTPExchangeStatus    = errors.New("http exchange status")
	errHTTPExchangeMalformed = errors.New("http exchange malformed")
	errHTTPExchangeTooLarge  = errors.New("http exchange too large")
)

func TestGetJSONSendsHeadersAndDecodesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, DefaultUserAgent, r.Header.Get("User-Agent"))
		assert.Equal(t, "Bearer token", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"name":"ariadne"}`))
	}))
	defer server.Close()

	var payload struct {
		Name string `json:"name"`
	}
	err := GetJSON(context.Background(), JSONRequest{
		RequestSpec: RequestSpec{
			Client:       server.Client(),
			URL:          server.URL,
			Headers:      map[string]string{"Authorization": "Bearer token"},
			UserAgent:    DefaultUserAgent,
			BuildError:   "build test request",
			ExecuteError: "execute test request",
			StatusError:  StatusError(errHTTPExchangeStatus),
		},
		DecodeError:       "decode test response",
		MalformedResponse: errHTTPExchangeMalformed,
	}, &payload)

	require.NoError(t, err)
	assert.Equal(t, "ariadne", payload.Name)
}

func TestGetJSONWrapsStatusAndDecodeErrors(t *testing.T) {
	t.Run("status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "nope", http.StatusTeapot)
		}))
		defer server.Close()

		var payload struct{}
		err := GetJSON(context.Background(), JSONRequest{
			RequestSpec: RequestSpec{
				Client:       server.Client(),
				URL:          server.URL,
				BuildError:   "build test request",
				ExecuteError: "execute test request",
				StatusError:  StatusError(errHTTPExchangeStatus),
			},
			DecodeError: "decode test response",
		}, &payload)

		require.Error(t, err)
		assert.ErrorIs(t, err, errHTTPExchangeStatus)
		assert.Contains(t, err.Error(), "418")
		assert.Contains(t, err.Error(), "nope")
	})

	t.Run("decode", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`not json`))
		}))
		defer server.Close()

		var payload struct{}
		err := GetJSON(context.Background(), JSONRequest{
			RequestSpec: RequestSpec{
				Client:       server.Client(),
				URL:          server.URL,
				BuildError:   "build test request",
				ExecuteError: "execute test request",
				StatusError:  StatusError(errHTTPExchangeStatus),
			},
			DecodeError:       "decode test response",
			MalformedResponse: errHTTPExchangeMalformed,
		}, &payload)

		require.Error(t, err)
		assert.ErrorIs(t, err, errHTTPExchangeMalformed)
	})
}

func TestFetchBytesReadsAndLimitsResponseBody(t *testing.T) {
	t.Run("accepts non-200 success status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte("created"))
		}))
		defer server.Close()

		body, err := FetchBytes(context.Background(), BytesRequest{
			RequestSpec: RequestSpec{
				Client:       server.Client(),
				URL:          server.URL,
				BuildError:   "build test request",
				ExecuteError: "execute test request",
				StatusError:  StatusError(errHTTPExchangeStatus),
			},
			ReadError: "read test response",
		})

		require.NoError(t, err)
		assert.Equal(t, []byte("created"), body)
	})

	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("hello"))
		}))
		defer server.Close()

		body, err := FetchBytes(context.Background(), BytesRequest{
			RequestSpec: RequestSpec{
				Client:       server.Client(),
				URL:          server.URL,
				BuildError:   "build test request",
				ExecuteError: "execute test request",
				StatusError:  StatusError(errHTTPExchangeStatus),
			},
			ReadError:     "read test response",
			MaxBodyBytes:  10,
			TooLargeError: errHTTPExchangeTooLarge,
		})

		require.NoError(t, err)
		assert.Equal(t, []byte("hello"), body)
	})

	t.Run("too large", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("hello"))
		}))
		defer server.Close()

		_, err := FetchBytes(context.Background(), BytesRequest{
			RequestSpec: RequestSpec{
				Client:       server.Client(),
				URL:          server.URL,
				BuildError:   "build test request",
				ExecuteError: "execute test request",
				StatusError:  StatusError(errHTTPExchangeStatus),
			},
			ReadError:     "read test response",
			MaxBodyBytes:  4,
			TooLargeError: errHTTPExchangeTooLarge,
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, errHTTPExchangeTooLarge)
	})
}
