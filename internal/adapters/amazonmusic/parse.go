package amazonmusic

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/xmbshwll/ariadne/internal/adapters/adapterutil"
	"github.com/xmbshwll/ariadne/internal/model"
)

var (
	errUnsupportedAmazonMusicHost = errors.New("unsupported amazon music host")
	errAmazonMusicNotAlbumURL     = errors.New("amazon music url is not an album url")
	errAmazonMusicNotSongURL      = errors.New("amazon music url is not a song url")
	errMissingAmazonMusicAlbumID  = errors.New("missing amazon music album id")
	errMissingAmazonMusicTrackID  = errors.New("missing amazon music track id")
)

func ParseAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	parsed, err := parseAmazonMusicURL(raw)
	if err != nil {
		return nil, err
	}

	segments := adapterutil.PathSegments(parsed.Path)
	if len(segments) != 2 || segments[0] != "albums" {
		return nil, fmt.Errorf("%w: %s", errAmazonMusicNotAlbumURL, raw)
	}

	asin := strings.TrimSpace(segments[1])
	if asin == "" {
		return nil, errMissingAmazonMusicAlbumID
	}

	return &model.ParsedAlbumURL{
		Service:      model.ServiceAmazonMusic,
		EntityType:   "album",
		ID:           asin,
		CanonicalURL: "https://music.amazon.com/albums/" + asin,
		RawURL:       raw,
	}, nil
}

func ParseSongURL(raw string) (*model.ParsedURL, error) {
	parsed, err := parseAmazonMusicURL(raw)
	if err != nil {
		return nil, err
	}

	segments := adapterutil.PathSegments(parsed.Path)
	asin := ""
	switch {
	case len(segments) == 2 && segments[0] == "tracks":
		asin = strings.TrimSpace(segments[1])
	case len(segments) == 2 && segments[0] == "albums":
		asin = strings.TrimSpace(parsed.Query().Get("trackAsin"))
	default:
		return nil, fmt.Errorf("%w: %s", errAmazonMusicNotSongURL, raw)
	}
	if asin == "" {
		return nil, errMissingAmazonMusicTrackID
	}

	return &model.ParsedURL{
		Service:      model.ServiceAmazonMusic,
		EntityType:   "song",
		ID:           asin,
		CanonicalURL: "https://music.amazon.com/tracks/" + asin,
		RawURL:       raw,
	}, nil
}

func parseAmazonMusicURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse amazon music url: %w", err)
	}

	host := strings.ToLower(parsed.Host)
	if host != "music.amazon.com" {
		return nil, fmt.Errorf("%w: %s", errUnsupportedAmazonMusicHost, parsed.Host)
	}
	return parsed, nil
}
