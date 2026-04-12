package bandcamp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xmbshwll/ariadne/internal/model"
)

func TestParseISODurationMillisecondsAccumulatesTotalMinutes(t *testing.T) {
	assert.Equal(t, 5400000, parseISODurationMilliseconds("PT90M"))
	assert.Equal(t, 9000000, parseISODurationMilliseconds("PT1H90M"))
}

func TestParseISODurationMillisecondsEdgeCases(t *testing.T) {
	assert.Equal(t, 0, parseISODurationMilliseconds(""))
	assert.Equal(t, 1500, parseISODurationMilliseconds("PT1.5S"))
	assert.Equal(t, 3723000, parseISODurationMilliseconds("PT1H2M3S"))
	assert.Equal(t, 0, parseISODurationMilliseconds("invalid"))
}

func TestToCanonicalAlbumSkipsEmptyArtist(t *testing.T) {
	album := toCanonicalAlbum(model.ParsedAlbumURL{ID: "album-id", CanonicalURL: "https://example.com/album-id", RegionHint: "gb"}, &schemaAlbum{
		Name:      "Example Album",
		ByArtist:  schemaMusicGroup{Name: ""},
		Publisher: schemaMusicGroup{Name: "Example Label"},
		Track: schemaTrackList{ItemListElement: []schemaTrackItem{{
			Position: 1,
			Item: schemaMusicRecording{
				Name:     "Intro",
				Duration: "PT1M",
			},
		}}},
	})

	assert.Equal(t, "gb", album.RegionHint)
	assert.Empty(t, album.Artists)
	assert.Empty(t, album.NormalizedArtists)
	assert.Empty(t, album.Tracks[0].Artists)
}

func TestToCanonicalSongUsesAlbumArtistAndRegionHint(t *testing.T) {
	song := toCanonicalSong(model.ParsedURL{ID: "track-id", CanonicalURL: "https://example.com/track-id", RegionHint: "us"}, &schemaAlbum{
		Name:          "Track Name",
		ByArtist:      schemaMusicGroup{Name: "Track Artist"},
		DatePublished: "27 Sep 2019 00:00:00 GMT",
		Duration:      "PT1M",
		InAlbum: schemaAlbumRelation{
			ID:       "https://example.bandcamp.com/album/example-album",
			Name:     "Compilation Album",
			ByArtist: schemaMusicGroup{Name: "Various Artists"},
		},
	})

	assert.Equal(t, "us", song.RegionHint)
	assert.Equal(t, []string{"Various Artists"}, song.AlbumArtists)
	assert.Equal(t, []string{"various artists"}, song.AlbumNormalizedArtists)
}

func TestToCanonicalSongLeavesAlbumArtistsEmptyWithoutAlbumArtist(t *testing.T) {
	song := toCanonicalSong(model.ParsedURL{ID: "track-id", CanonicalURL: "https://example.com/track-id"}, &schemaAlbum{
		Name:     "Track Name",
		ByArtist: schemaMusicGroup{Name: "Track Artist"},
		Duration: "PT1M",
		InAlbum:  schemaAlbumRelation{ID: "https://example.bandcamp.com/album/example-album", Name: "Album"},
	})

	assert.Empty(t, song.AlbumArtists)
	assert.Empty(t, song.AlbumNormalizedArtists)
}
