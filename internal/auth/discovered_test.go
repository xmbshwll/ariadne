package auth_test

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xmbshwll/ariadne/internal/auth"
)

var (
	errDiscoveryFailed           = errors.New("discovery failed")
	errUnexpectedCredentialValue = errors.New("unexpected credential value")
)

func isRejected(err error) bool { return errors.Is(err, errDiscoveryFailed) }

// TestDiscoveredCredential pins the discovered-Credential seam: discover once
// and cache, skip discovery when a credential was observed incidentally, drop
// the cache on invalidation, and share one in-flight discovery.
func TestDiscoveredCredential(t *testing.T) {
	t.Run("discovers once and caches", func(t *testing.T) {
		var discoveries atomic.Int64
		credential := auth.NewDiscoveredCredential(func(context.Context) (string, error) {
			discoveries.Add(1)
			return "client-id-1", nil
		}, isRejected)

		first, err := credential.Credential(context.Background())
		require.NoError(t, err)
		second, err := credential.Credential(context.Background())
		require.NoError(t, err)

		assert.Equal(t, "client-id-1", first)
		assert.Equal(t, first, second)
		assert.Equal(t, int64(1), discoveries.Load())
	})

	t.Run("an observed credential skips discovery", func(t *testing.T) {
		var discoveries atomic.Int64
		credential := auth.NewDiscoveredCredential(func(context.Context) (string, error) {
			discoveries.Add(1)
			return "discovered", nil
		}, isRejected)

		credential.Observe("  observed-id  ")

		got, err := credential.Credential(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "observed-id", got)
		assert.Zero(t, discoveries.Load(), "observed values must not trigger discovery")
	})

	t.Run("invalidation forces a rediscovery", func(t *testing.T) {
		var discoveries atomic.Int64
		credential := auth.NewDiscoveredCredential(func(context.Context) (string, error) {
			discoveries.Add(1)
			return "client-id-" + time.Now().Format("150405.000000000"), nil
		}, isRejected)

		first, err := credential.Credential(context.Background())
		require.NoError(t, err)
		credential.Invalidate()
		second, err := credential.Credential(context.Background())
		require.NoError(t, err)

		assert.NotEqual(t, first, second)
		assert.Equal(t, int64(2), discoveries.Load())
	})

	t.Run("discovery failures pass through", func(t *testing.T) {
		credential := auth.NewDiscoveredCredential(func(context.Context) (string, error) {
			return "", errDiscoveryFailed
		}, isRejected)

		_, err := credential.Credential(context.Background())

		require.ErrorIs(t, err, errDiscoveryFailed)
	})

	t.Run("concurrent callers share one discovery", func(t *testing.T) {
		var discoveries atomic.Int64
		start := make(chan struct{})
		credential := auth.NewDiscoveredCredential(func(context.Context) (string, error) {
			discoveries.Add(1)
			<-start
			return "shared-id", nil
		}, isRejected)

		const callers = 8
		errCh := make(chan error, callers)
		for range callers {
			go func() {
				value, err := credential.Credential(context.Background())
				if err == nil && value != "shared-id" {
					err = fmt.Errorf("unexpected credential %q: %w", value, errUnexpectedCredentialValue)
				}
				errCh <- err
			}()
		}
		start <- struct{}{}
		for range callers {
			require.NoError(t, <-errCh)
		}
		assert.Equal(t, int64(1), discoveries.Load(), "one in-flight discovery serves every caller")
	})
}
