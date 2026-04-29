package amazonmusic

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
)

func TestAdapter(t *testing.T) {
	adapter := New(nil)

	parsed, err := adapter.ParseAlbumURL("https://music.amazon.com/albums/B0064UPU4G")
	require.NoError(t, err)
	require.NotNil(t, parsed)
	assert.Equal(t, "B0064UPU4G", parsed.ID)

	_, err = adapter.FetchAlbum(context.Background(), model.ParsedAlbumURL{
		Service:      model.ServiceAmazonMusic,
		EntityType:   "album",
		ID:           "B0064UPU4G",
		CanonicalURL: "https://music.amazon.com/albums/B0064UPU4G",
	})
	require.ErrorIs(t, err, ErrDeferredRuntimeAdapter)

	song, err := adapter.ParseSongURL("https://music.amazon.com/albums/B0064UPU4G?trackAsin=B0064TRACK")
	require.NoError(t, err)
	require.NotNil(t, song)
	assert.Equal(t, "B0064TRACK", song.ID)

	_, err = adapter.FetchSong(context.Background(), *song)
	require.ErrorIs(t, err, ErrDeferredRuntimeAdapter)

	upcResults, err := adapter.SearchByUPC(context.Background(), "123")
	require.NoError(t, err)
	assert.Empty(t, upcResults)
}
