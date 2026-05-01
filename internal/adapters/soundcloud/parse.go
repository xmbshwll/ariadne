package soundcloud

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/parseutil"
)

var (
	errUnsupportedSoundCloudHost        = errors.New("unsupported soundcloud host")
	errSoundCloudNotAlbumURL            = errors.New("soundcloud url is not an album-like set url")
	errSoundCloudNotSongURL             = errors.New("soundcloud url is not a song url")
	errMissingSoundCloudUserOrSetSlug   = errors.New("missing soundcloud user or set slug")
	errMissingSoundCloudUserOrTrackSlug = errors.New("missing soundcloud user or track slug")
)

func ParseAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	parsed, err := parseSoundCloudURL(raw)
	if err != nil {
		return nil, err
	}

	segments := parseutil.PathSegments(parsed.Path)
	if len(segments) != 3 || segments[1] != "sets" {
		return nil, fmt.Errorf("%w: %s", errSoundCloudNotAlbumURL, raw)
	}

	userSlug := segments[0]
	setSlug := segments[2]
	if userSlug == "" || setSlug == "" {
		return nil, errMissingSoundCloudUserOrSetSlug
	}

	canonicalURL := fmt.Sprintf("https://soundcloud.com/%s/sets/%s", userSlug, setSlug)
	return &model.ParsedAlbumURL{
		Service:      model.ServiceSoundCloud,
		EntityType:   "album",
		ID:           userSlug + "/sets/" + setSlug,
		CanonicalURL: canonicalURL,
		RawURL:       raw,
	}, nil
}

func ParseSongURL(raw string) (*model.ParsedURL, error) {
	parsed, err := parseSoundCloudURL(raw)
	if err != nil {
		return nil, err
	}

	segments := parseutil.PathSegments(parsed.Path)
	if len(segments) != 2 {
		return nil, fmt.Errorf("%w: %s", errSoundCloudNotSongURL, raw)
	}

	userSlug := segments[0]
	trackSlug := segments[1]
	if userSlug == "" || trackSlug == "" {
		return nil, errMissingSoundCloudUserOrTrackSlug
	}
	if isSoundCloudNonTrackPath(trackSlug) {
		return nil, fmt.Errorf("%w: %s", errSoundCloudNotSongURL, raw)
	}

	canonicalURL := fmt.Sprintf("https://soundcloud.com/%s/%s", userSlug, trackSlug)
	return &model.ParsedURL{
		Service:      model.ServiceSoundCloud,
		EntityType:   "song",
		ID:           userSlug + "/" + trackSlug,
		CanonicalURL: canonicalURL,
		RawURL:       raw,
	}, nil
}

func isSoundCloudNonTrackPath(segment string) bool {
	switch segment {
	case "sets", "likes", "followers", "following", "posts", "activities", "comments", "tracks":
		return true
	default:
		return false
	}
}

func parseSoundCloudURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse soundcloud url: %w", err)
	}

	host := strings.ToLower(parsed.Host)
	if host != "soundcloud.com" && host != "www.soundcloud.com" {
		return nil, fmt.Errorf("%w: %s", errUnsupportedSoundCloudHost, parsed.Host)
	}
	return parsed, nil
}
