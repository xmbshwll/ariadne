package auth_test

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xmbshwll/ariadne/internal/auth"
	"github.com/xmbshwll/ariadne/internal/httpx"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	errTokenMissing = errors.New("token source missing credentials")
	errTokenEmpty   = errors.New("token empty")
	errTokenFetch   = errors.New("token fetch failed")
)

type accessTokenResult struct {
	accessToken string
	err         error
}

func TestTokenSourceRequiresCredentials(t *testing.T) {
	var fetchCalled bool
	source := auth.NewTokenSource(auth.TokenSourceConfig{
		Credentials:        auth.ClientCredentials{},
		MissingCredentials: errTokenMissing,
		EmptyAccessToken:   errTokenEmpty,
		Fetch: func(context.Context, auth.ClientCredentials) (auth.Token, error) {
			fetchCalled = true
			return auth.Token{}, nil
		},
	})

	_, err := source.AccessToken(context.Background())

	require.ErrorIs(t, err, errTokenMissing)
	assert.False(t, fetchCalled)
}

func TestTokenSourceCachesUntilRefreshMargin(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	var fetches int
	source := auth.NewTokenSource(auth.TokenSourceConfig{
		Credentials:        auth.ClientCredentials{ClientID: "client", ClientSecret: "secret"},
		MissingCredentials: errTokenMissing,
		EmptyAccessToken:   errTokenEmpty,
		RefreshMargin:      10 * time.Second,
		Now:                func() time.Time { return now },
		Fetch: func(context.Context, auth.ClientCredentials) (auth.Token, error) {
			fetches++
			return auth.Token{AccessToken: "token-" + string(rune('0'+fetches)), ExpiresIn: time.Minute}, nil
		},
	})

	first, err := source.AccessToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "token-1", first)

	now = now.Add(49 * time.Second)
	second, err := source.AccessToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "token-1", second)

	now = now.Add(time.Second)
	third, err := source.AccessToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "token-2", third)
	assert.Equal(t, 2, fetches)
}

func TestTokenSourceSerializesConcurrentRefresh(t *testing.T) {
	started := make(chan struct{}, 8)
	allowResponse := make(chan struct{})
	var fetches atomic.Int32
	source := auth.NewTokenSource(auth.TokenSourceConfig{
		Credentials:        auth.ClientCredentials{ClientID: "client", ClientSecret: "secret"},
		MissingCredentials: errTokenMissing,
		EmptyAccessToken:   errTokenEmpty,
		Fetch: func(context.Context, auth.ClientCredentials) (auth.Token, error) {
			fetches.Add(1)
			started <- struct{}{}
			<-allowResponse
			return auth.Token{AccessToken: "token", ExpiresIn: time.Hour}, nil
		},
	})

	var wg sync.WaitGroup
	errCh := make(chan error, 8)
	for range 8 {
		wg.Go(func() {
			accessToken, err := source.AccessToken(context.Background())
			if err == nil {
				assert.Equal(t, "token", accessToken)
			}
			errCh <- err
		})
	}

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		require.FailNow(t, "timed out waiting for token refresh")
	}

	select {
	case <-started:
		require.FailNow(t, "saw concurrent token refresh")
	case <-time.After(100 * time.Millisecond):
	}
	close(allowResponse)
	wg.Wait()
	close(errCh)

	for err := range errCh {
		require.NoError(t, err)
	}
	assert.EqualValues(t, 1, fetches.Load())
}

func TestTokenSourceCallerCancellationDoesNotCancelSharedRefresh(t *testing.T) {
	started := make(chan struct{}, 1)
	allowResponse := make(chan struct{})
	var fetches atomic.Int32
	source := auth.NewTokenSource(auth.TokenSourceConfig{
		Credentials:        auth.ClientCredentials{ClientID: "client", ClientSecret: "secret"},
		MissingCredentials: errTokenMissing,
		EmptyAccessToken:   errTokenEmpty,
		Fetch: func(ctx context.Context, _ auth.ClientCredentials) (auth.Token, error) {
			fetches.Add(1)
			started <- struct{}{}
			select {
			case <-allowResponse:
				return auth.Token{AccessToken: "token", ExpiresIn: time.Hour}, nil
			case <-ctx.Done():
				return auth.Token{}, ctx.Err()
			}
		},
	})

	cancelledCtx, cancel := context.WithCancel(context.Background())
	firstErrCh := make(chan error, 1)
	go func() {
		_, err := source.AccessToken(cancelledCtx)
		firstErrCh <- err
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		require.FailNow(t, "timed out waiting for token refresh")
	}
	cancel()
	require.ErrorIs(t, <-firstErrCh, context.Canceled)

	secondResultCh := make(chan accessTokenResult, 1)
	go func() {
		accessToken, err := source.AccessToken(context.Background())
		secondResultCh <- accessTokenResult{accessToken: accessToken, err: err}
	}()

	close(allowResponse)
	secondResult := <-secondResultCh
	require.NoError(t, secondResult.err)
	assert.Equal(t, "token", secondResult.accessToken)
	assert.EqualValues(t, 1, fetches.Load())
}

