package youtubemusic_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	youtubemusic "github.com/xmbshwll/ariadne/internal/adapters/youtubemusic"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xmbshwll/ariadne/internal/adapters"
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
	adapter := youtubemusic.New(nil)

	parsed, err := adapter.ParseSongURL("https://music.youtube.com/watch?v=dQw4w9WgXcQ&list=RDAMVMdQw4w9WgXcQ")
	require.NoError(t, err)
	require.NotNil(t, parsed)
	assert.Equal(t, "dQw4w9WgXcQ", parsed.ID)

	_, err = adapter.FetchSong(context.Background(), *parsed)
	assert.ErrorIs(t, err, youtubemusic.ErrDeferredRuntimeAdapter)
	assert.ErrorIs(t, err, adapters.ErrRuntimeDeferred)
}

func TestUnsupportedIdentifierSearches(t *testing.T) {
	adapter := youtubemusic.New(nil)

	_, err := adapter.SearchAlbumByUPC(context.Background(), "00602537184945")
	assert.ErrorIs(t, err, adapters.ErrUnsupported)
	_, err = adapter.SearchAlbumByISRC(context.Background(), []string{"GBAYE0601690"})
	assert.ErrorIs(t, err, adapters.ErrUnsupported)
}

func TestExtractTrackTitlesPreservesRepeatedTitles(t *testing.T) {
	body := youTubeMusicTrackTitleBody("Intro", "Interlude", "Intro")
	assert.Equal(t, []string{"Intro", "Interlude", "Intro"}, youtubemusic.ExtractTrackTitles(body))
}

func TestExtractTrackTitlesSkipsImmediateDuplicateParserArtifacts(t *testing.T) {
	body := youTubeMusicTrackTitleBody("Intro", "Intro", "Interlude")
	assert.Equal(t, []string{"Intro", "Interlude"}, youtubemusic.ExtractTrackTitles(body))
}

func TestShouldSkipTrackTitleOnlySkipsCountLabels(t *testing.T) {
	assert.True(t, youtubemusic.ShouldSkipTrackTitle("1,234 views"))
	assert.True(t, youtubemusic.ShouldSkipTrackTitle("123 Wiedergaben"))
	assert.False(t, youtubemusic.ShouldSkipTrackTitle("Views"))
	assert.False(t, youtubemusic.ShouldSkipTrackTitle("Wiedergaben"))
}

func youTubeMusicTrackTitleBody(titles ...string) []byte {
	parts := make([]string, 0, len(titles))
	for _, title := range titles {
		parts = append(parts, fmt.Sprintf(`musicResponsiveListItemFlexColumnRenderer\x22:\x7b\x22text\x22:\x7b\x22runs\x22:\x5b\x7b\x22text\x22:\x22%s\x22`, title))
	}
	return []byte(strings.Join(parts, " "))
}
