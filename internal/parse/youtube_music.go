package parse

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/xmbshwll/ariadne/internal/model"
)

// YouTubeMusicAlbumURL parses YouTube Music album-like URLs into the shared parsed representation.
func YouTubeMusicAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse youtube music url: %w", err)
	}

	host := strings.ToLower(parsed.Host)
	if host != "music.youtube.com" {
		return nil, fmt.Errorf("unsupported youtube music host: %s", parsed.Host)
	}

	segments := pathSegments(parsed.Path)
	switch {
	case len(segments) == 2 && segments[0] == "browse":
		browseID := strings.TrimSpace(segments[1])
		if browseID == "" {
			return nil, fmt.Errorf("missing youtube music browse id")
		}
		return &model.ParsedAlbumURL{
			Service:      model.ServiceYouTubeMusic,
			EntityType:   "album",
			ID:           browseID,
			CanonicalURL: "https://music.youtube.com/browse/" + browseID,
			RawURL:       raw,
		}, nil
	case len(segments) == 1 && segments[0] == "playlist":
		playlistID := strings.TrimSpace(parsed.Query().Get("list"))
		if playlistID == "" {
			return nil, fmt.Errorf("missing youtube music playlist id")
		}
		return &model.ParsedAlbumURL{
			Service:      model.ServiceYouTubeMusic,
			EntityType:   "album",
			ID:           playlistID,
			CanonicalURL: "https://music.youtube.com/playlist?list=" + playlistID,
			RawURL:       raw,
		}, nil
	default:
		return nil, fmt.Errorf("youtube music url is not an album url: %s", raw)
	}
}
