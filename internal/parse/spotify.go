package parse

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/xmbshwll/ariadne/internal/model"
)

// SpotifyAlbumURL parses a Spotify album URL into the shared parsed representation.
func SpotifyAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	return spotifyEntityURL(raw, albumPathSegment, "album", errSpotifyNotAlbumURL, errMissingSpotifyAlbumID)
}

// SpotifySongURL parses a Spotify track URL into the shared parsed representation.
func SpotifySongURL(raw string) (*model.ParsedAlbumURL, error) {
	return spotifyEntityURL(raw, "track", "song", errSpotifyNotSongURL, errMissingSpotifyTrackID)
}

func spotifyEntityURL(raw string, pathSegment string, entityType string, notEntityErr error, missingIDErr error) (*model.ParsedAlbumURL, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse spotify url: %w", err)
	}

	host := strings.ToLower(parsed.Host)
	if host != "open.spotify.com" && host != "spotify.com" && host != "www.spotify.com" {
		return nil, fmt.Errorf("%w: %s", errUnsupportedSpotifyHost, parsed.Host)
	}

	segments := pathSegments(parsed.Path)
	if len(segments) < 2 || segments[0] != pathSegment {
		return nil, fmt.Errorf("%w: %s", notEntityErr, raw)
	}

	id := segments[1]
	if id == "" {
		return nil, missingIDErr
	}

	return &model.ParsedAlbumURL{
		Service:      model.ServiceSpotify,
		EntityType:   entityType,
		ID:           id,
		CanonicalURL: "https://open.spotify.com/" + pathSegment + "/" + id,
		RawURL:       raw,
	}, nil
}
