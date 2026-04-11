package applemusic

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
)

func TestSearchAlbumByMetadata(t *testing.T) {
	fixture := newTestFixture(t, buildTestPayloads(t))

	results, err := fixture.adapter.SearchByMetadata(context.Background(), model.CanonicalAlbum{
		Title:      abbeyRoadRemastered,
		Artists:    []string{"The Beatles"},
		RegionHint: "gb",
	})
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, "1474815798", results[0].CandidateID)
	assert.Equal(t, "1441164426", results[1].CandidateID)
	assert.Equal(t, "https://music.apple.com/us/album/abbey-road-remastered/1441164426", results[1].MatchURL)
	assert.Equal(t, "gb", results[1].RegionHint)
}

func TestSearchAlbumByMetadataUsesAdapterDefaultStorefront(t *testing.T) {
	payloads := buildTestPayloads(t)
	fixture := newTestFixture(t, payloads)
	defaultStorefrontAdapter := New(fixture.httpClient, WithLookupBaseURL(fixture.serverURL), WithDefaultStorefront("gb"))

	results, err := defaultStorefrontAdapter.SearchByMetadata(context.Background(), model.CanonicalAlbum{
		Title:   abbeyRoadRemastered,
		Artists: []string{"The Beatles"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, results)
	assert.Equal(t, "gb", results[0].RegionHint)
}

func TestSearchByUPCWithoutOfficialAuthReturnsNoResults(t *testing.T) {
	fixture := newTestFixture(t, buildTestPayloads(t))

	results, err := fixture.adapter.SearchByUPC(context.Background(), "123")
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestSearchByISRCWithoutOfficialAuthReturnsNoResults(t *testing.T) {
	fixture := newTestFixture(t, buildTestPayloads(t))

	results, err := fixture.adapter.SearchByISRC(context.Background(), []string{"ABC"})
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestSearchSongByMetadata(t *testing.T) {
	fixture := newTestFixture(t, buildTestPayloads(t))

	results, err := fixture.adapter.SearchSongByMetadata(context.Background(), model.CanonicalSong{
		Title:      comeTogetherTitle,
		Artists:    []string{"The Beatles"},
		RegionHint: "gb",
	})
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, "1441164430", results[0].CandidateID)
	assert.Equal(t, abbeyRoadRemastered, results[0].AlbumTitle)
	assert.Equal(t, comeTogetherISRC, results[0].ISRC)
}
