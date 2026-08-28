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

type httpStatusProbe struct{ status int }

func (e *httpStatusProbe) Error() string       { return "probe status error" }
func (e *httpStatusProbe) HTTPStatusCode() int { return e.status }

func transient(status int) error { return &httpStatusProbe{status: status} }

// TestRetry pins the shared transient-retry policy: retry only transient
// failures, stop at first success or non-transient error, exhaust attempts
// otherwise, and abort the wait when the caller cancels.
func TestRetry(t *testing.T) {
	tests := []struct {
		name        string
		attempts    int
		backoff     time.Duration
		failures    []error // one per call before the success; empty means never succeed
		cancelOn    int     // 1-based call after which ctx is canceled; 0 = never
		wantCalls   int
		wantErr     error  // compared with errors.Is (identity)
		wantErrText string // compared with ErrorContains (fresh instances)
		wantAbort   bool   // expect context.Canceled
	}{
		{
			name:      "stops at the first success",
			attempts:  3,
			backoff:   time.Millisecond,
			failures:  []error{transient(http.StatusBadGateway)},
			wantCalls: 2,
		},
		{
			name:      "returns a non-transient error immediately",
			attempts:  3,
			backoff:   time.Millisecond,
			failures:  []error{errRetryBoom},
			wantCalls: 1,
			wantErr:   errRetryBoom,
		},
		{
			name:        "exhausts all attempts on transient failures",
			attempts:    3,
			backoff:     time.Millisecond,
			failures:    []error{transient(http.StatusServiceUnavailable), transient(http.StatusServiceUnavailable), transient(http.StatusServiceUnavailable), transient(http.StatusServiceUnavailable)},
			wantCalls:   3,
			wantErrText: "probe status error",
		},
		{
			name:      "aborts the backoff when the caller cancels",
			attempts:  5,
			backoff:   10 * time.Millisecond,
			failures:  []error{transient(http.StatusBadGateway)},
			cancelOn:  1,
			wantCalls: 1,
			wantAbort: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			calls := 0
			err := httpx.Retry(ctx, tt.attempts, tt.backoff, func(context.Context) error {
				calls++
				if tt.cancelOn != 0 && calls == tt.cancelOn {
					cancel()
				}
				if calls <= len(tt.failures) {
					return tt.failures[calls-1]
				}
				return nil
			})

			assert.Equal(t, tt.wantCalls, calls, tt.name)
			if tt.wantAbort {
				require.ErrorIs(t, err, context.Canceled, tt.name)
				return
			}
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr, tt.name)
				return
			}
			if tt.wantErrText != "" {
				require.ErrorContains(t, err, tt.wantErrText, tt.name)
				return
			}
			require.NoError(t, err, tt.name)
		})
	}
}
