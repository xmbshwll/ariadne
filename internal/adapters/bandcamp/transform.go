package bandcamp

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/normalize"
	"github.com/xmbshwll/ariadne/internal/parse"
)

func extractSchema(body []byte) (*schemaAlbum, error) {
	matches := jsonLDPattern.FindSubmatch(body)
	if len(matches) != 2 {
		return nil, errBandcampJSONLDNotFound
	}

	var schema schemaAlbum
	if err := json.Unmarshal(matches[1], &schema); err != nil {
		return nil, errors.Join(errMalformedBandcampJSONLD, fmt.Errorf("unmarshal bandcamp json-ld: %w", err))
	}
	return &schema, nil
}

func toCanonicalAlbum(parsed model.ParsedAlbumURL, album *schemaAlbum) *model.CanonicalAlbum {
	artists := nonEmptyArtistList(album.ByArtist.Name)
	tracks := make([]model.CanonicalTrack, 0, len(album.Track.ItemListElement))
	totalDurationMS := 0
	for _, item := range album.Track.ItemListElement {
		durationMS := parseISODurationMilliseconds(item.Item.Duration)
		totalDurationMS += durationMS
		tracks = append(tracks, model.CanonicalTrack{
			TrackNumber:     item.Position,
			Title:           item.Item.Name,
			NormalizedTitle: normalize.Text(item.Item.Name),
			DurationMS:      durationMS,
			Artists:         artists,
		})
	}

	return &model.CanonicalAlbum{
		Service:           model.ServiceBandcamp,
		SourceID:          parsed.ID,
		SourceURL:         parsed.CanonicalURL,
		RegionHint:        parsed.RegionHint,
		Title:             album.Name,
		NormalizedTitle:   normalize.Text(album.Name),
		Artists:           artists,
		NormalizedArtists: normalize.Artists(artists),
		ReleaseDate:       dateOnly(album.DatePublished),
		Label:             album.Publisher.Name,
		TrackCount:        len(tracks),
		TotalDurationMS:   totalDurationMS,
		ArtworkURL:        schemaImageURL(album.Image),
		EditionHints:      normalize.EditionHints(album.Name),
		Tracks:            tracks,
	}
}

func toCanonicalSong(parsed model.ParsedURL, track *schemaAlbum) *model.CanonicalSong {
	artists := nonEmptyArtistList(track.ByArtist.Name)
	albumArtists := nonEmptyArtistList(track.InAlbum.ByArtist.Name)
	albumID := ""
	if parsedAlbum, err := parse.BandcampAlbumURL(track.InAlbum.ID); err == nil {
		albumID = parsedAlbum.ID
	}
	return &model.CanonicalSong{
		Service:                model.ServiceBandcamp,
		SourceID:               parsed.ID,
		SourceURL:              parsed.CanonicalURL,
		RegionHint:             parsed.RegionHint,
		Title:                  track.Name,
		NormalizedTitle:        normalize.Text(track.Name),
		Artists:                artists,
		NormalizedArtists:      normalize.Artists(artists),
		DurationMS:             parseISODurationMilliseconds(track.Duration),
		AlbumID:                albumID,
		AlbumTitle:             track.InAlbum.Name,
		AlbumNormalizedTitle:   normalize.Text(track.InAlbum.Name),
		AlbumArtists:           albumArtists,
		AlbumNormalizedArtists: normalize.Artists(albumArtists),
		ReleaseDate:            dateOnly(track.DatePublished),
		ArtworkURL:             schemaImageURL(track.Image),
		EditionHints:           normalize.EditionHints(track.Name),
	}
}

func schemaImageURL(value any) string {
	switch image := value.(type) {
	case string:
		return image
	case []any:
		for _, entry := range image {
			if urlValue, ok := entry.(string); ok && urlValue != "" {
				return urlValue
			}
		}
	}
	return ""
}

func parseISODurationMilliseconds(value string) int {
	if value == "" {
		return 0
	}
	value = strings.TrimPrefix(value, "P")
	value = strings.TrimPrefix(value, "T")
	var totalSeconds float64
	for len(value) > 0 {
		index := strings.IndexAny(value, "HMS")
		if index <= 0 {
			break
		}
		number := value[:index]
		unit := value[index]
		value = value[index+1:]

		suffix := ""
		switch unit {
		case 'H':
			suffix = "h"
		case 'M':
			suffix = "m"
		case 'S':
			suffix = "s"
		default:
			continue
		}

		parsed, err := time.ParseDuration(number + suffix)
		if err != nil {
			continue
		}
		totalSeconds += parsed.Seconds()
	}
	return int(totalSeconds * 1000)
}

func dateOnly(value string) string {
	if len(value) < 10 {
		return value
	}
	parsed, err := time.Parse(time.RFC1123, value)
	if err == nil {
		return parsed.Format("2006-01-02")
	}
	parsed, err = time.Parse("02 Jan 2006 15:04:05 MST", value)
	if err == nil {
		return parsed.Format("2006-01-02")
	}
	prefix := value[:10]
	if _, err := time.Parse("2006-01-02", prefix); err == nil {
		return prefix
	}
	return value
}

func nonEmptyArtistList(artist string) []string {
	artist = strings.TrimSpace(artist)
	if artist == "" {
		return nil
	}
	return []string{artist}
}
