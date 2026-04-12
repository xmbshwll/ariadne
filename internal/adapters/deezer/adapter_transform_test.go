package deezer

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
)

func TestToCanonicalAlbum(t *testing.T) {
	albumBytes := mustReadTestFile(t, "testdata/source-payload.json")
	trackBytes := mustReadTestFile(t, "testdata/tracks.json")

	var album albumResponse
	require.NoError(t, json.Unmarshal(albumBytes, &album))

	var tracks tracksResponse
	require.NoError(t, json.Unmarshal(trackBytes, &tracks))

	adapter := New(nil)
	parsed := model.ParsedAlbumURL{
		Service:      model.ServiceDeezer,
		EntityType:   "album",
		ID:           "12047952",
		CanonicalURL: "https://www.deezer.com/album/12047952",
		RawURL:       "https://www.deezer.com/album/12047952",
	}

	got := adapter.toCanonicalAlbum(parsed, album, tracks)
	assert.Equal(t, "Abbey Road (Remastered)", got.Title)
	assert.Equal(t, "602547670342", got.UPC)
	assert.Equal(t, "EMI Catalogue", got.Label)
	assert.Equal(t, 17, got.TrackCount)
	require.NotEmpty(t, got.Tracks)
	assert.Equal(t, deezerComeTogetherISRC, got.Tracks[0].ISRC)
	assert.Equal(t, 258000, got.Tracks[0].DurationMS)
	assert.Equal(t, "The Beatles", got.Artists[0])
}
