package parse

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/xmbshwll/ariadne/internal/model"
)

// TIDALAlbumURL parses a TIDAL album URL into the shared parsed representation.
func TIDALAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	return tidalEntityURL(raw, albumPathSegment, "album", errTIDALNotAlbumURL, errMissingTIDALAlbumID)
}

// TIDALSongURL parses a TIDAL track URL into the shared parsed representation.
func TIDALSongURL(raw string) (*model.ParsedAlbumURL, error) {
	return tidalEntityURL(raw, "track", "song", errTIDALNotSongURL, errMissingTIDALTrackID)
}

func tidalEntityURL(raw string, pathSegment string, entityType string, notEntityErr error, missingIDErr error) (*model.ParsedAlbumURL, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse tidal url: %w", err)
	}

	host := strings.ToLower(parsed.Host)
	if host != "tidal.com" && host != "www.tidal.com" && host != "listen.tidal.com" {
		return nil, fmt.Errorf("%w: %s", errUnsupportedTIDALHost, parsed.Host)
	}

	segments := pathSegments(parsed.Path)
	if len(segments) < 2 {
		return nil, fmt.Errorf("%w: %s", errInvalidTIDALPath, parsed.Path)
	}

	index := 0
	if segments[0] == "browse" {
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
		Service:      model.ServiceTIDAL,
		EntityType:   entityType,
		ID:           id,
		CanonicalURL: "https://tidal.com/" + pathSegment + "/" + id,
		RawURL:       raw,
	}, nil
}
