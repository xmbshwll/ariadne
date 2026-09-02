package deezer_test

import (
	"testing"

	deezer "github.com/xmbshwll/ariadne/internal/adapters/deezer"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xmbshwll/ariadne/internal/model"
)

func TestAdapterIdentityAndParsing(t *testing.T) {
	adapter := deezer.New(nil)

	assert.Equal(t, model.ServiceDeezer, adapter.Service())

	album, err := adapter.ParseAlbumURL("https://www.deezer.com/album/12047952")
	require.NoError(t, err)
	assert.Equal(t, model.ServiceDeezer, album.Service)
	assert.Equal(t, "12047952", album.ID)

	song, err := adapter.ParseSongURL("https://www.deezer.com/track/116348128")
	require.NoError(t, err)
	assert.Equal(t, model.ServiceDeezer, song.Service)
	assert.Equal(t, "116348128", song.ID)
}
