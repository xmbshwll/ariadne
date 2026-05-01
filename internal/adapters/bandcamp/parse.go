package bandcamp

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/parseutil"
)

var (
	errMissingBandcampHost     = errors.New("missing bandcamp host")
	errUnsupportedBandcampHost = errors.New("unsupported bandcamp host")
	errBandcampNotAlbumURL     = errors.New("bandcamp url is not an album url")
	errBandcampNotSongURL      = errors.New("bandcamp url is not a song url")
)

func ParseAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	return parseEntityURL(raw, albumPathSegment, "album", errBandcampNotAlbumURL)
}

func ParseSongURL(raw string) (*model.ParsedURL, error) {
	return parseEntityURL(raw, "track", "song", errBandcampNotSongURL)
}

const albumPathSegment = "album"

func parseEntityURL(raw string, pathSegment string, entityType string, notEntityErr error) (*model.ParsedAlbumURL, error) {
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

	segments := parseutil.PathSegments(parsed.Path)
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
