package parse

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/xmbshwll/ariadne/internal/model"
)

// TIDALAlbumURL parses a TIDAL album URL into the shared parsed representation.
func TIDALAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
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
		return nil, fmt.Errorf("%w: %s", errInvalidTIDALAlbumPath, parsed.Path)
	}

	index := 0
	if segments[0] == "browse" {
		index++
	}
	if len(segments[index:]) < 2 || segments[index] != albumPathSegment {
		return nil, fmt.Errorf("%w: %s", errTIDALNotAlbumURL, raw)
	}

	id := segments[index+1]
	if id == "" {
		return nil, errMissingTIDALAlbumID
	}

	return &model.ParsedAlbumURL{
		Service:      model.ServiceTIDAL,
		EntityType:   "album",
		ID:           id,
		CanonicalURL: "https://tidal.com/" + albumPathSegment + "/" + id,
		RawURL:       raw,
	}, nil
}