func TestTokenSourceDoesNotCacheFetchError(t *testing.T) {
	var fetches int
	source := auth.NewTokenSource(auth.TokenSourceConfig{
		Credentials:        auth.ClientCredentials{ClientID: "client", ClientSecret: "secret"},
		MissingCredentials: errTokenMissing,
		EmptyAccessToken:   errTokenEmpty,
		Fetch: func(context.Context, auth.ClientCredentials) (auth.Token, error) {
			fetches++
			if fetches == 1 {
				return auth.Token{}, errTokenFetch
			}
			return auth.Token{AccessToken: "token", ExpiresIn: time.Hour}, nil
		},
	})

	_, err := source.AccessToken(context.Background())
	require.ErrorIs(t, err, errTokenFetch)

	accessToken, err := source.AccessToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "token", accessToken)
	assert.Equal(t, 2, fetches)
}

func TestTokenSourceRejectsEmptyToken(t *testing.T) {
	source := auth.NewTokenSource(auth.TokenSourceConfig{
		Credentials:        auth.ClientCredentials{ClientID: "client", ClientSecret: "secret"},
		MissingCredentials: errTokenMissing,
		EmptyAccessToken:   errTokenEmpty,
		// A blank access token is empty, whatever the provider returned.
		Fetch: func(context.Context, auth.ClientCredentials) (auth.Token, error) {
			return auth.Token{AccessToken: " ", ExpiresIn: time.Hour}, nil
		},
	})

	_, err := source.AccessToken(context.Background())

	require.ErrorIs(t, err, errTokenEmpty)
}

var errTestTokenSentinel = errors.New("test token error")

func TestTokenSourceRetriesTransientHTTPErrors(t *testing.T) {
	var fetches int
	source := auth.NewTokenSource(auth.TokenSourceConfig{
		Credentials:         auth.ClientCredentials{ClientID: "client", ClientSecret: "secret"},
		MissingCredentials:  errTokenMissing,
		EmptyAccessToken:    errTokenEmpty,
		MaxRefreshAttempts:  3,
		RefreshRetryBackoff: time.Millisecond,
		Fetch: func(context.Context, auth.ClientCredentials) (auth.Token, error) {
			fetches++
			if fetches < 3 {
				return auth.Token{}, httpx.StatusError(errTestTokenSentinel)(http.StatusServiceUnavailable, "temporarily_unavailable")
			}
			return auth.Token{AccessToken: "token", ExpiresIn: time.Hour}, nil
		},
	})

	token, err := source.AccessToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "token", token)
	assert.Equal(t, 3, fetches)
}

func TestTokenSourceDoesNotRetryNonTransientErrors(t *testing.T) {
	var fetches int
	source := auth.NewTokenSource(auth.TokenSourceConfig{
		Credentials:         auth.ClientCredentials{ClientID: "client", ClientSecret: "secret"},
		MissingCredentials:  errTokenMissing,
		EmptyAccessToken:    errTokenEmpty,
		MaxRefreshAttempts:  3,
		RefreshRetryBackoff: time.Millisecond,
		Fetch: func(context.Context, auth.ClientCredentials) (auth.Token, error) {
			fetches++
			return auth.Token{}, httpx.StatusError(errTestTokenSentinel)(http.StatusUnauthorized, "unauthorized")
		},
	})

	_, err := source.AccessToken(context.Background())
	require.Error(t, err)
	assert.Equal(t, 1, fetches)
}

func TestClientCredentialsBasicAuthorization(t *testing.T) {
	credentials := auth.ClientCredentials{ClientID: "client", ClientSecret: "secret"}

	assert.Equal(t, "Basic Y2xpZW50OnNlY3JldA==", credentials.BasicAuthorization())
}
