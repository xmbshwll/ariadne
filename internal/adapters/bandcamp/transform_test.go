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

func TestToCanonicalAlbumSkipsEmptyArtist(t *testing.T) {
	album := toCanonicalAlbum(model.ParsedAlbumURL{ID: "album-id", CanonicalURL: "https://example.com/album-id"}, &schemaAlbum{
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

	assert.Empty(t, album.Artists)
	assert.Empty(t, album.NormalizedArtists)
	assert.Empty(t, album.Tracks[0].Artists)
}
