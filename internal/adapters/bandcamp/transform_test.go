package bandcamp_test

import (
	"testing"

	bandcamp "github.com/xmbshwll/ariadne/internal/adapters/bandcamp"
	"github.com/xmbshwll/ariadne/internal/adapters/canonical"

	"github.com/stretchr/testify/assert"

	"github.com/xmbshwll/ariadne/internal/model"
)

func TestParseISODurationMillisecondsAccumulatesTotalMinutes(t *testing.T) {
	assert.Equal(t, 5400000, canonical.ISODurationMilliseconds("PT90M"))
	assert.Equal(t, 9000000, canonical.ISODurationMilliseconds("PT1H90M"))
}

func TestParseISODurationMillisecondsEdgeCases(t *testing.T) {
	assert.Equal(t, 0, canonical.ISODurationMilliseconds(""))
	assert.Equal(t, 1500, canonical.ISODurationMilliseconds("PT1.5S"))
	assert.Equal(t, 3723000, canonical.ISODurationMilliseconds("PT1H2M3S"))
	assert.Equal(t, 0, canonical.ISODurationMilliseconds("invalid"))
}

func TestToCanonicalAlbumSkipsEmptyArtist(t *testing.T) {
	album := bandcamp.ToCanonicalAlbum(model.ParsedAlbumURL{ID: "album-id", CanonicalURL: "https://example.com/album-id", RegionHint: "gb"}, &bandcamp.SchemaAlbum{
		Name:      "Example Album",
		ByArtist:  bandcamp.SchemaMusicGroup{Name: ""},
		Publisher: bandcamp.SchemaMusicGroup{Name: "Example Label"},
		Track: bandcamp.SchemaTrackList{ItemListElement: []bandcamp.SchemaTrackItem{{
			Position: 1,
			Item: bandcamp.SchemaMusicRecording{
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
	song := bandcamp.ToCanonicalSong(model.ParsedURL{ID: "track-id", CanonicalURL: "https://example.com/track-id", RegionHint: "us"}, &bandcamp.SchemaAlbum{
		Name:          "Track Name",
		ByArtist:      bandcamp.SchemaMusicGroup{Name: "Track Artist"},
		DatePublished: "27 Sep 2019 00:00:00 GMT",
		Duration:      "PT1M",
		InAlbum: bandcamp.SchemaAlbumRelation{
			ID:       "https://example.bandcamp.com/album/example-album",
			Name:     "Compilation Album",
			ByArtist: bandcamp.SchemaMusicGroup{Name: "Various Artists"},
		},
	})

	assert.Equal(t, "us", song.RegionHint)
	assert.Equal(t, []string{"Various Artists"}, song.AlbumArtists)
	assert.Equal(t, []string{"various artists"}, song.AlbumNormalizedArtists)
}

func TestToCanonicalSongLeavesAlbumArtistsEmptyWithoutAlbumArtist(t *testing.T) {
	song := bandcamp.ToCanonicalSong(model.ParsedURL{ID: "track-id", CanonicalURL: "https://example.com/track-id"}, &bandcamp.SchemaAlbum{
		Name:     "Track Name",
		ByArtist: bandcamp.SchemaMusicGroup{Name: "Track Artist"},
		Duration: "PT1M",
		InAlbum:  bandcamp.SchemaAlbumRelation{ID: "https://example.bandcamp.com/album/example-album", Name: "Album"},
	})

	assert.Empty(t, song.AlbumArtists)
	assert.Empty(t, song.AlbumNormalizedArtists)
}
