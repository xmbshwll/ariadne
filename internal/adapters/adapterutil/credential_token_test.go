package adapterutil_test

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	adapterutil "github.com/xmbshwll/ariadne/internal/adapters/adapterutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	errCredentialTokenMissing = errors.New("credential token missing credentials")
	errCredentialTokenEmpty   = errors.New("credential token empty")
	errCredentialTokenFetch   = errors.New("credential token fetch failed")
)

type accessTokenResult struct {
	accessToken string
	err         error
}

func TestCredentialTokenSourceRequiresCredentials(t *testing.T) {
	var fetchCalled bool
	source := adapterutil.NewCredentialTokenSource(adapterutil.CredentialTokenSourceConfig{
		Credentials:        func() adapterutil.ClientCredentials { return adapterutil.ClientCredentials{} },
		MissingCredentials: errCredentialTokenMissing,
		EmptyAccessToken:   errCredentialTokenEmpty,
		Fetch: func(context.Context, adapterutil.ClientCredentials) (adapterutil.CredentialToken, error) {
			fetchCalled = true
			return adapterutil.CredentialToken{}, nil
		},
	})

	_, err := source.AccessToken(context.Background())

	require.ErrorIs(t, err, errCredentialTokenMissing)
	assert.False(t, fetchCalled)
}

