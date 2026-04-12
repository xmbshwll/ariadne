package youtubemusic

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

const (
	defaultBaseURL = "https://music.youtube.com"
	searchLimit    = 5
)

var (
	canonicalURLPattern               = regexp.MustCompile(`(?i)<link rel="canonical" href="([^"]+)"`)
	ogTitlePattern                    = regexp.MustCompile(`(?i)<meta property="og:title" content="([^"]+)"`)
	ogImagePattern                    = regexp.MustCompile(`(?i)<meta property="og:image" content="([^"]+)"`)
	subtitleArtistPattern             = regexp.MustCompile(`subtitle\\x22:\\x7b\\x22runs\\x22:\\x5b\\x7b\\x22text\\x22:\\x22Album\\x22\\x7d,\\x7b\\x22text\\x22:\\x22 .*?\\x7d,\\x7b\\x22text\\x22:\\x22([^\\]+?)\\x22`)
	trackTitlePattern                 = regexp.MustCompile(`musicResponsiveListItemFlexColumnRenderer\\x22:\\x7b\\x22text\\x22:\\x7b\\x22runs\\x22:\\x5b\\x7b\\x22text\\x22:\\x22([^\\]+?)\\x22`)
	albumResultPattern                = regexp.MustCompile(`title\\x22:\\x7b\\x22runs\\x22:\\x5b\\x7b\\x22text\\x22:\\x22([^\\]+?)\\x22,\\x22navigationEndpoint\\x22:\\x7b.*?browseId\\x22:\\x22([^\\]+?)\\x22.*?pageType\\x22:\\x22MUSIC_PAGE_TYPE_ALBUM\\x22.*?subtitle\\x22:\\x7b\\x22runs\\x22:\\x5b\\x7b\\x22text\\x22:\\x22Album\\x22\\x7d,\\x7b\\x22text\\x22:\\x22 .*?\\x7d,\\x7b\\x22text\\x22:\\x22([^\\]+?)\\x22`)
	errUnexpectedYouTubeMusicService  = errors.New("unexpected youtube music service")
	errUnexpectedYouTubeMusicStatus   = errors.New("unexpected youtube music status")
	errMalformedYouTubeMusicPage      = errors.New("malformed youtube music page")
	errYouTubeMusicAlbumTitleNotFound = errors.New("youtube music album title not found")
	errNilYouTubeMusicCanonicalAlbum  = errors.New("youtube music adapter returned nil canonical album")
)

type Option func(*Adapter)

func WithBaseURL(baseURL string) Option {
	return func(adapter *Adapter) {
		adapter.baseURL = strings.TrimRight(baseURL, "/")
	}
}

type Adapter struct {
	client  *http.Client
	baseURL string
}

func New(client *http.Client, opts ...Option) *Adapter {
	if client == nil {
		client = http.DefaultClient
	}
	adapter := &Adapter{client: client, baseURL: defaultBaseURL}
	for _, opt := range opts {
		opt(adapter)
	}
	return adapter
}

func (a *Adapter) Service() model.ServiceName {
	return model.ServiceYouTubeMusic
}

func (a *Adapter) ParseAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	parsed, err := parse.YouTubeMusicAlbumURL(raw)
	if err != nil {
		return nil, fmt.Errorf("parse youtube music album url: %w", err)
	}
	return parsed, nil
}

func (a *Adapter) SearchByUPC(_ context.Context, _ string) ([]model.CandidateAlbum, error) {
	return nil, nil
}

func (a *Adapter) SearchByISRC(_ context.Context, _ []string) ([]model.CandidateAlbum, error) {
	return nil, nil
}
