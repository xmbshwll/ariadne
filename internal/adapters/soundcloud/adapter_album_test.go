package soundcloud

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
)

func TestFetchAlbumFromHydration(t *testing.T) {
	fixture := newTestFixture(t)

	album, err := fixture.adapter.FetchAlbum(context.Background(), model.ParsedAlbumURL{
		Service:      model.ServiceSoundCloud,
		EntityType:   "album",
		ID:           "evidence-official/sets/cats-dogs-6",
		CanonicalURL: fixture.server.URL + "/album",
		RawURL:       fixture.server.URL + "/album",
	})
	require.NoError(t, err)
	assert.Equal(t, soundCloudCatsAndDogs, album.Title)
	assert.Equal(t, fixture.server.URL+"/album", album.SourceURL)
	assert.Equal(t, 17, album.TrackCount)
	assert.Equal(t, "826257014467", album.UPC)
	require.NotEmpty(t, album.Tracks)
	assert.Equal(t, soundCloudTrackISRC, album.Tracks[0].ISRC)
	assert.Equal(t, "Rhymesayers", album.Label)
}

func TestExtractPlaylistHydrationRequiresExactURLMatch(t *testing.T) {
	fixture := newTestFixture(t)

	body := fmt.Appendf(
		nil,
		`<html><body><script>window.__sc_hydration = [{"hydratable":"playlist","data":%s}];</script></body></html>`,
		fixture.sourcePayload,
	)
	playlist, err := extractPlaylistHydration(body, fixture.server.URL+"/missing-album")
	require.Error(t, err)
	assert.Nil(t, playlist)
	assert.ErrorIs(t, err, errSoundCloudPlaylistNotFound)
}

func TestSearchAlbumByMetadata(t *testing.T) {
	fixture := newTestFixture(t)

	results, err := fixture.adapter.SearchByMetadata(context.Background(), model.CanonicalAlbum{
		Title:   soundCloudCatsAndDogs,
		Artists: []string{"Evidence"},
	})
	require.NoError(t, err)
	require.Len(t, results, 5)
	assert.Equal(t, "evidence-official/sets/cats-dogs-3", results[0].CandidateID)
	assert.Equal(t, "evidence-official/sets/cats-dogs-6", results[1].CandidateID)
	assert.Equal(t, soundCloudTrackISRC, results[1].Tracks[0].ISRC)
}
