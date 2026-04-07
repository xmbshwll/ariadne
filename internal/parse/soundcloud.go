package parse

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/xmbshwll/ariadne/internal/model"
)

// SoundCloudAlbumURL parses SoundCloud album-like set URLs into the shared parsed representation.
func SoundCloudAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse soundcloud url: %w", err)
	}

	host := strings.ToLower(parsed.Host)
	if host != "soundcloud.com" && host != "www.soundcloud.com" {
		return nil, fmt.Errorf("%w: %s", errUnsupportedSoundCloudHost, parsed.Host)
	}

	segments := pathSegments(parsed.Path)
	if len(segments) < 3 || segments[1] != "sets" {
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
		EntityType:   "set",
		ID:           userSlug + "/sets/" + setSlug,
		CanonicalURL: canonicalURL,
		RawURL:       raw,
	}, nil
}
