// Package auth holds what every Music Service needs to authenticate: client
// credentials, and a token source that fetches a bearer token once, caches it
// until just before it expires, and shares one in-flight refresh between
// concurrent callers. Provider-specific flows (Apple Music's signed developer
// token) live in a subpackage.
package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/xmbshwll/ariadne/internal/httpx"
	"golang.org/x/sync/singleflight"
)

const (
	defaultTokenRefreshMargin   = 30 * time.Second
	defaultMaxRefreshAttempts   = 3
	defaultRefreshRetryBackoff  = 250 * time.Millisecond
	defaultTokenSingleflightKey = "auth-token"
)

var (
	errTokenSourceNotConfigured = errors.New("token source not configured")
	errTokenResultInvalid       = errors.New("token result invalid")
)

// ClientCredentials is an API client id and secret pair.
type ClientCredentials struct {
	ClientID     string
	ClientSecret string
}

// Configured reports whether both halves are present, which is what decides
// whether a Music Service is enabled at all.
func (c ClientCredentials) Configured() bool {
	return strings.TrimSpace(c.ClientID) != "" && strings.TrimSpace(c.ClientSecret) != ""
}

// BasicAuthorization renders the credentials as an HTTP Basic header value.
func (c ClientCredentials) BasicAuthorization() string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(c.ClientID+":"+c.ClientSecret))
}

// Token is one issued bearer token.
type Token struct {
	AccessToken string
	ExpiresIn   time.Duration
}

// FetchFunc issues the provider's token request.
type FetchFunc func(context.Context, ClientCredentials) (Token, error)

// TokenSourceConfig is what varies per Music Service about its token flow. The
// errors are provider sentinels so callers keep branching on their own
// ErrCredentialsNotConfigured. RefreshMargin, Now, MaxRefreshAttempts and
// RefreshRetryBackoff exist for tests; production leaves them alone.
type TokenSourceConfig struct {
	Service             string
	Credentials         ClientCredentials
	MissingCredentials  error
	EmptyAccessToken    error
	Fetch               FetchFunc
	RefreshTimeout      time.Duration
	RefreshMargin       time.Duration
	MaxRefreshAttempts  int
	RefreshRetryBackoff time.Duration
	Now                 func() time.Time
}

// TokenSource fetches access tokens for one Music Service, caching them until
// shortly before expiry and serializing concurrent refreshes.
type TokenSource struct {
	config TokenSourceConfig
	mu     sync.Mutex
	group  singleflight.Group
	cached cachedToken
}

type cachedToken struct {
	accessToken string
	expiresAt   time.Time
}

// NewTokenSource fills in the shared defaults, so a provider states only what
// is specific to it.
func NewTokenSource(config TokenSourceConfig) *TokenSource {
	if config.RefreshMargin <= 0 {
		config.RefreshMargin = defaultTokenRefreshMargin
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.MaxRefreshAttempts <= 0 {
		config.MaxRefreshAttempts = defaultMaxRefreshAttempts
	}
	if config.RefreshRetryBackoff <= 0 {
		config.RefreshRetryBackoff = defaultRefreshRetryBackoff
	}
	return &TokenSource{config: config}
}

// CredentialsConfigured reports whether the service behind this source has
// usable credentials.
func (s *TokenSource) CredentialsConfigured() bool {
	if s == nil {
		return false
	}
	return s.config.Credentials.Configured()
}

// AccessToken returns a valid token, fetching one when the cache is empty or
// close to expiry.
func (s *TokenSource) AccessToken(ctx context.Context) (string, error) {
	if s == nil || s.config.Fetch == nil {
		return "", errTokenSourceNotConfigured
	}
	if ctx == nil {
		ctx = context.Background()
	}

	credentials := s.config.Credentials
	if !credentials.Configured() {
		return "", s.config.MissingCredentials
	}
	if accessToken, ok := s.cachedAccessToken(); ok {
		return accessToken, nil
	}
	if err := ctx.Err(); err != nil {
		//nolint:wrapcheck // Return caller cancellation unchanged.
		return "", err
	}

	// The refresh is shared, so it must outlive whichever caller happened to
	// start it.
	detachedCtx := context.WithoutCancel(ctx)
	resultCh := s.group.DoChan(s.singleflightKey(), func() (any, error) {
		if accessToken, ok := s.cachedAccessToken(); ok {
			return accessToken, nil
		}
		return s.refreshAccessToken(detachedCtx, credentials)
	})

	select {
	case <-ctx.Done():
		//nolint:wrapcheck // Return caller cancellation unchanged.
		return "", ctx.Err()
	case result := <-resultCh:
		if result.Err != nil {
			return "", result.Err
		}
		accessToken, ok := result.Val.(string)
		if !ok {
			return "", errTokenResultInvalid
		}
		return accessToken, nil
	}
}

func (s *TokenSource) singleflightKey() string {
	if s.config.Service == "" {
		return defaultTokenSingleflightKey
	}
	return s.config.Service + "-token"
}

func (s *TokenSource) cachedAccessToken() (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cached.accessToken == "" || !s.config.Now().Before(s.cached.expiresAt) {
		return "", false
	}
	return s.cached.accessToken, true
}

func (s *TokenSource) refreshAccessToken(ctx context.Context, credentials ClientCredentials) (string, error) {
	var accessToken string
	err := httpx.Retry(ctx, s.config.MaxRefreshAttempts, s.config.RefreshRetryBackoff, func(ctx context.Context) error {
		token, fetchErr := s.fetchAndCacheToken(ctx, credentials)
		accessToken = token
		return fetchErr
	})
	if err != nil {
		return "", fmt.Errorf("refresh credential token: %w", err)
	}
	return accessToken, nil
}

func (s *TokenSource) fetchAndCacheToken(ctx context.Context, credentials ClientCredentials) (string, error) {
	refreshCtx, cancel := s.refreshContext(ctx)
	defer cancel()

	token, err := s.config.Fetch(refreshCtx, credentials)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(token.AccessToken) == "" {
		return "", s.config.EmptyAccessToken
	}

	expiresAt := s.config.Now().Add(max(token.ExpiresIn-s.config.RefreshMargin, 0))
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cached.accessToken != "" && s.config.Now().Before(s.cached.expiresAt) {
		return s.cached.accessToken, nil
	}
	s.cached = cachedToken{accessToken: token.AccessToken, expiresAt: expiresAt}
	return s.cached.accessToken, nil
}

func (s *TokenSource) refreshContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if s.config.RefreshTimeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, s.config.RefreshTimeout)
}
