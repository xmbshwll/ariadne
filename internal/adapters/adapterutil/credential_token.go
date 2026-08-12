package adapterutil

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

const (
	defaultCredentialTokenRefreshMargin = 30 * time.Second
	defaultMaxRefreshAttempts           = 3
	defaultRefreshRetryBackoff          = 250 * time.Millisecond
)

var (
	errCredentialTokenSourceNotConfigured = errors.New("credential token source not configured")
	errCredentialTokenResultInvalid       = errors.New("credential token result invalid")
)

type ClientCredentials struct {
	ClientID     string
	ClientSecret string
}

func (c ClientCredentials) Configured() bool {
	return strings.TrimSpace(c.ClientID) != "" && strings.TrimSpace(c.ClientSecret) != ""
}

func (c ClientCredentials) BasicAuthorization() string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(c.ClientID+":"+c.ClientSecret))
}

type CredentialToken struct {
	AccessToken string
	ExpiresIn   time.Duration
}

type CredentialTokenFetchFunc func(context.Context, ClientCredentials) (CredentialToken, error)

// ClientCredentialsTokenConfig carries the per-service varying bits of the
// common client-credentials token flow; NewClientCredentialsTokenSource owns
// the wiring defaults.
type ClientCredentialsTokenConfig struct {
	Service            string
	ClientID           string
	ClientSecret       string
	MissingCredentials error
	EmptyAccessToken   error
	Fetch              CredentialTokenFetchFunc
	RefreshTimeout     time.Duration
}

// NewClientCredentialsTokenSource builds a CredentialTokenSource for a Music
// Service using the client-credentials flow.
func NewClientCredentialsTokenSource(config ClientCredentialsTokenConfig) *CredentialTokenSource {
	return NewCredentialTokenSource(CredentialTokenSourceConfig{
		Credentials: func() ClientCredentials {
			return ClientCredentials{ClientID: config.ClientID, ClientSecret: config.ClientSecret}
		},
		MissingCredentials: config.MissingCredentials,
		EmptyAccessToken:   config.EmptyAccessToken,
		IsEmptyAccessToken: func(accessToken string) bool { return strings.TrimSpace(accessToken) == "" },
		Fetch:              config.Fetch,
		RefreshTimeout:     config.RefreshTimeout,
		SingleflightKey:    config.Service + "-token",
	})
}

type CredentialTokenSourceConfig struct {
	Credentials        func() ClientCredentials
	MissingCredentials error
	EmptyAccessToken   error
	IsEmptyAccessToken func(string) bool
	Fetch              CredentialTokenFetchFunc
	RefreshMargin      time.Duration
	RefreshTimeout     time.Duration
	SingleflightKey    string
	Now                func() time.Time
	// MaxRefreshAttempts is the maximum number of fetch attempts for token refresh.
	// Transient HTTP errors (StatusBadGateway, StatusServiceUnavailable, StatusGatewayTimeout)
	// trigger retries with exponential backoff.
	// When zero or negative, defaults to 3.
	MaxRefreshAttempts int
	// RefreshRetryBackoff is the initial backoff between retry attempts, doubled each attempt.
	// When zero or negative, defaults to 250ms.
	RefreshRetryBackoff time.Duration
}

type CredentialTokenSource struct {
	config CredentialTokenSourceConfig
	mu     sync.Mutex
	group  singleflight.Group
	cached cachedCredentialToken
}

type cachedCredentialToken struct {
	accessToken string
	expiresAt   time.Time
}

func NewCredentialTokenSource(config CredentialTokenSourceConfig) *CredentialTokenSource {
	if config.RefreshMargin == 0 {
		config.RefreshMargin = defaultCredentialTokenRefreshMargin
	}
	if config.SingleflightKey == "" {
		config.SingleflightKey = "credential-token"
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.IsEmptyAccessToken == nil {
		config.IsEmptyAccessToken = func(accessToken string) bool { return accessToken == "" }
	}
	if config.MaxRefreshAttempts <= 0 {
		config.MaxRefreshAttempts = defaultMaxRefreshAttempts
	}
	if config.RefreshRetryBackoff <= 0 {
		config.RefreshRetryBackoff = defaultRefreshRetryBackoff
	}
	return &CredentialTokenSource{config: config}
}

func (s *CredentialTokenSource) CredentialsConfigured() bool {
	if s == nil || s.config.Credentials == nil {
		return false
	}
	return s.config.Credentials().Configured()
}

func (s *CredentialTokenSource) AccessToken(ctx context.Context) (string, error) {
	if s == nil || s.config.Credentials == nil || s.config.Fetch == nil {
		return "", errCredentialTokenSourceNotConfigured
	}
	if ctx == nil {
		ctx = context.Background()
	}

	credentials := s.config.Credentials()
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

	detachedCtx := context.WithoutCancel(ctx)
	resultCh := s.group.DoChan(s.config.SingleflightKey, func() (any, error) {
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
			return "", errCredentialTokenResultInvalid
		}
		return accessToken, nil
	}
}

func (s *CredentialTokenSource) cachedAccessToken() (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cached.accessToken == "" || !s.config.Now().Before(s.cached.expiresAt) {
		return "", false
	}
	return s.cached.accessToken, true
}

func (s *CredentialTokenSource) refreshAccessToken(ctx context.Context, credentials ClientCredentials) (string, error) {
	var lastErr error
	for attempt := range s.config.MaxRefreshAttempts {
		token, err := s.fetchAndCacheToken(ctx, credentials)
		if err == nil {
			return token, nil
		}
		lastErr = err
		if attempt == s.config.MaxRefreshAttempts-1 || !IsTransientHTTPError(err) {
			break
		}
		if waitErr := waitForRefreshRetry(ctx, attempt, s.config.RefreshRetryBackoff); waitErr != nil {
			return "", waitErr
		}
	}
	return "", lastErr
}

func (s *CredentialTokenSource) fetchAndCacheToken(ctx context.Context, credentials ClientCredentials) (string, error) {
	refreshCtx, cancel := s.refreshContext(ctx)
	defer cancel()

	token, err := s.config.Fetch(refreshCtx, credentials)
	if err != nil {
		return "", err
	}
	if s.config.IsEmptyAccessToken(token.AccessToken) {
		return "", s.config.EmptyAccessToken
	}

	expiresAt := s.config.Now().Add(max(token.ExpiresIn-s.config.RefreshMargin, 0))
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cached.accessToken != "" && s.config.Now().Before(s.cached.expiresAt) {
		return s.cached.accessToken, nil
	}
	s.cached = cachedCredentialToken{accessToken: token.AccessToken, expiresAt: expiresAt}
	return s.cached.accessToken, nil
}

func waitForRefreshRetry(ctx context.Context, attempt int, baseBackoff time.Duration) error {
	delay := baseBackoff * time.Duration(1<<attempt)
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return fmt.Errorf("wait for credential token refresh retry: %w", ctx.Err())
	case <-timer.C:
		return nil
	}
}

func (s *CredentialTokenSource) refreshContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if s.config.RefreshTimeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, s.config.RefreshTimeout)
}
