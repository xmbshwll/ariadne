package targetsearch

import (
	"context"
	"errors"
	"net"
)

type unavailableError struct {
	err error
}

func (e unavailableError) Error() string {
	return e.err.Error()
}

func (e unavailableError) Unwrap() error {
	return e.err
}

// Unavailable classifies err as a recoverable Target Search unavailability.
func Unavailable(err error) error {
	if err == nil || IsUnavailable(err) {
		return err
	}
	return unavailableError{err: err}
}

// IsUnavailable reports whether err describes an unavailable Target Search path.
func IsUnavailable(err error) bool {
	var unavailable unavailableError
	return errors.As(err, &unavailable)
}

// IsRecoverableTimeout reports whether err can be skipped while ctx remains active.
func IsRecoverableTimeout(ctx context.Context, err error) bool {
	if err == nil || ctx.Err() != nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
