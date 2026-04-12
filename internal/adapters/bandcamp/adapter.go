package bandcamp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/parse"
)

var (
	jsonLDPattern                = regexp.MustCompile(`(?s)<script type="application/ld\+json">\s*(\{.*?\})\s*</script>`)
	errUnexpectedBandcampService = errors.New("unexpected bandcamp service")
	errUnexpectedBandcampStatus  = errors.New("unexpected bandcamp status")
	errBandcampJSONLDNotFound    = errors.New("bandcamp json-ld not found")
	errMalformedBandcampJSONLD   = errors.New("malformed bandcamp json-ld")
)

// Option configures the Bandcamp adapter.
type Option func(*Adapter)

// WithSearchBaseURL overrides the Bandcamp search base URL.
func WithSearchBaseURL(baseURL string) Option {
	return func(adapter *Adapter) {
		adapter.searchBaseURL = strings.TrimRight(baseURL, "/")
	}
}

// Adapter implements Bandcamp source and metadata target operations via HTML scraping.
type Adapter struct {
	client        *http.Client
	searchBaseURL string
}

// New creates a Bandcamp adapter.
func New(client *http.Client, opts ...Option) *Adapter {
	if client == nil {
		client = http.DefaultClient
	}
	adapter := &Adapter{
		client:        client,
		searchBaseURL: "https://bandcamp.com",
	}
	for _, opt := range opts {
		opt(adapter)
	}
	return adapter
}

// Service returns the service implemented by this adapter.
func (a *Adapter) Service() model.ServiceName {
	return model.ServiceBandcamp
}

// ParseAlbumURL parses a Bandcamp album URL.
func (a *Adapter) ParseAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	parsed, err := parse.BandcampAlbumURL(raw)
	if err != nil {
		return nil, fmt.Errorf("parse bandcamp album url: %w", err)
	}
	return parsed, nil
}

// ParseSongURL parses a Bandcamp track URL.
func (a *Adapter) ParseSongURL(raw string) (*model.ParsedURL, error) {
	parsed, err := parse.BandcampSongURL(raw)
	if err != nil {
		return nil, fmt.Errorf("parse bandcamp song url: %w", err)
	}
	return parsed, nil
}

// SearchByUPC is not supported for Bandcamp.
func (a *Adapter) SearchByUPC(_ context.Context, _ string) ([]model.CandidateAlbum, error) {
	return nil, nil
}

// SearchByISRC is not supported for Bandcamp.
func (a *Adapter) SearchByISRC(_ context.Context, _ []string) ([]model.CandidateAlbum, error) {
	return nil, nil
}

// SearchSongByISRC is not supported for Bandcamp.
func (a *Adapter) SearchSongByISRC(_ context.Context, _ string) ([]model.CandidateSong, error) {
	return nil, nil
}
