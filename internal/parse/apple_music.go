package parse

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/xmbshwll/ariadne/internal/model"
)

const albumPathSegment = "album"

// AppleMusicAlbumURL parses an Apple Music album URL into the shared parsed representation.
func AppleMusicAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse apple music url: %w", err)
	}

	host := strings.ToLower(parsed.Host)
	if host != "music.apple.com" {
		return nil, fmt.Errorf("%w: %s", errUnsupportedAppleMusicHost, parsed.Host)
	}

	segments := pathSegments(parsed.Path)
	if len(segments) < 4 {
		return nil, fmt.Errorf("%w: %s", errInvalidAppleMusicAlbumPath, parsed.Path)
	}
	if segments[1] != albumPathSegment {
		return nil, fmt.Errorf("%w: %s", errAppleMusicNotAlbumURL, raw)
	}

	storefront := segments[0]
	id := segments[len(segments)-1]
	if storefront == "" || id == "" {
		return nil, errMissingAppleMusicStorefrontOrAlbumID
	}

	canonicalURL := fmt.Sprintf("https://music.apple.com/%s/%s/%s/%s", storefront, albumPathSegment, segments[len(segments)-2], id)
	if len(segments) == 3 {
		canonicalURL = fmt.Sprintf("https://music.apple.com/%s/%s/%s", storefront, albumPathSegment, id)
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
