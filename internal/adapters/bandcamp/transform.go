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
			Artists:         []string{album.ByArtist.Name},
		})
	}

	artists := []string{album.ByArtist.Name}
	return &model.CanonicalAlbum{
		Service:           model.ServiceBandcamp,
		SourceID:          parsed.ID,
		SourceURL:         parsed.CanonicalURL,
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
	albumID := ""
	if parsedAlbum, err := parse.BandcampAlbumURL(track.InAlbum.ID); err == nil {
		albumID = parsedAlbum.ID
	}
	return &model.CanonicalSong{
		Service:                model.ServiceBandcamp,
		SourceID:               parsed.ID,
		SourceURL:              parsed.CanonicalURL,
		Title:                  track.Name,
		NormalizedTitle:        normalize.Text(track.Name),
		Artists:                artists,
		NormalizedArtists:      normalize.Artists(artists),
		DurationMS:             parseISODurationMilliseconds(track.Duration),
		AlbumID:                albumID,
		AlbumTitle:             track.InAlbum.Name,
		AlbumNormalizedTitle:   normalize.Text(track.InAlbum.Name),
		AlbumArtists:           artists,
		AlbumNormalizedArtists: normalize.Artists(artists),
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
	var hours, minutes int
	var seconds float64
	for len(value) > 0 {
		index := strings.IndexAny(value, "HMS")
		if index <= 0 {
			break
		}
		number := value[:index]
		unit := value[index]
		value = value[index+1:]
		switch unit {
		case 'H':
			parsed, _ := time.ParseDuration(number + "h")
			hours = int(parsed.Hours())
		case 'M':
			parsed, _ := time.ParseDuration(number + "m")
			minutes = int(parsed.Minutes()) % 60
		case 'S':
			parsed, err := time.ParseDuration(number + "s")
			if err == nil {
				seconds = parsed.Seconds()
			}
		}
	}
	return int((float64(hours*3600+minutes*60) + seconds) * 1000)
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
	return value[:10]
}

func nonEmptyArtistList(artist string) []string {
	artist = strings.TrimSpace(artist)
	if artist == "" {
		return nil
	}
	return []string{artist}
}
