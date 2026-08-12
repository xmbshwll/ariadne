package targetsearch

import (
	"context"
	"errors"
	"net"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

var errTargetSearchUnavailable = errors.New("target search unavailable")

type timeoutError struct{}

func (timeoutError) Error() string { return "timeout" }
func (timeoutError) Timeout() bool { return true }

func TestUnavailablePreservesErrorChain(t *testing.T) {
	err := Unavailable(errTargetSearchUnavailable)

	assert.True(t, IsUnavailable(err))
	assert.ErrorIs(t, err, errTargetSearchUnavailable)
}

func TestIsRecoverableTimeout(t *testing.T) {
	deadlineCtx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()

	netTimeoutErr := &url.Error{
		Op:  "Get",
		URL: "https://example.test",
		Err: &net.OpError{Op: "dial", Net: "tcp", Err: timeoutError{}},
	}

	tests := []struct {
		name string
		ctx  context.Context
		err  error
		want bool
	}{
		{name: "deadline exceeded with active context", ctx: context.Background(), err: context.DeadlineExceeded, want: true},
		{name: "deadline exceeded with expired context", ctx: deadlineCtx, err: context.DeadlineExceeded, want: false},
		{name: "net timeout with active context", ctx: context.Background(), err: netTimeoutErr, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsRecoverableTimeout(tt.ctx, tt.err))
		})
	}
}
