package bandcamp

import (
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/xmbshwll/ariadne/internal/adapters/base"
	"github.com/xmbshwll/ariadne/internal/model"
)

var (
	jsonLDPattern                      = regexp.MustCompile(`(?s)<script type="application/ld\+json">\s*(\{.*?\})\s*</script>`)
	errUnexpectedBandcampService       = errors.New("unexpected bandcamp service")
	errUnexpectedBandcampStatus        = errors.New("unexpected bandcamp status")
	errBandcampJSONLDNotFound          = errors.New("bandcamp json-ld not found")
	ErrMalformedBandcampJSONLD         = errors.New("malformed bandcamp json-ld")
	errMalformedBandcampSearchResponse = errors.New("malformed bandcamp search response")
)

// Option configures the Bandcamp adapter.
type Option func(*Adapter)

// WithSearchBaseURL overrides the Bandcamp search base URL.
func WithSearchBaseURL(baseURL string) Option {
	return func(adapter *Adapter) {
		adapter.searchBaseURL = strings.TrimRight(baseURL, "/")
	}
}

// Adapter implements Bandcamp source and metadata target operations.
type Adapter struct {
	base.Unsupported

	client        *http.Client
	searchBaseURL string
}

// New creates a Bandcamp adapter.
func New(client *http.Client, opts ...Option) *Adapter {
	if client == nil {
		client = http.DefaultClient
	}
	adapter := &Adapter{
		Unsupported:   base.Unsupported{ServiceName: model.ServiceBandcamp},
		client:        client,
		searchBaseURL: "https://bandcamp.com",
	}
	for _, opt := range opts {
		opt(adapter)
	}
	return adapter
}

// ParseAlbumURL parses a Bandcamp album URL.
func (a *Adapter) ParseAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	return ParseAlbumURL(raw)
}

// ParseSongURL parses a Bandcamp track URL.
func (a *Adapter) ParseSongURL(raw string) (*model.ParsedURL, error) {
	return ParseSongURL(raw)
}
