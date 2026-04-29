package youtubemusic

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/adapters/adapterutil"
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

func TestParseSongURLAndDeferredFetch(t *testing.T) {
	adapter := New(nil)

	parsed, err := adapter.ParseSongURL("https://music.youtube.com/watch?v=dQw4w9WgXcQ&list=RDAMVMdQw4w9WgXcQ")
	require.NoError(t, err)
	require.NotNil(t, parsed)
	assert.Equal(t, "dQw4w9WgXcQ", parsed.ID)

	_, err = adapter.FetchSong(context.Background(), *parsed)
	assert.ErrorIs(t, err, ErrDeferredRuntimeAdapter)
	assert.ErrorIs(t, err, adapterutil.ErrRuntimeDeferred)
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

func TestExtractTrackTitlesPreservesRepeatedTitles(t *testing.T) {
	body := youTubeMusicTrackTitleBody("Intro", "Interlude", "Intro")
	assert.Equal(t, []string{"Intro", "Interlude", "Intro"}, extractTrackTitles(body))
}

func TestExtractTrackTitlesSkipsImmediateDuplicateParserArtifacts(t *testing.T) {
	body := youTubeMusicTrackTitleBody("Intro", "Intro", "Interlude")
	assert.Equal(t, []string{"Intro", "Interlude"}, extractTrackTitles(body))
}

func TestShouldSkipTrackTitleOnlySkipsCountLabels(t *testing.T) {
	assert.True(t, shouldSkipTrackTitle("1,234 views"))
	assert.True(t, shouldSkipTrackTitle("123 Wiedergaben"))
	assert.False(t, shouldSkipTrackTitle("Views"))
	assert.False(t, shouldSkipTrackTitle("Wiedergaben"))
}

func youTubeMusicTrackTitleBody(titles ...string) []byte {
	parts := make([]string, 0, len(titles))
	for _, title := range titles {
		parts = append(parts, fmt.Sprintf(`musicResponsiveListItemFlexColumnRenderer\x22:\x7b\x22text\x22:\x7b\x22runs\x22:\x5b\x7b\x22text\x22:\x22%s\x22`, title))
	}
	return []byte(strings.Join(parts, " "))
}
