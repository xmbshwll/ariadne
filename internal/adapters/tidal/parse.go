package tidal

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/adapters/adapterutil"
)

var (
	errUnsupportedTIDALHost = errors.New("unsupported tidal host")
	errInvalidTIDALPath     = errors.New("invalid tidal path")
	errTIDALNotAlbumURL     = errors.New("tidal url is not an album url")
	errTIDALNotSongURL      = errors.New("tidal url is not a song url")
	errMissingTIDALAlbumID  = errors.New("missing tidal album id")
	errMissingTIDALTrackID  = errors.New("missing tidal track id")
)

func ParseAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	return parseEntityURL(raw, albumPathSegment, "album", errTIDALNotAlbumURL, errMissingTIDALAlbumID)
}

func ParseSongURL(raw string) (*model.ParsedURL, error) {
	return parseEntityURL(raw, "track", "song", errTIDALNotSongURL, errMissingTIDALTrackID)
}

const albumPathSegment = "album"

func parseEntityURL(raw string, pathSegment string, entityType string, notEntityErr error, missingIDErr error) (*model.ParsedAlbumURL, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse tidal url: %w", err)
	}

	host := strings.ToLower(parsed.Host)
	if host != "tidal.com" && host != "www.tidal.com" && host != "listen.tidal.com" {
		return nil, fmt.Errorf("%w: %s", errUnsupportedTIDALHost, parsed.Host)
	}

	segments := adapterutil.PathSegments(parsed.Path)
	if len(segments) < 2 {
		return nil, fmt.Errorf("%w: %s", errInvalidTIDALPath, parsed.Path)
	}

	index := 0
	if segments[0] == "browse" {
		index++
	}
	if len(segments[index:]) != 2 || segments[index] != pathSegment {
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
