package youtubemusic

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/xmbshwll/ariadne/internal/adapters/adapterutil"
	"github.com/xmbshwll/ariadne/internal/model"
)

var (
	errUnsupportedYouTubeMusicHost   = errors.New("unsupported youtube music host")
	errMissingYouTubeMusicBrowseID   = errors.New("missing youtube music browse id")
	errMissingYouTubeMusicPlaylistID = errors.New("missing youtube music playlist id")
	errMissingYouTubeMusicVideoID    = errors.New("missing youtube music video id")
	errYouTubeMusicNotAlbumURL       = errors.New("youtube music url is not an album url")
	errYouTubeMusicNotSongURL        = errors.New("youtube music url is not a song url")
)

func ParseAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	parsed, err := parseYouTubeMusicURL(raw)
	if err != nil {
		return nil, err
	}

	segments := adapterutil.PathSegments(parsed.Path)
	switch {
	case len(segments) == 2 && segments[0] == "browse":
		browseID := strings.TrimSpace(segments[1])
		if browseID == "" {
			return nil, errMissingYouTubeMusicBrowseID
		}
		return &model.ParsedAlbumURL{
			Service:      model.ServiceYouTubeMusic,
			EntityType:   model.EntityTypeAlbum,
			ID:           browseID,
			CanonicalURL: "https://music.youtube.com/browse/" + browseID,
			RawURL:       raw,
		}, nil
	case len(segments) == 1 && segments[0] == "playlist":
		playlistID := strings.TrimSpace(parsed.Query().Get("list"))
		if playlistID == "" {
			return nil, errMissingYouTubeMusicPlaylistID
		}
		return &model.ParsedAlbumURL{
			Service:      model.ServiceYouTubeMusic,
			EntityType:   model.EntityTypeAlbum,
			ID:           playlistID,
			CanonicalURL: "https://music.youtube.com/playlist?list=" + playlistID,
			RawURL:       raw,
		}, nil
	default:
		return nil, fmt.Errorf("%w: %s", errYouTubeMusicNotAlbumURL, raw)
	}
}

func ParseSongURL(raw string) (*model.ParsedURL, error) {
	parsed, err := parseYouTubeMusicURL(raw)
	if err != nil {
		return nil, err
	}

	segments := adapterutil.PathSegments(parsed.Path)
	if len(segments) != 1 || segments[0] != "watch" {
		return nil, fmt.Errorf("%w: %s", errYouTubeMusicNotSongURL, raw)
	}
	videoID := strings.TrimSpace(parsed.Query().Get("v"))
	if videoID == "" {
		return nil, errMissingYouTubeMusicVideoID
	}
	return &model.ParsedURL{
		Service:      model.ServiceYouTubeMusic,
		EntityType:   model.EntityTypeSong,
		ID:           videoID,
		CanonicalURL: "https://music.youtube.com/watch?v=" + videoID,
		RawURL:       raw,
	}, nil
}

func parseYouTubeMusicURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse youtube music url: %w", err)
	}

	host := strings.ToLower(parsed.Host)
	if host != "music.youtube.com" {
		return nil, fmt.Errorf("%w: %s", errUnsupportedYouTubeMusicHost, parsed.Host)
	}
	return parsed, nil
}
