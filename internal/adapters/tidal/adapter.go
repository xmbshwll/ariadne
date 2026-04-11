package tidal

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/parse"
)

const (
	defaultAPIBaseURL  = "https://openapi.tidal.com/v2"
	defaultAuthBaseURL = "https://auth.tidal.com/v1"
	defaultCountryCode = "US"
	searchLimit        = 5
)

var (
	ErrCredentialsNotConfigured = errors.New("tidal credentials not configured")

	errUnexpectedTIDALService     = errors.New("unexpected tidal service")
	errTIDALAlbumNotFound         = errors.New("tidal album not found")
	errTIDALTrackNotFound         = errors.New("tidal track not found")
	errUnexpectedTIDALAPIStatus   = errors.New("unexpected tidal api status")
	errUnexpectedTIDALTokenStatus = errors.New("unexpected tidal token status")
	errMalformedTIDALAPIResponse  = errors.New("malformed tidal api response")
	errEmptyTIDALAccessToken      = errors.New("empty tidal access token")
)

type Option func(*Adapter)

func WithCredentials(clientID string, clientSecret string) Option {
	return func(adapter *Adapter) {
		adapter.clientID = strings.TrimSpace(clientID)
		adapter.clientSecret = strings.TrimSpace(clientSecret)
	}
}

func WithAPIBaseURL(baseURL string) Option {
	return func(adapter *Adapter) {
		adapter.apiBaseURL = strings.TrimRight(baseURL, "/")
	}
}

func WithAuthBaseURL(baseURL string) Option {
	return func(adapter *Adapter) {
		adapter.authBaseURL = strings.TrimRight(baseURL, "/")
	}
}

func WithDefaultCountryCode(countryCode string) Option {
	return func(adapter *Adapter) {
		adapter.defaultCountryCode = normalizeCountryCode(countryCode)
	}
}

type Adapter struct {
	client             *http.Client
	clientID           string
	clientSecret       string
	apiBaseURL         string
	authBaseURL        string
	defaultCountryCode string

	tokenMu    sync.Mutex
	token      cachedToken
	tokenGroup singleflight.Group
}

type cachedToken struct {
	accessToken string
	expiresAt   time.Time
}

func New(client *http.Client, opts ...Option) *Adapter {
	if client == nil {
		client = http.DefaultClient
	}
	adapter := &Adapter{
		client:             client,
		apiBaseURL:         defaultAPIBaseURL,
		authBaseURL:        defaultAuthBaseURL,
		defaultCountryCode: defaultCountryCode,
	}
	for _, opt := range opts {
		opt(adapter)
	}
	return adapter
}

func (a *Adapter) Service() model.ServiceName {
	return model.ServiceTIDAL
}

func (a *Adapter) ParseAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	parsed, err := parse.TIDALAlbumURL(raw)
	if err != nil {
		return nil, fmt.Errorf("parse tidal album url: %w", err)
	}
	return parsed, nil
}

func (a *Adapter) ParseSongURL(raw string) (*model.ParsedURL, error) {
	parsed, err := parse.TIDALSongURL(raw)
	if err != nil {
		return nil, fmt.Errorf("parse tidal song url: %w", err)
	}
	return parsed, nil
}
