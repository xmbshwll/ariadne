package applemusic

import (
	"context"
	"net/http"
	"net/http/httptest"
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

func TestSearchAlbumByMetadataKeepsEarlierResultsWhenLaterQueriesFail(t *testing.T) {
	payloads := buildTestPayloads(t)
	searchRequests := 0

	mux := http.NewServeMux()
	mux.HandleFunc("/lookup", lookupHandler(payloads))
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		searchRequests++
		if searchRequests > 1 {
			http.Error(w, "transient search failure", http.StatusBadGateway)
			return
		}
		searchHandler(payloads)(w, r)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	adapter := New(server.Client(), WithLookupBaseURL(server.URL), WithDefaultStorefront("gb"))
	results, err := adapter.SearchByMetadata(context.Background(), model.CanonicalAlbum{
		Title:   abbeyRoadRemastered,
		Artists: []string{"The Beatles"},
	})
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Greater(t, searchRequests, 1)
}

func TestBuildMetadataQueriesPrefersArtistQueriesBeforeTitleOnlyFallbacks(t *testing.T) {
	queries := buildMetadataQueries("Solid Static (Deluxe Edition)", []string{"Musica Transonic + Mainliner"})
	require.GreaterOrEqual(t, len(queries), 4)
	assert.Equal(t, "Solid Static (Deluxe Edition) Musica Transonic + Mainliner", queries[0])
	assert.Equal(t, "Solid Static (Deluxe Edition)", queries[len(queries)-2])
	assert.Equal(t, "Solid Static", queries[len(queries)-1])
}
