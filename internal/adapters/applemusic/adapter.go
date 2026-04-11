package applemusic

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/parse"
)

const (
	defaultLookupBaseURL = "https://itunes.apple.com"
	defaultAPIBaseURL    = "https://api.music.apple.com/v1"
	searchLimit          = 5
	entitySong           = "song"
	wrapperTypeTrack     = "track"
)

var (
	errUnexpectedAppleMusicService = errors.New("unexpected apple music service")
	errAppleMusicAlbumNotFound     = errors.New("apple music album not found")
	errAppleMusicSongNotFound      = errors.New("apple music song not found")
	errUnexpectedAppleMusicStatus  = errors.New("unexpected apple music status")

	errUnexpectedAppleMusicOfficialStatus = errors.New("unexpected apple music official status")
	errAppleMusicOfficialAlbumNotFound    = errors.New("apple music official album not found")

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
		adapter.appleMusicKeyID = strings.TrimSpace(keyID)
		adapter.appleMusicTeamID = strings.TrimSpace(teamID)
		adapter.appleMusicPrivateKeyPath = strings.TrimSpace(privateKeyPath)
	}
}

// Adapter implements Apple Music source operations using the public lookup API.
type Adapter struct {
	client                   *http.Client
	lookupBaseURL            string
	apiBaseURL               string
	defaultStorefront        string
	appleMusicKeyID          string
	appleMusicTeamID         string
	appleMusicPrivateKeyPath string
	tokenMu                  sync.Mutex
	cachedToken              string
	tokenExpiresAt           time.Time
}

// New creates an Apple Music adapter.
func New(client *http.Client, opts ...Option) *Adapter {
	if client == nil {
		client = http.DefaultClient
	}
	adapter := &Adapter{
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

// Service returns the service implemented by this adapter.
func (a *Adapter) Service() model.ServiceName {
	return model.ServiceAppleMusic
}

// ParseAlbumURL parses an Apple Music album URL.
func (a *Adapter) ParseAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	parsed, err := parse.AppleMusicAlbumURL(raw)
	if err != nil {
		return nil, fmt.Errorf("parse apple music album url: %w", err)
	}
	return parsed, nil
}

// ParseSongURL parses an Apple Music song URL.
func (a *Adapter) ParseSongURL(raw string) (*model.ParsedAlbumURL, error) {
	parsed, err := parse.AppleMusicSongURL(raw)
	if err != nil {
		return nil, fmt.Errorf("parse apple music song url: %w", err)
	}
	return parsed, nil
}
