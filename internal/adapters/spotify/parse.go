package spotify

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/adapters/adapterutil"
)

var (
	errUnsupportedSpotifyHost = errors.New("unsupported spotify host")
	errSpotifyNotAlbumURL     = errors.New("spotify url is not an album url")
	errSpotifyNotSongURL      = errors.New("spotify url is not a song url")
	errMissingSpotifyAlbumID  = errors.New("missing spotify album id")
	errMissingSpotifyTrackID  = errors.New("missing spotify track id")
)

func ParseAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	return parseEntityURL(raw, albumPathSegment, "album", errSpotifyNotAlbumURL, errMissingSpotifyAlbumID)
}

func ParseSongURL(raw string) (*model.ParsedURL, error) {
	return parseEntityURL(raw, "track", "song", errSpotifyNotSongURL, errMissingSpotifyTrackID)
}

const albumPathSegment = "album"

func parseEntityURL(raw string, pathSegment string, entityType string, notEntityErr error, missingIDErr error) (*model.ParsedAlbumURL, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse spotify url: %w", err)
	}

	host := strings.ToLower(parsed.Host)
	if host != "open.spotify.com" && host != "spotify.com" && host != "www.spotify.com" {
		return nil, fmt.Errorf("%w: %s", errUnsupportedSpotifyHost, parsed.Host)
	}

	segments := adapterutil.PathSegments(parsed.Path)
	if len(segments) != 2 || segments[0] != pathSegment {
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
