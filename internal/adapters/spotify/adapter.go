package spotify

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/parse"
)

const (
	defaultWebBaseURL  = "https://open.spotify.com"
	defaultAPIBaseURL  = "https://api.spotify.com/v1"
	defaultAuthBaseURL = "https://accounts.spotify.com/api"
	searchLimit        = 5
)

var (
	initialStatePattern = regexp.MustCompile(`<script id="initialState" type="text/plain">([^<]+)</script>`)

	errUnexpectedSpotifyService     = errors.New("unexpected spotify service")
	errUnexpectedSpotifyStatus      = errors.New("unexpected spotify status")
	errSpotifyAlbumNotFound         = errors.New("spotify album not found")
	errSpotifyTrackNotFound         = errors.New("spotify track not found")
	errUnexpectedSpotifyAPIStatus   = errors.New("unexpected spotify api status")
	errUnexpectedSpotifyTokenStatus = errors.New("unexpected spotify token status")
	errEmptySpotifyAccessToken      = errors.New("empty spotify access token")
	errInitialStateScriptNotFound   = errors.New("initial state script not found")

	// ErrCredentialsNotConfigured indicates that a Web API operation requires Spotify credentials.
	ErrCredentialsNotConfigured = errors.New("spotify credentials not configured")
)

// Option configures the Spotify adapter.
type Option func(*Adapter)

// WithCredentials sets Spotify client credentials explicitly.
func WithCredentials(clientID string, clientSecret string) Option {
	return func(adapter *Adapter) {
		adapter.clientID = clientID
		adapter.clientSecret = clientSecret
	}
}

// WithAPIBaseURL overrides the Spotify Web API base URL.
func WithAPIBaseURL(baseURL string) Option {
	return func(adapter *Adapter) {
		adapter.apiBaseURL = strings.TrimRight(baseURL, "/")
	}
}

// WithAuthBaseURL overrides the Spotify auth API base URL.
func WithAuthBaseURL(baseURL string) Option {
	return func(adapter *Adapter) {
		adapter.authBaseURL = strings.TrimRight(baseURL, "/")
	}
}

// WithWebBaseURL overrides the Spotify web base URL used for bootstrap fetches.
func WithWebBaseURL(baseURL string) Option {
	return func(adapter *Adapter) {
		adapter.webBaseURL = strings.TrimRight(baseURL, "/")
	}
}

// Adapter implements Spotify source and target operations.
type Adapter struct {
	client       *http.Client
	clientID     string
	clientSecret string
	apiBaseURL   string
	authBaseURL  string
	webBaseURL   string

	tokenMu sync.Mutex
	token   cachedToken
}

// New creates a Spotify adapter.
func New(client *http.Client, opts ...Option) *Adapter {
	if client == nil {
		client = http.DefaultClient
	}
	adapter := &Adapter{
		client:      client,
		apiBaseURL:  defaultAPIBaseURL,
		authBaseURL: defaultAuthBaseURL,
		webBaseURL:  defaultWebBaseURL,
	}
	for _, opt := range opts {
		opt(adapter)
	}
	return adapter
}

// Service returns the service implemented by this adapter.
func (a *Adapter) Service() model.ServiceName {
	return model.ServiceSpotify
}

// ParseAlbumURL parses a Spotify album URL.
func (a *Adapter) ParseAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	parsed, err := parse.SpotifyAlbumURL(raw)
	if err != nil {
		return nil, fmt.Errorf("parse spotify album url: %w", err)
	}
	return parsed, nil
}

// ParseSongURL parses a Spotify track URL.
func (a *Adapter) ParseSongURL(raw string) (*model.ParsedAlbumURL, error) {
	parsed, err := parse.SpotifySongURL(raw)
	if err != nil {
		return nil, fmt.Errorf("parse spotify song url: %w", err)
	}
	return parsed, nil
}
