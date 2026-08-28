package deezer

import (
	"errors"
	"net/http"

	"github.com/xmbshwll/ariadne/internal/adapters/base"
	"github.com/xmbshwll/ariadne/internal/model"
)

const (
	defaultBaseURL        = "https://api.deezer.com"
	metadataSearchLimit   = 5
	identifierSearchLimit = 5
)

var (
	errUnexpectedDeezerService = errors.New("unexpected deezer service")
	errUnexpectedDeezerStatus  = errors.New("unexpected deezer status")
	ErrMalformedDeezerResponse = errors.New("malformed deezer response")
	errDeezerAlbumNotFound     = errors.New("deezer album not found")
	errDeezerTrackNotFound     = errors.New("deezer track not found")
)

// Option customizes a Deezer adapter.
type Option func(*Adapter)

// WithBaseURL redirects every Deezer API request to baseURL, for tests.
func WithBaseURL(baseURL string) Option {
	return func(a *Adapter) {
		a.baseURL = baseURL
	}
}

// Adapter implements Deezer source operations.
type Adapter struct {
	base.Unsupported

	baseURL string
	client  *http.Client
}

// New creates a Deezer adapter.
func New(client *http.Client, opts ...Option) *Adapter {
	if client == nil {
		client = http.DefaultClient
	}
	adapter := &Adapter{
		Unsupported: base.Unsupported{ServiceName: model.ServiceDeezer},
		baseURL:     defaultBaseURL,
		client:      client,
	}
	for _, opt := range opts {
		opt(adapter)
	}
	return adapter
}

// ParseAlbumURL parses a Deezer album URL.
func (a *Adapter) ParseAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	return ParseAlbumURL(raw)
}

// ParseSongURL parses a Deezer track URL.
func (a *Adapter) ParseSongURL(raw string) (*model.ParsedURL, error) {
	return ParseSongURL(raw)
}
