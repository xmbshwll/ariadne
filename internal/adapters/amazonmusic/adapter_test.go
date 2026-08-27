package amazonmusic_test

import (
	"context"
	"testing"

	amazonmusic "github.com/xmbshwll/ariadne/internal/adapters/amazonmusic"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xmbshwll/ariadne/internal/adapters/adapterutil"
	"github.com/xmbshwll/ariadne/internal/resolve"
)

func TestAdapter(t *testing.T) {
	adapter := amazonmusic.New(nil)

	parsed, err := adapter.ParseAlbumURL("https://music.amazon.com/albums/B0064UPU4G")
	require.NoError(t, err)
	require.NotNil(t, parsed)
	assert.Equal(t, "B0064UPU4G", parsed.ID)

	_, err = adapter.FetchAlbum(context.Background(), *parsed)
	require.ErrorIs(t, err, amazonmusic.ErrDeferredRuntimeAdapter)
	assert.ErrorIs(t, err, adapterutil.ErrRuntimeDeferred)

	song, err := adapter.ParseSongURL("https://music.amazon.com/albums/B0064UPU4G?trackAsin=B0064TRACK")
	require.NoError(t, err)
	require.NotNil(t, song)
	assert.Equal(t, "B0064TRACK", song.ID)

	_, err = adapter.FetchSong(context.Background(), *song)
	require.ErrorIs(t, err, amazonmusic.ErrDeferredRuntimeAdapter)
	assert.ErrorIs(t, err, adapterutil.ErrRuntimeDeferred)

	assert.NotImplements(t, (*resolve.UPCSearcher)(nil), adapter)
}
