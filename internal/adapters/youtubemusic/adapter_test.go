package youtubemusic

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchAlbum(t *testing.T) {
	sourcePage := mustReadYouTubeMusicSourcePage(t)

	server := newYouTubeMusicTestServer(map[string][]byte{
		youtubeMusicBrowsePath: sourcePage,
	})
	defer server.Close()

	adapter := newYouTubeMusicTestAdapter(server)
	album, err := adapter.FetchAlbum(context.Background(), newYouTubeMusicAlbumSource(server.URL))
	require.NoError(t, err)
	require.NotNil(t, album)
	assert.Equal(t, "Abbey Road (Super Deluxe Edition)", album.Title)
	assert.Equal(t, "https://music.youtube.com/playlist?list=OLAK5uy_lqcFZTOPHGwcnP0nYMzNuY0IES0fl7Fe4", album.SourceURL)
	assert.Equal(t, "OLAK5uy_lqcFZTOPHGwcnP0nYMzNuY0IES0fl7Fe4", album.SourceID)
	assert.Equal(t, []string{"The Beatles"}, album.Artists)
	assert.NotZero(t, album.TrackCount)
	require.NotEmpty(t, album.Tracks)
	assert.Equal(t, "Come Together (2019 Mix)", album.Tracks[0].Title)
	assert.NotEmpty(t, album.ArtworkURL)
}

func TestUnsupportedIdentifierSearches(t *testing.T) {
	adapter := New(nil)

	upcResults, err := adapter.SearchByUPC(context.Background(), "123")
	require.NoError(t, err)
	assert.Empty(t, upcResults)

	isrcResults, err := adapter.SearchByISRC(context.Background(), []string{"ABC"})
	require.NoError(t, err)
	assert.Empty(t, isrcResults)
}
