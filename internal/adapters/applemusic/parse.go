package applemusic

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/adapters/adapterutil"
)

var (
	errUnsupportedAppleMusicHost            = errors.New("unsupported apple music host")
	errInvalidAppleMusicAlbumPath           = errors.New("invalid apple music album path")
	errAppleMusicNotAlbumURL                = errors.New("apple music url is not an album url")
	errMissingAppleMusicStorefrontOrAlbumID = errors.New("missing storefront or album id")
	errMissingAppleMusicTrackID             = errors.New("missing apple music track id")
)

func ParseAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	return parseAppleMusicAlbumURL(raw)
}

func ParseSongURL(raw string) (*model.ParsedURL, error) {
	parsed, err := parseAppleMusicAlbumURL(raw)
	if err != nil {
		return nil, err
	}
	trackID := strings.TrimSpace(parsedQuery(raw, "i"))
	if trackID == "" {
		return nil, fmt.Errorf("%w: %s", errMissingAppleMusicTrackID, raw)
	}
	parsed.EntityType = "song"
	parsed.ID = trackID
	parsed.CanonicalURL = parsed.CanonicalURL + "?i=" + url.QueryEscape(trackID)
	return parsed, nil
}

func parseAppleMusicAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse apple music url: %w", err)
	}

	host := strings.ToLower(parsed.Host)
	if host != "music.apple.com" {
		return nil, fmt.Errorf("%w: %s", errUnsupportedAppleMusicHost, parsed.Host)
	}

	segments := adapterutil.PathSegments(parsed.Path)
	if len(segments) != 4 {
		return nil, fmt.Errorf("%w: %s", errInvalidAppleMusicAlbumPath, parsed.Path)
	}
	if segments[1] != albumPathSegment {
		return nil, fmt.Errorf("%w: %s", errAppleMusicNotAlbumURL, raw)
	}

	storefront := segments[0]
	id := segments[3]
	if storefront == "" || id == "" {
		return nil, errMissingAppleMusicStorefrontOrAlbumID
	}

	canonicalURL := fmt.Sprintf("https://music.apple.com/%s/%s/%s/%s", storefront, albumPathSegment, segments[2], id)

	return &model.ParsedAlbumURL{
		Service:      model.ServiceAppleMusic,
		EntityType:   "album",
		ID:           id,
		CanonicalURL: canonicalURL,
		RegionHint:   storefront,
		RawURL:       raw,
	}, nil
}

func parsedQuery(raw string, key string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return parsed.Query().Get(key)
}

const albumPathSegment = "album"