func TestCredentialTokenSourceCachesUntilRefreshMargin(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	var fetches int
	source := adapterutil.NewCredentialTokenSource(adapterutil.CredentialTokenSourceConfig{
		Credentials: func() adapterutil.ClientCredentials {
			return adapterutil.ClientCredentials{ClientID: "client", ClientSecret: "secret"}
		},
		MissingCredentials: errCredentialTokenMissing,
		EmptyAccessToken:   errCredentialTokenEmpty,
		RefreshMargin:      10 * time.Second,
		Now:                func() time.Time { return now },
		Fetch: func(context.Context, adapterutil.ClientCredentials) (adapterutil.CredentialToken, error) {
			fetches++
			return adapterutil.CredentialToken{AccessToken: "token-" + string(rune('0'+fetches)), ExpiresIn: time.Minute}, nil
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

func TestCredentialTokenSourceSerializesConcurrentRefresh(t *testing.T) {
	started := make(chan struct{}, 8)
	allowResponse := make(chan struct{})
	var fetches atomic.Int32
	source := adapterutil.NewCredentialTokenSource(adapterutil.CredentialTokenSourceConfig{
		Credentials: func() adapterutil.ClientCredentials {
			return adapterutil.ClientCredentials{ClientID: "client", ClientSecret: "secret"}
		},
		MissingCredentials: errCredentialTokenMissing,
		EmptyAccessToken:   errCredentialTokenEmpty,
		Fetch: func(context.Context, adapterutil.ClientCredentials) (adapterutil.CredentialToken, error) {
			fetches.Add(1)
			started <- struct{}{}
			<-allowResponse
			return adapterutil.CredentialToken{AccessToken: "token", ExpiresIn: time.Hour}, nil
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

func TestCredentialTokenSourceCallerCancellationDoesNotCancelSharedRefresh(t *testing.T) {
	started := make(chan struct{}, 1)
	allowResponse := make(chan struct{})
	var fetches atomic.Int32
	source := adapterutil.NewCredentialTokenSource(adapterutil.CredentialTokenSourceConfig{
		Credentials: func() adapterutil.ClientCredentials {
			return adapterutil.ClientCredentials{ClientID: "client", ClientSecret: "secret"}
		},
		MissingCredentials: errCredentialTokenMissing,
		EmptyAccessToken:   errCredentialTokenEmpty,
		Fetch: func(ctx context.Context, _ adapterutil.ClientCredentials) (adapterutil.CredentialToken, error) {
			fetches.Add(1)
			started <- struct{}{}
			select {
			case <-allowResponse:
				return adapterutil.CredentialToken{AccessToken: "token", ExpiresIn: time.Hour}, nil
			case <-ctx.Done():
				return adapterutil.CredentialToken{}, ctx.Err()
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

func TestCredentialTokenSourceDoesNotCacheFetchError(t *testing.T) {
	var fetches int
	source := adapterutil.NewCredentialTokenSource(adapterutil.CredentialTokenSourceConfig{
		Credentials: func() adapterutil.ClientCredentials {
			return adapterutil.ClientCredentials{ClientID: "client", ClientSecret: "secret"}
		},
		MissingCredentials: errCredentialTokenMissing,
		EmptyAccessToken:   errCredentialTokenEmpty,
		Fetch: func(context.Context, adapterutil.ClientCredentials) (adapterutil.CredentialToken, error) {
			fetches++
			if fetches == 1 {
				return adapterutil.CredentialToken{}, errCredentialTokenFetch
			}
			return adapterutil.CredentialToken{AccessToken: "token", ExpiresIn: time.Hour}, nil
		},
	})

	_, err := source.AccessToken(context.Background())
	require.ErrorIs(t, err, errCredentialTokenFetch)

	accessToken, err := source.AccessToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "token", accessToken)
	assert.Equal(t, 2, fetches)
}

func TestCredentialTokenSourceRejectsEmptyToken(t *testing.T) {
	source := adapterutil.NewCredentialTokenSource(adapterutil.CredentialTokenSourceConfig{
		Credentials: func() adapterutil.ClientCredentials {
			return adapterutil.ClientCredentials{ClientID: "client", ClientSecret: "secret"}
		},
		MissingCredentials: errCredentialTokenMissing,
		EmptyAccessToken:   errCredentialTokenEmpty,
		IsEmptyAccessToken: func(accessToken string) bool { return accessToken == " " },
		Fetch: func(context.Context, adapterutil.ClientCredentials) (adapterutil.CredentialToken, error) {
			return adapterutil.CredentialToken{AccessToken: " ", ExpiresIn: time.Hour}, nil
		},
	})

	_, err := source.AccessToken(context.Background())

	require.ErrorIs(t, err, errCredentialTokenEmpty)
}

var errTestTokenSentinel = errors.New("test token error")

func TestCredentialTokenSourceRetriesTransientHTTPErrors(t *testing.T) {
	var fetches int
	source := adapterutil.NewCredentialTokenSource(adapterutil.CredentialTokenSourceConfig{
		Credentials: func() adapterutil.ClientCredentials {
			return adapterutil.ClientCredentials{ClientID: "client", ClientSecret: "secret"}
		},
		MissingCredentials:  errCredentialTokenMissing,
		EmptyAccessToken:    errCredentialTokenEmpty,
		MaxRefreshAttempts:  3,
		RefreshRetryBackoff: time.Millisecond,
		Fetch: func(context.Context, adapterutil.ClientCredentials) (adapterutil.CredentialToken, error) {
			fetches++
			if fetches < 3 {
				return adapterutil.CredentialToken{}, adapterutil.StatusError(errTestTokenSentinel)(http.StatusServiceUnavailable, "temporarily_unavailable")
			}
			return adapterutil.CredentialToken{AccessToken: "token", ExpiresIn: time.Hour}, nil
		},
	})

	token, err := source.AccessToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "token", token)
	assert.Equal(t, 3, fetches)
}

func TestCredentialTokenSourceDoesNotRetryNonTransientErrors(t *testing.T) {
	var fetches int
	source := adapterutil.NewCredentialTokenSource(adapterutil.CredentialTokenSourceConfig{
		Credentials: func() adapterutil.ClientCredentials {
			return adapterutil.ClientCredentials{ClientID: "client", ClientSecret: "secret"}
		},
		MissingCredentials:  errCredentialTokenMissing,
		EmptyAccessToken:    errCredentialTokenEmpty,
		MaxRefreshAttempts:  3,
		RefreshRetryBackoff: time.Millisecond,
		Fetch: func(context.Context, adapterutil.ClientCredentials) (adapterutil.CredentialToken, error) {
			fetches++
			return adapterutil.CredentialToken{}, adapterutil.StatusError(errTestTokenSentinel)(http.StatusUnauthorized, "unauthorized")
		},
	})

	_, err := source.AccessToken(context.Background())
	require.Error(t, err)
	assert.Equal(t, 1, fetches)
}

func TestClientCredentialsBasicAuthorization(t *testing.T) {
	credentials := adapterutil.ClientCredentials{ClientID: "client", ClientSecret: "secret"}

	assert.Equal(t, "Basic Y2xpZW50OnNlY3JldA==", credentials.BasicAuthorization())
}
