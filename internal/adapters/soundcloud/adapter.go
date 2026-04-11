package soundcloud

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
	defaultSiteBaseURL = "https://soundcloud.com"
	defaultAPIBaseURL  = "https://api-v2.soundcloud.com"
	searchLimit        = 5
)

var (
	hydrationPattern = regexp.MustCompile(`(?s)__sc_hydration\s*=\s*(\[.*?\]);`)
	scriptSrcPattern = regexp.MustCompile(`(?i)<script[^>]+src="([^"]+)"`)
	clientIDPattern  = regexp.MustCompile(`client_id[:=]\s*"([a-zA-Z0-9]{32})"`)

	errUnexpectedSoundCloudService   = errors.New("unexpected soundcloud service")
	errUnexpectedSoundCloudStatus    = errors.New("unexpected soundcloud status")
	errUnexpectedSoundCloudAPIStatus = errors.New("unexpected soundcloud api status")
	errSoundCloudClientIDNotFound    = errors.New("soundcloud client id not found")
	errSoundCloudHydrationNotFound   = errors.New("soundcloud hydration payload not found")
	errSoundCloudPlaylistNotFound    = errors.New("soundcloud playlist hydration not found")
	errSoundCloudTrackNotFound       = errors.New("soundcloud track hydration not found")
)

type Option func(*Adapter)

func WithSiteBaseURL(baseURL string) Option {
	return func(adapter *Adapter) {
		adapter.siteBaseURL = strings.TrimRight(baseURL, "/")
	}
}

func WithAPIBaseURL(baseURL string) Option {
	return func(adapter *Adapter) {
		adapter.apiBaseURL = strings.TrimRight(baseURL, "/")
	}
}

type Adapter struct {
	client      *http.Client
	siteBaseURL string
	apiBaseURL  string

	clientIDMu sync.Mutex
	clientID   string
}

func New(client *http.Client, opts ...Option) *Adapter {
	if client == nil {
		client = http.DefaultClient
	}
	adapter := &Adapter{
		client:      client,
		siteBaseURL: defaultSiteBaseURL,
		apiBaseURL:  defaultAPIBaseURL,
	}
	for _, opt := range opts {
		opt(adapter)
	}
	return adapter
}

func (a *Adapter) Service() model.ServiceName {
	return model.ServiceSoundCloud
}

func (a *Adapter) ParseAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	parsed, err := parse.SoundCloudAlbumURL(raw)
	if err != nil {
		return nil, fmt.Errorf("parse soundcloud album url: %w", err)
	}
	return parsed, nil
}

func (a *Adapter) ParseSongURL(raw string) (*model.ParsedURL, error) {
	parsed, err := parse.SoundCloudSongURL(raw)
	if err != nil {
		return nil, fmt.Errorf("parse soundcloud song url: %w", err)
	}
	return parsed, nil
}
