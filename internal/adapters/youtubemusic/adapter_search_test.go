package youtubemusic

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchByMetadataHydratesBrowseResult(t *testing.T) {
	sourcePage := mustReadYouTubeMusicFixture(t, filepath.Join("testdata", "source-page.html"))
	searchPage := mustReadYouTubeMusicFixture(t, filepath.Join("testdata", "search-page.html"))

	server := newYouTubeMusicTestServer(map[string][]byte{
		youtubeMusicSearchPath: searchPage,
		youtubeMusicBrowsePath: sourcePage,
	})
	defer server.Close()

	adapter := newYouTubeMusicTestAdapter(server)
	results, err := adapter.SearchByMetadata(context.Background(), youTubeMusicAbbeyRoadAlbum())
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "OLAK5uy_lqcFZTOPHGwcnP0nYMzNuY0IES0fl7Fe4", results[0].CandidateID)
	assert.Equal(t, "Abbey Road (Super Deluxe Edition)", results[0].Title)
	assert.NotEmpty(t, results[0].Tracks)
}

func TestSearchByMetadataKeepsEarlierResultsWhenLaterHydrationFails(t *testing.T) {
	sourcePage := mustReadYouTubeMusicFixture(t, filepath.Join("testdata", "source-page.html"))
	searchPage := youTubeMusicSearchPage(
		youTubeMusicAlbumSearchResult(youtubeMusicAbbeyRoadTitle, "GOOD", youtubeMusicAbbeyRoadArtist),
		youTubeMusicAlbumSearchResult("Broken Album", "BROKEN", youtubeMusicAbbeyRoadArtist),
	)

	server := newYouTubeMusicTestServer(map[string][]byte{
		youtubeMusicSearchPath: searchPage,
		"/browse/GOOD":         sourcePage,
		"/browse/BROKEN":       []byte(youtubeMusicBrokenPageHTML),
	})
	defer server.Close()

	adapter := newYouTubeMusicTestAdapter(server)
	results, err := adapter.SearchByMetadata(context.Background(), youTubeMusicAbbeyRoadAlbum())
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "OLAK5uy_lqcFZTOPHGwcnP0nYMzNuY0IES0fl7Fe4", results[0].CandidateID)
}

func TestSearchByMetadataReturnsMalformedPageErrorWhenNothingRecovers(t *testing.T) {
	searchPage := mustReadYouTubeMusicFixture(t, filepath.Join("testdata", "search-page.html"))

	server := newYouTubeMusicTestServer(map[string][]byte{
		youtubeMusicSearchPath: searchPage,
		youtubeMusicBrowsePath: []byte(youtubeMusicBrokenPageHTML),
	})
	defer server.Close()

	adapter := newYouTubeMusicTestAdapter(server)
	_, err := adapter.SearchByMetadata(context.Background(), youTubeMusicAbbeyRoadAlbum())
	require.Error(t, err)
	assert.ErrorIs(t, err, errMalformedYouTubeMusicPage)
}
