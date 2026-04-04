package parse

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/xmbshwll/ariadne/internal/model"
)

// DeezerAlbumURL parses a Deezer album URL into the shared parsed representation.
func DeezerAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse deezer url: %w", err)
	}

	host := strings.ToLower(parsed.Host)
	if host != "www.deezer.com" && host != "deezer.com" {
		return nil, fmt.Errorf("unsupported deezer host: %s", parsed.Host)
	}

	segments := pathSegments(parsed.Path)
	if len(segments) < 2 {
		return nil, fmt.Errorf("invalid deezer album path: %s", parsed.Path)
	}

	regionHint := ""
	index := 0
	if isRegionSegment(segments[0]) {
		regionHint = segments[0]
		index++
	}

	if len(segments[index:]) < 2 || segments[index] != "album" {
		return nil, fmt.Errorf("deezer url is not an album url: %s", raw)
	}

	id := segments[index+1]
	if id == "" {
		return nil, fmt.Errorf("missing deezer album id")
	}

	return &model.ParsedAlbumURL{
		Service:      model.ServiceDeezer,
		EntityType:   "album",
		ID:           id,
		CanonicalURL: "https://www.deezer.com/album/" + id,
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
