package deezer

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/parseutil"
)

var (
	errUnsupportedDeezerHost = errors.New("unsupported deezer host")
	errInvalidDeezerPath     = errors.New("invalid deezer path")
	errDeezerNotAlbumURL     = errors.New("deezer url is not an album url")
	errDeezerNotSongURL      = errors.New("deezer url is not a song url")
	errMissingDeezerAlbumID  = errors.New("missing deezer album id")
	errMissingDeezerTrackID  = errors.New("missing deezer track id")
)

func ParseAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	return parseEntityURL(raw, albumPathSegment, "album", errDeezerNotAlbumURL, errMissingDeezerAlbumID)
}

func ParseSongURL(raw string) (*model.ParsedURL, error) {
	return parseEntityURL(raw, "track", "song", errDeezerNotSongURL, errMissingDeezerTrackID)
}

const albumPathSegment = "album"

func parseEntityURL(raw string, pathSegment string, entityType string, notEntityErr error, missingIDErr error) (*model.ParsedAlbumURL, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse deezer url: %w", err)
	}

	host := strings.ToLower(parsed.Host)
	if host != "www.deezer.com" && host != "deezer.com" {
		return nil, fmt.Errorf("%w: %s", errUnsupportedDeezerHost, parsed.Host)
	}

	segments := parseutil.PathSegments(parsed.Path)
	if len(segments) == 0 {
		return nil, fmt.Errorf("%w: %s", errInvalidDeezerPath, parsed.Path)
	}

	regionHint := ""
	index := 0
	if parseutil.IsRegionSegment(segments[0]) {
		regionHint = segments[0]
		index++
	}
	if len(segments) <= index {
		return nil, fmt.Errorf("%w: %s", errInvalidDeezerPath, parsed.Path)
	}
	if segments[index] != pathSegment {
		return nil, fmt.Errorf("%w: %s", notEntityErr, raw)
	}
	if len(segments) == index+1 {
		return nil, missingIDErr
	}

	id := segments[index+1]
	if id == "" {
		return nil, missingIDErr
	}
	if len(segments) != index+2 {
		return nil, fmt.Errorf("%w: %s", errInvalidDeezerPath, parsed.Path)
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
