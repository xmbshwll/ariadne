package httpx_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xmbshwll/ariadne/internal/httpx"
)

var errRetryBoom = errors.New("retry boom")

func statusErr(status int) error {
	return &httpStatusProbe{status: status}
}

type httpStatusProbe struct{ status int }

func (e *httpStatusProbe) Error() string       { return "probe status error" }
func (e *httpStatusProbe) HTTPStatusCode() int { return e.status }

func TestRetryStopsAtFirstSuccess(t *testing.T) {
	calls := 0
	err := httpx.Retry(context.Background(), 3, time.Millisecond, func(context.Context) error {
		calls++
		if calls == 1 {
			return statusErr(http.StatusBadGateway)
		}
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 2, calls)
}

func TestRetryReturnsNonTransientErrorImmediately(t *testing.T) {
	calls := 0
	err := httpx.Retry(context.Background(), 3, time.Millisecond, func(context.Context) error {
		calls++
		return errRetryBoom
	})
	require.ErrorIs(t, err, errRetryBoom)
	assert.Equal(t, 1, calls, "a non-transient error must not be retried")
}

func TestRetryGivesUpAfterAttempts(t *testing.T) {
	calls := 0
	err := httpx.Retry(context.Background(), 3, time.Millisecond, func(context.Context) error {
		calls++
		return statusErr(http.StatusServiceUnavailable)
	})
	require.Error(t, err)
	assert.Equal(t, 3, calls)
}

func TestRetryAbortsOnContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	err := httpx.Retry(ctx, 5, 10*time.Millisecond, func(ctx context.Context) error {
		cancel()
		return statusErr(http.StatusBadGateway)
	})
	require.ErrorIs(t, err, context.Canceled)
}
