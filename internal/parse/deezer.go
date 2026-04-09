package parse

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/xmbshwll/ariadne/internal/model"
)

// DeezerAlbumURL parses a Deezer album URL into the shared parsed representation.
func DeezerAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	return deezerEntityURL(raw, albumPathSegment, "album", errDeezerNotAlbumURL, errMissingDeezerAlbumID)
}

// DeezerSongURL parses a Deezer track URL into the shared parsed representation.
func DeezerSongURL(raw string) (*model.ParsedAlbumURL, error) {
	return deezerEntityURL(raw, "track", "song", errDeezerNotSongURL, errMissingDeezerTrackID)
}

func deezerEntityURL(raw string, pathSegment string, entityType string, notEntityErr error, missingIDErr error) (*model.ParsedAlbumURL, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse deezer url: %w", err)
	}

	host := strings.ToLower(parsed.Host)
	if host != "www.deezer.com" && host != "deezer.com" {
		return nil, fmt.Errorf("%w: %s", errUnsupportedDeezerHost, parsed.Host)
	}

	segments := pathSegments(parsed.Path)
	if len(segments) < 2 {
		return nil, fmt.Errorf("%w: %s", errInvalidDeezerAlbumPath, parsed.Path)
	}

	regionHint := ""
	index := 0
	if isRegionSegment(segments[0]) {
		regionHint = segments[0]
		index++
	}

	if len(segments[index:]) < 2 || segments[index] != pathSegment {
		return nil, fmt.Errorf("%w: %s", notEntityErr, raw)
	}

	id := segments[index+1]
	if id == "" {
		return nil, missingIDErr
	}

	return &model.ParsedAlbumURL{
		Service:      model.ServiceDeezer,
		EntityType:   entityType,
		ID:           id,
		CanonicalURL: "https://www.deezer.com/" + pathSegment + "/" + id,
		RegionHint:   regionHint,
		RawURL:       raw,
	}, nil
}

func pathSegments(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

func isRegionSegment(segment string) bool {
	if len(segment) != 2 {
		return false
	}
	for _, r := range segment {
		if r < 'a' || r > 'z' {
			return false
		}
	}
	return true
}
