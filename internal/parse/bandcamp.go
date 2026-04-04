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
		return nil, fmt.Errorf("missing bandcamp host")
	}

	segments := pathSegments(parsed.Path)
	if len(segments) < 2 || segments[0] != "album" {
		return nil, fmt.Errorf("bandcamp url is not an album url: %s", raw)
	}

	slug := segments[1]
	canonicalURL := fmt.Sprintf("%s://%s/album/%s", parsed.Scheme, parsed.Host, slug)
	if parsed.Scheme == "" {
		canonicalURL = fmt.Sprintf("https://%s/album/%s", parsed.Host, slug)
	}

	return &model.ParsedAlbumURL{
		Service:      model.ServiceBandcamp,
		EntityType:   "album",
		ID:           slug,
		CanonicalURL: canonicalURL,
		RawURL:       raw,
	}, nil
}
