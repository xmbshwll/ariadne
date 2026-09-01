package bandcamp

import (
	"time"

	"github.com/xmbshwll/ariadne/internal/adapters/canonical"
	"github.com/xmbshwll/ariadne/internal/htmlx"
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/normalize"
)

func extractSchema(body []byte) (*SchemaAlbum, error) {
	schema, err := htmlx.DecodeJSONBlock[SchemaAlbum](body, jsonLDPattern, errBandcampJSONLDNotFound, "unmarshal bandcamp json-ld", ErrMalformedBandcampJSONLD)
	if err != nil {
		return nil, err
	}
	return &schema, nil
}

func ToCanonicalAlbum(parsed model.ParsedAlbumURL, album *SchemaAlbum) *model.CanonicalAlbum {
	artists := canonical.SingleArtistList(album.ByArtist.Name)
	tracks := make([]model.CanonicalTrack, 0, len(album.Track.ItemListElement))
	totalDurationMS := 0
	for _, item := range album.Track.ItemListElement {
		durationMS := canonical.ISODurationMilliseconds(item.Item.Duration)
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
		ReleaseDate:       displayDateOnly(album.DatePublished),
		Label:             album.Publisher.Name,
		TrackCount:        len(tracks),
		TotalDurationMS:   totalDurationMS,
		ArtworkURL:        schemaImageURL(album.Image),
		EditionHints:      normalize.EditionHints(album.Name),
		Tracks:            tracks,
	}
}

func ToCanonicalSong(parsed model.ParsedURL, track *SchemaAlbum) *model.CanonicalSong {
	artists := canonical.SingleArtistList(track.ByArtist.Name)
	albumArtists := canonical.SingleArtistList(track.InAlbum.ByArtist.Name)
	albumID := ""
	if parsedAlbum, err := ParseAlbumURL(track.InAlbum.ID); err == nil {
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
		DurationMS:             canonical.ISODurationMilliseconds(track.Duration),
		AlbumID:                albumID,
		AlbumTitle:             track.InAlbum.Name,
		AlbumNormalizedTitle:   normalize.Text(track.InAlbum.Name),
		AlbumArtists:           albumArtists,
		AlbumNormalizedArtists: normalize.Artists(albumArtists),
		ReleaseDate:            displayDateOnly(track.DatePublished),
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

func displayDateOnly(value string) string {
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
