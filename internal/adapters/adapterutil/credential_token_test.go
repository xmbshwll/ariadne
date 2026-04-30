package adapterutil

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	errCredentialTokenMissing = errors.New("credential token missing credentials")
	errCredentialTokenEmpty   = errors.New("credential token empty")
	errCredentialTokenFetch   = errors.New("credential token fetch failed")
)

func TestCredentialTokenSourceRequiresCredentials(t *testing.T) {
	var fetchCalled bool
	source := NewCredentialTokenSource(CredentialTokenSourceConfig{
		Credentials:        func() ClientCredentials { return ClientCredentials{} },
		MissingCredentials: errCredentialTokenMissing,
		EmptyAccessToken:   errCredentialTokenEmpty,
		Fetch: func(context.Context, ClientCredentials) (CredentialToken, error) {
			fetchCalled = true
			return CredentialToken{}, nil
		},
	})

	_, err := source.AccessToken(context.Background())

	require.ErrorIs(t, err, errCredentialTokenMissing)
	assert.False(t, fetchCalled)
}

func TestCredentialTokenSourceCachesUntilRefreshMargin(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	var fetches int
	source := NewCredentialTokenSource(CredentialTokenSourceConfig{
		Credentials: func() ClientCredentials {
			return ClientCredentials{ClientID: "client", ClientSecret: "secret"}
		},
		MissingCredentials: errCredentialTokenMissing,
		EmptyAccessToken:   errCredentialTokenEmpty,
		RefreshMargin:      10 * time.Second,
		Now:                func() time.Time { return now },
		Fetch: func(context.Context, ClientCredentials) (CredentialToken, error) {
			fetches++
			return CredentialToken{AccessToken: "token-" + string(rune('0'+fetches)), ExpiresIn: time.Minute}, nil
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
	source := NewCredentialTokenSource(CredentialTokenSourceConfig{
		Credentials: func() ClientCredentials {
			return ClientCredentials{ClientID: "client", ClientSecret: "secret"}
		},
		MissingCredentials: errCredentialTokenMissing,
		EmptyAccessToken:   errCredentialTokenEmpty,
		Fetch: func(context.Context, ClientCredentials) (CredentialToken, error) {
			fetches.Add(1)
			started <- struct{}{}
			<-allowResponse
			return CredentialToken{AccessToken: "token", ExpiresIn: time.Hour}, nil
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

func TestCredentialTokenSourceDoesNotCacheFetchError(t *testing.T) {
	var fetches int
	source := NewCredentialTokenSource(CredentialTokenSourceConfig{
		Credentials: func() ClientCredentials {
			return ClientCredentials{ClientID: "client", ClientSecret: "secret"}
		},
		MissingCredentials: errCredentialTokenMissing,
		EmptyAccessToken:   errCredentialTokenEmpty,
		Fetch: func(context.Context, ClientCredentials) (CredentialToken, error) {
			fetches++
			if fetches == 1 {
				return CredentialToken{}, errCredentialTokenFetch
			}
			return CredentialToken{AccessToken: "token", ExpiresIn: time.Hour}, nil
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
	source := NewCredentialTokenSource(CredentialTokenSourceConfig{
		Credentials: func() ClientCredentials {
			return ClientCredentials{ClientID: "client", ClientSecret: "secret"}
		},
		MissingCredentials: errCredentialTokenMissing,
		EmptyAccessToken:   errCredentialTokenEmpty,
		IsEmptyAccessToken: func(accessToken string) bool { return accessToken == " " },
		Fetch: func(context.Context, ClientCredentials) (CredentialToken, error) {
			return CredentialToken{AccessToken: " ", ExpiresIn: time.Hour}, nil
		},
	})

	_, err := source.AccessToken(context.Background())

	require.ErrorIs(t, err, errCredentialTokenEmpty)
}

func TestClientCredentialsBasicAuthorization(t *testing.T) {
	credentials := ClientCredentials{ClientID: "client", ClientSecret: "secret"}

	assert.Equal(t, "Basic Y2xpZW50OnNlY3JldA==", credentials.BasicAuthorization())
}
