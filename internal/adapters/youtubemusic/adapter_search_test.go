package youtubemusic

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchByMetadataHydratesBrowseResult(t *testing.T) {
	sourcePage := mustReadYouTubeMusicSourcePage(t)
	searchPage := mustReadYouTubeMusicSearchPage(t)

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
	sourcePage := mustReadYouTubeMusicSourcePage(t)
	searchPage := youTubeMusicAlbumSearchPage(
		youTubeMusicSearchResult{Title: youtubeMusicAbbeyRoadTitle, BrowseID: "GOOD", Artist: youtubeMusicAbbeyRoadArtist},
		youTubeMusicSearchResult{Title: "Broken Album", BrowseID: "BROKEN", Artist: youtubeMusicAbbeyRoadArtist},
	)

	server := newYouTubeMusicTestServer(map[string][]byte{
		youtubeMusicSearchPath: searchPage,
		"/browse/GOOD":         sourcePage,
		"/browse/BROKEN":       youTubeMusicBrokenBrowsePage(),
	})
	defer server.Close()

	adapter := newYouTubeMusicTestAdapter(server)
	results, err := adapter.SearchByMetadata(context.Background(), youTubeMusicAbbeyRoadAlbum())
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "OLAK5uy_lqcFZTOPHGwcnP0nYMzNuY0IES0fl7Fe4", results[0].CandidateID)
}

func TestSearchByMetadataReturnsMalformedPageErrorWhenNothingRecovers(t *testing.T) {
	searchPage := mustReadYouTubeMusicSearchPage(t)

	server := newYouTubeMusicTestServer(map[string][]byte{
		youtubeMusicSearchPath: searchPage,
		youtubeMusicBrowsePath: youTubeMusicBrokenBrowsePage(),
	})
	defer server.Close()

	adapter := newYouTubeMusicTestAdapter(server)
	_, err := adapter.SearchByMetadata(context.Background(), youTubeMusicAbbeyRoadAlbum())
	require.Error(t, err)
	assert.ErrorIs(t, err, errMalformedYouTubeMusicPage)
}
