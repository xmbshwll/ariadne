package adapterutil

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

const defaultCredentialTokenRefreshMargin = 30 * time.Second

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

	result, err, _ := s.group.Do(s.config.SingleflightKey, func() (any, error) {
		if accessToken, ok := s.cachedAccessToken(); ok {
			return accessToken, nil
		}
		return s.refreshAccessToken(ctx, credentials)
	})
	if err != nil {
		//nolint:wrapcheck // Preserve service-specific fetch errors across singleflight.
		return "", err
	}
	accessToken, ok := result.(string)
	if !ok {
		return "", errCredentialTokenResultInvalid
	}
	return accessToken, nil
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

func (s *CredentialTokenSource) refreshContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if s.config.RefreshTimeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, s.config.RefreshTimeout)
}
