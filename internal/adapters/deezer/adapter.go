package deezer

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/parse"
)

const (
	defaultBaseURL        = "https://api.deezer.com"
	metadataSearchLimit   = 5
	identifierSearchLimit = 5
)

var (
	errUnexpectedDeezerService = errors.New("unexpected deezer service")
	errUnexpectedDeezerStatus  = errors.New("unexpected deezer status")
	errMalformedDeezerResponse = errors.New("malformed deezer response")
	errDeezerAlbumNotFound     = errors.New("deezer album not found")
	errDeezerTrackNotFound     = errors.New("deezer track not found")
)

// Adapter implements Deezer source operations.
type Adapter struct {
	baseURL string
	client  *http.Client
}

// New creates a Deezer adapter.
func New(client *http.Client) *Adapter {
	return newAdapter(client, defaultBaseURL)
}

func newAdapter(client *http.Client, baseURL string) *Adapter {
	if client == nil {
		client = http.DefaultClient
	}
	return &Adapter{
		baseURL: baseURL,
		client:  client,
	}
}

// Service returns the service implemented by this adapter.
func (a *Adapter) Service() model.ServiceName {
	return model.ServiceDeezer
}

// ParseAlbumURL parses a Deezer album URL.
func (a *Adapter) ParseAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	parsed, err := parse.DeezerAlbumURL(raw)
	if err != nil {
		return nil, fmt.Errorf("parse deezer album url: %w", err)
	}
	return parsed, nil
}

// ParseSongURL parses a Deezer track URL.
func (a *Adapter) ParseSongURL(raw string) (*model.ParsedURL, error) {
	parsed, err := parse.DeezerSongURL(raw)
	if err != nil {
		return nil, fmt.Errorf("parse deezer song url: %w", err)
	}
	return parsed, nil
}
