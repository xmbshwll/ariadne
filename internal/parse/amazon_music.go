package parse

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/xmbshwll/ariadne/internal/model"
)

// AmazonMusicAlbumURL parses Amazon Music album URLs into the shared parsed representation.
func AmazonMusicAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse amazon music url: %w", err)
	}

	host := strings.ToLower(parsed.Host)
	if host != "music.amazon.com" {
		return nil, fmt.Errorf("%w: %s", errUnsupportedAmazonMusicHost, parsed.Host)
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
