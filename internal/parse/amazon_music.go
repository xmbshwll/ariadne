package parse

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/xmbshwll/ariadne/internal/model"
)

// AmazonMusicAlbumURL parses Amazon Music album URLs into the shared parsed representation.
func AmazonMusicAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	parsed, err := parseAmazonMusicURL(raw)
	if err != nil {
		return nil, err
	}

	segments := pathSegments(parsed.Path)
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

// AmazonMusicSongURL parses Amazon Music track URLs into the shared parsed representation.
func AmazonMusicSongURL(raw string) (*model.ParsedURL, error) {
	parsed, err := parseAmazonMusicURL(raw)
	if err != nil {
		return nil, err
	}

	segments := pathSegments(parsed.Path)
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
