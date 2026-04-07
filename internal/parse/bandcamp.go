package parse

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/xmbshwll/ariadne/internal/model"
)

// BandcampAlbumURL parses a Bandcamp album URL into the shared parsed representation.
func BandcampAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse bandcamp url: %w", err)
	}

	host := strings.ToLower(parsed.Host)
	if host == "" {
		return nil, errMissingBandcampHost
	}

	segments := pathSegments(parsed.Path)
	if len(segments) < 2 || segments[0] != albumPathSegment {
		return nil, fmt.Errorf("%w: %s", errBandcampNotAlbumURL, raw)
	}

	slug := segments[1]
	canonicalURL := fmt.Sprintf("%s://%s/%s/%s", parsed.Scheme, parsed.Host, albumPathSegment, slug)
	if parsed.Scheme == "" {
		canonicalURL = fmt.Sprintf("https://%s/%s/%s", parsed.Host, albumPathSegment, slug)
	}

	return &model.ParsedAlbumURL{
		Service:      model.ServiceBandcamp,
		EntityType:   "album",
		ID:           slug,
		CanonicalURL: canonicalURL,
		RawURL:       raw,
	}, nil
}
