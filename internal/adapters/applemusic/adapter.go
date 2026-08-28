package applemusic

import (
	"errors"
	"net/http"
	"strings"

	"github.com/xmbshwll/ariadne/internal/adapters/base"
	"github.com/xmbshwll/ariadne/internal/auth/appleauth"
	"github.com/xmbshwll/ariadne/internal/model"
)

const (
	defaultLookupBaseURL = "https://itunes.apple.com"
	defaultAPIBaseURL    = "https://api.music.apple.com/v1"
	searchLimit          = 5
	EntitySong           = "song"
	wrapperTypeTrack     = "track"
)

var (
	errUnexpectedAppleMusicService = errors.New("unexpected apple music service")
	errAppleMusicAlbumNotFound     = errors.New("apple music album not found")
	ErrAppleMusicSongNotFound      = errors.New("apple music song not found")
	errUnexpectedAppleMusicStatus  = errors.New("unexpected apple music status")

	errUnexpectedAppleMusicOfficialStatus  = errors.New("unexpected apple music official status")
	ErrMalformedAppleMusicOfficialResponse = errors.New("malformed apple music official response")
	errAppleMusicOfficialAlbumNotFound     = errors.New("apple music official album not found")

	// ErrCredentialsNotConfigured indicates that an Apple Music official API operation requires developer token credentials.
	ErrCredentialsNotConfigured = errors.New("apple music credentials not configured")
)

// Option configures the Apple Music adapter.
type Option func(*Adapter)

// WithLookupBaseURL overrides the iTunes lookup API base URL.
func WithLookupBaseURL(baseURL string) Option {
	return func(adapter *Adapter) {
		adapter.lookupBaseURL = strings.TrimRight(baseURL, "/")
	}
}

// WithDefaultStorefront sets the default Apple Music storefront used when the
// source album does not already carry a storefront hint.
func WithDefaultStorefront(storefront string) Option {
	return func(adapter *Adapter) {
		adapter.defaultStorefront = strings.ToLower(strings.TrimSpace(storefront))
	}
}

// WithAPIBaseURL overrides the official Apple Music API base URL.
func WithAPIBaseURL(baseURL string) Option {
	return func(adapter *Adapter) {
		adapter.apiBaseURL = strings.TrimRight(baseURL, "/")
	}
}

// WithDeveloperTokenAuth enables official Apple Music API calls by generating
// MusicKit developer tokens from the provided .p8 key material.
func WithDeveloperTokenAuth(keyID string, teamID string, privateKeyPath string) Option {
	return func(adapter *Adapter) {
		adapter.developerTokens = appleauth.NewTokenSource(appleauth.Config{
			KeyID:          strings.TrimSpace(keyID),
			TeamID:         strings.TrimSpace(teamID),
			PrivateKeyPath: strings.TrimSpace(privateKeyPath),
		})
	}
}

// Adapter implements Apple Music source operations using the public lookup API.
type Adapter struct {
	base.Unsupported

	client            *http.Client
	lookupBaseURL     string
	apiBaseURL        string
	defaultStorefront string
	developerTokens   *appleauth.TokenSource
}

// New creates an Apple Music adapter.
func New(client *http.Client, opts ...Option) *Adapter {
	if client == nil {
		client = http.DefaultClient
	}
	adapter := &Adapter{
		Unsupported:       base.Unsupported{ServiceName: model.ServiceAppleMusic},
		client:            client,
		lookupBaseURL:     defaultLookupBaseURL,
		apiBaseURL:        defaultAPIBaseURL,
		defaultStorefront: "us",
	}
	for _, opt := range opts {
		opt(adapter)
	}
	return adapter
}

// ParseAlbumURL parses an Apple Music album URL.
func (a *Adapter) ParseAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	return ParseAlbumURL(raw)
}

// ParseSongURL parses an Apple Music song URL.
func (a *Adapter) ParseSongURL(raw string) (*model.ParsedURL, error) {
	return ParseSongURL(raw)
}
