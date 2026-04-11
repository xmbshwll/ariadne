package applemusic

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchByUPCWithOfficialAuth(t *testing.T) {
	fixture := newTestFixture(t, buildTestPayloads(t))

	results, err := fixture.authAdapter.SearchByUPC(context.Background(), "00602567713449")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "1441164426", results[0].CandidateID)
	assert.Equal(t, "https://music.apple.com/gb/album/abbey-road-remastered/1441164426", results[0].MatchURL)
	assert.Equal(t, "gb", results[0].RegionHint)
	assert.Equal(t, "00602567713449", results[0].UPC)
}

func TestSearchByISRCWithOfficialAuth(t *testing.T) {
	fixture := newTestFixture(t, buildTestPayloads(t))

	results, err := fixture.authAdapter.SearchByISRC(context.Background(), []string{comeTogetherISRC})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "1441164426", results[0].CandidateID)
	require.Len(t, results[0].Tracks, 2)
	assert.Equal(t, comeTogetherISRC, results[0].Tracks[0].ISRC)
}

func TestSearchSongByISRCWithOfficialAuth(t *testing.T) {
	fixture := newTestFixture(t, buildTestPayloads(t))

	results, err := fixture.authAdapter.SearchSongByISRC(context.Background(), comeTogetherISRC)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "1441164430", results[0].CandidateID)
	assert.Equal(t, comeTogetherTitle, results[0].Title)
}

func TestHydrateOfficialAlbumsKeepsPartialResultsWhenLaterHydrationFails(t *testing.T) {
	fixture := newTestFixture(t, buildTestPayloads(t))

	results, err := fixture.authAdapter.hydrateOfficialAlbums(context.Background(), []string{"1441164426", "missing"}, "gb")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "1441164426", results[0].CandidateID)
}

func TestHydrateSongsKeepsPartialResultsWhenLaterHydrationFails(t *testing.T) {
	fixture := newTestFixture(t, buildTestPayloads(t))

	results, err := fixture.authAdapter.hydrateSongs(context.Background(), []string{"1441164430", "missing"}, "gb")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "1441164430", results[0].CandidateID)
}
