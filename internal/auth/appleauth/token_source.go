package appleauth

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"
)

// defaultDeveloperTokenRefreshMargin is how long before a developer token's TTL
// it is regenerated. Apple's tokens are signed locally, so the margin only
// guards against clock skew on the receiving service.
const defaultDeveloperTokenRefreshMargin = 5 * time.Minute

// ErrDeveloperTokenSourceNotConfigured is answered when a token source was
// built without the full key material.
var ErrDeveloperTokenSourceNotConfigured = errors.New("apple music developer token source not configured")

// TokenSource generates Apple Music developer tokens from configured key
// material, caching each token until shortly before its TTL. The MusicKit
// developer token is a signed JWT rather than a fetched bearer token, so it
// needs no single-flight refresh: generation is local and cheap.
type TokenSource struct {
	config Config
	now    func() time.Time
	margin time.Duration

	mu        sync.Mutex
	token     string
	expiresAt time.Time
}

// NewTokenSource builds a cached developer token source for one Music Service.
func NewTokenSource(cfg Config) *TokenSource {
	margin := defaultDeveloperTokenRefreshMargin
	if cfg.TTL > 0 && cfg.TTL < margin {
		margin = cfg.TTL / 2
	}
	return &TokenSource{config: cfg, now: time.Now, margin: margin}
}

// Configured reports whether the key material is complete.
func (s *TokenSource) Configured() bool {
	if s == nil {
		return false
	}
	return strings.TrimSpace(s.config.KeyID) != "" &&
		strings.TrimSpace(s.config.TeamID) != "" &&
		strings.TrimSpace(s.config.PrivateKeyPath) != ""
}

// AccessToken returns the cached developer token, regenerating it when empty or
// within the refresh margin of expiry.
func (s *TokenSource) AccessToken(ctx context.Context) (string, error) {
	if s == nil {
		return "", ErrDeveloperTokenSourceNotConfigured
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	if s.token != "" && now.Before(s.expiresAt) {
		return s.token, nil
	}
	if !s.Configured() {
		return "", ErrDeveloperTokenSourceNotConfigured
	}
	token, err := GenerateDeveloperToken(s.config, now.UTC())
	if err != nil {
		return "", err
	}
	ttl := s.config.TTL
	if ttl <= 0 {
		ttl = defaultTokenTTL
	}
	s.token = token
	s.expiresAt = now.Add(ttl - s.margin)
	return s.token, nil
}
