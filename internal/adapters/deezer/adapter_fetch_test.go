package deezer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
)

func TestFetchSong(t *testing.T) {
	albumBytes, trackBytes := mustReadDeezerAlbumFixtures(t)
	searchBytes := mustReadDeezerAlbumSearchFixture(t)

	server := newTestServer(t, albumBytes, trackBytes, searchBytes)

	adapter := newTestAdapter(server)

	song, err := adapter.FetchSong(context.Background(), model.ParsedURL{
		Service:      model.ServiceDeezer,
		EntityType:   "song",
		ID:           "116348128",
		CanonicalURL: "https://www.deezer.com/track/116348128",
	})
	require.NoError(t, err)
	assert.Equal(t, deezerComeTogetherISRC, song.ISRC)
	assert.Equal(t, "Abbey Road (Remastered)", song.AlbumTitle)
}
