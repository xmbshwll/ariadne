package parse

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/xmbshwll/ariadne/internal/model"
)

// AppleMusicAlbumURL parses an Apple Music album URL into the shared parsed representation.
func AppleMusicAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse apple music url: %w", err)
	}

	host := strings.ToLower(parsed.Host)
	if host != "music.apple.com" {
		return nil, fmt.Errorf("unsupported apple music host: %s", parsed.Host)
	}

	segments := pathSegments(parsed.Path)
	if len(segments) < 4 {
		return nil, fmt.Errorf("invalid apple music album path: %s", parsed.Path)
	}
	if segments[1] != "album" {
		return nil, fmt.Errorf("apple music url is not an album url: %s", raw)
	}

	storefront := segments[0]
	id := segments[len(segments)-1]
	if storefront == "" || id == "" {
		return nil, fmt.Errorf("missing storefront or album id")
	}

	canonicalURL := fmt.Sprintf("https://music.apple.com/%s/album/%s/%s", storefront, segments[len(segments)-2], id)
	if len(segments) == 3 {
		canonicalURL = fmt.Sprintf("https://music.apple.com/%s/album/%s", storefront, id)
	}

	return &model.ParsedAlbumURL{
		Service:      model.ServiceAppleMusic,
		EntityType:   "album",
		ID:           id,
		CanonicalURL: canonicalURL,
		RegionHint:   storefront,
		RawURL:       raw,
	}, nil
}
