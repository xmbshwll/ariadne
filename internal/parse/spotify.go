package parse

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/xmbshwll/ariadne/internal/model"
)

// SpotifyAlbumURL parses a Spotify album URL into the shared parsed representation.
func SpotifyAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse spotify url: %w", err)
	}

	host := strings.ToLower(parsed.Host)
	if host != "open.spotify.com" && host != "spotify.com" && host != "www.spotify.com" {
		return nil, fmt.Errorf("unsupported spotify host: %s", parsed.Host)
	}

	segments := pathSegments(parsed.Path)
	if len(segments) < 2 || segments[0] != "album" {
		return nil, fmt.Errorf("spotify url is not an album url: %s", raw)
	}

	id := segments[1]
	if id == "" {
		return nil, fmt.Errorf("missing spotify album id")
	}

	return &model.ParsedAlbumURL{
		Service:      model.ServiceSpotify,
		EntityType:   "album",
		ID:           id,
		CanonicalURL: "https://open.spotify.com/album/" + id,
		RawURL:       raw,
	}, nil
}
