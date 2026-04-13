package parse

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/xmbshwll/ariadne/internal/model"
)

// BandcampAlbumURL parses a Bandcamp album URL into the shared parsed representation.
func BandcampAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	return bandcampEntityURL(raw, albumPathSegment, "album", errBandcampNotAlbumURL)
}

// BandcampSongURL parses a Bandcamp track URL into the shared parsed representation.
func BandcampSongURL(raw string) (*model.ParsedAlbumURL, error) {
	return bandcampEntityURL(raw, "track", "song", errBandcampNotSongURL)
}

func bandcampEntityURL(raw string, pathSegment string, entityType string, notEntityErr error) (*model.ParsedAlbumURL, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse bandcamp url: %w", err)
	}

	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		return nil, errMissingBandcampHost
	}
	if !isSupportedBandcampHost(host) {
		return nil, fmt.Errorf("%w: %s", errUnsupportedBandcampHost, parsed.Host)
	}

	segments := pathSegments(parsed.Path)
	if len(segments) < 2 || segments[0] != pathSegment {
		return nil, fmt.Errorf("%w: %s", notEntityErr, raw)
	}

	slug := segments[1]
	canonicalURL := fmt.Sprintf("%s://%s/%s/%s", parsed.Scheme, parsed.Host, pathSegment, slug)
	if parsed.Scheme == "" {
		canonicalURL = fmt.Sprintf("https://%s/%s/%s", parsed.Host, pathSegment, slug)
	}

	return &model.ParsedAlbumURL{
		Service:      model.ServiceBandcamp,
		EntityType:   entityType,
		ID:           slug,
		CanonicalURL: canonicalURL,
		RawURL:       raw,
	}, nil
}

func isSupportedBandcampHost(host string) bool {
	if strings.HasSuffix(host, ".bandcamp.com") || host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
