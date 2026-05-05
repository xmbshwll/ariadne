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

func TestIsRecoverableTimeoutRequiresActiveContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()

	assert.True(t, IsRecoverableTimeout(context.Background(), context.DeadlineExceeded))
	assert.False(t, IsRecoverableTimeout(ctx, context.DeadlineExceeded))
}

func TestIsRecoverableTimeoutAcceptsNetTimeouts(t *testing.T) {
	err := &url.Error{
		Op:  "Get",
		URL: "https://example.test",
		Err: &net.OpError{Op: "dial", Net: "tcp", Err: timeoutError{}},
	}

	assert.True(t, IsRecoverableTimeout(context.Background(), err))
}
