package deezer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
)

func TestFetchSong(t *testing.T) {
	albumBytes := mustReadTestFile(t, "testdata/source-payload.json")
	trackBytes := mustReadTestFile(t, "testdata/tracks.json")
	searchBytes := []byte(`{"data":[{"id":12047952,"title":"Abbey Road (Remastered)"}]}`)

	server := newTestServer(t, albumBytes, trackBytes, searchBytes)
	defer server.Close()

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
