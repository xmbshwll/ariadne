package applemusic

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/xmbshwll/ariadne/internal/adapters/adapterutil"
	"github.com/xmbshwll/ariadne/internal/model"
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
	parsed, segments, err := parseAppleMusicURL(raw)
	if err != nil {
		return nil, err
	}

	storefront := segments[0]
	trackID := strings.TrimSpace(parsed.Query().Get("i"))
	if trackID == "" {
		return nil, fmt.Errorf("%w: %s", errMissingAppleMusicTrackID, raw)
	}

	return &model.ParsedURL{
		Service:    model.ServiceAppleMusic,
		EntityType: model.EntityTypeSong,
		ID:         trackID,
		CanonicalURL: fmt.Sprintf(
			"https://music.apple.com/%s/%s/%s/%s?i=%s",
			storefront,
			url.PathEscape(albumPathSegment),
			url.PathEscape(segments[2]),
			url.PathEscape(segments[3]),
			url.QueryEscape(trackID),
		),
		RegionHint: storefront,
		RawURL:     raw,
	}, nil
}

const albumPathSegment = "album"

// parseAppleMusicURL validates host, path segments, and returns the parsed URL plus segments.
func parseAppleMusicURL(raw string) (*url.URL, []string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, nil, fmt.Errorf("parse apple music url: %w", err)
	}

	host := strings.ToLower(parsed.Host)
	if host != "music.apple.com" {
		return nil, nil, fmt.Errorf("%w: %s", errUnsupportedAppleMusicHost, parsed.Host)
	}

	segments := adapterutil.PathSegments(parsed.Path)
	if len(segments) != 4 {
		return nil, nil, fmt.Errorf("%w: %s", errInvalidAppleMusicAlbumPath, parsed.Path)
	}
	if !adapterutil.IsRegionSegment(segments[0]) {
		return nil, nil, fmt.Errorf("%w: %s", errInvalidAppleMusicAlbumPath, parsed.Path)
	}
	if segments[1] != albumPathSegment {
		return nil, nil, fmt.Errorf("%w: %s", errAppleMusicNotAlbumURL, raw)
	}

	return parsed, segments, nil
}

func parseAppleMusicAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	_, segments, err := parseAppleMusicURL(raw)
	if err != nil {
		return nil, err
	}

	storefront := segments[0]
	id := segments[3]
	if storefront == "" || id == "" {
		return nil, errMissingAppleMusicStorefrontOrAlbumID
	}

	canonicalURL := fmt.Sprintf(
		"https://music.apple.com/%s/%s/%s/%s",
		storefront,
		url.PathEscape(albumPathSegment),
		url.PathEscape(segments[2]),
		url.PathEscape(id),
	)

	return &model.ParsedAlbumURL{
		Service:      model.ServiceAppleMusic,
		EntityType:   model.EntityTypeAlbum,
		ID:           id,
		CanonicalURL: canonicalURL,
		RegionHint:   storefront,
		RawURL:       raw,
	}, nil
}
