package deezer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
)

func TestSearchByUPCReturnsMissWithoutError(t *testing.T) {
	server := newJSONRouteServer(map[string]jsonRoute{
		"/album/upc:602547670342": jsonOK([]byte(`{"id":0}`)),
	})
	defer server.Close()

	adapter := newTestAdapter(server)
	results, err := adapter.SearchByUPC(context.Background(), "602547670342")
	require.NoError(t, err)
	assert.Nil(t, results)
}

func TestAlbumSearches(t *testing.T) {
	albumBytes := mustReadTestFile(t, "testdata/source-payload.json")
	trackBytes := mustReadTestFile(t, "testdata/tracks.json")
	searchBytes := mustReadTestFile(t, "testdata/search-album-single.json")

	server := newTestServer(t, albumBytes, trackBytes, searchBytes)
	defer server.Close()

	adapter := newTestAdapter(server)
	ctx := context.Background()

	t.Run("search by upc", func(t *testing.T) {
		results, err := adapter.SearchByUPC(ctx, "602547670342")
		require.NoError(t, err)
		assertSingleCandidate(t, results)
	})

	t.Run("search by isrc", func(t *testing.T) {
		results, err := adapter.SearchByISRC(ctx, []string{deezerComeTogetherISRC, "GBAYE0601691"})
		require.NoError(t, err)
		assertSingleCandidate(t, results)
	})

	t.Run("search by metadata", func(t *testing.T) {
		results, err := adapter.SearchByMetadata(ctx, model.CanonicalAlbum{
			Title:   "Abbey Road (Remastered)",
			Artists: []string{"The Beatles"},
		})
		require.NoError(t, err)
		assertSingleCandidate(t, results)
	})
}

func TestSongSearches(t *testing.T) {
	albumBytes := mustReadTestFile(t, "testdata/source-payload.json")
	trackBytes := mustReadTestFile(t, "testdata/tracks.json")
	searchBytes := mustReadTestFile(t, "testdata/search-album-single.json")

	server := newTestServer(t, albumBytes, trackBytes, searchBytes)
	defer server.Close()

	adapter := newTestAdapter(server)
	ctx := context.Background()

	isrcResults, err := adapter.SearchSongByISRC(ctx, deezerComeTogetherISRC)
	require.NoError(t, err)
	assertSingleSongCandidate(t, isrcResults)

	metadataResults, err := adapter.SearchSongByMetadata(ctx, model.CanonicalSong{Title: "Come Together", Artists: []string{"The Beatles"}})
	require.NoError(t, err)
	require.Len(t, metadataResults, 2)
	assert.Equal(t, "116348128", metadataResults[0].CandidateID)
}

func TestSearchByISRCKeepsEarlierResultsWhenLaterQueriesFail(t *testing.T) {
	albumBytes := mustReadTestFile(t, "testdata/source-payload.json")
	trackBytes := mustReadTestFile(t, "testdata/tracks.json")

	server := newJSONRouteServer(map[string]jsonRoute{
		"/track/isrc:" + deezerComeTogetherISRC: jsonOK([]byte(deezerComeTogetherTrackPayload)),
		"/track/isrc:BADISRC":                   jsonError(http.StatusBadGateway, "temporary failure"),
		deezerAlbumPath:                         jsonOK(albumBytes),
		deezerAlbumTracksPath:                   jsonOK(trackBytes),
	})
	defer server.Close()

	adapter := newTestAdapter(server)
	results, err := adapter.SearchByISRC(context.Background(), []string{deezerComeTogetherISRC, "BADISRC"})
	require.NoError(t, err)
	assertSingleCandidate(t, results)
}

func TestSearchByMetadataKeepsEarlierResultsWhenLaterHydrationFails(t *testing.T) {
	albumBytes := mustReadTestFile(t, "testdata/source-payload.json")
	trackBytes := mustReadTestFile(t, "testdata/tracks.json")

	var server *httptest.Server
	server = newJSONTestServer(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case deezerAlbumSearchPath:
			_, _ = w.Write([]byte(`{"data":[{"id":12047952,"title":"Abbey Road (Remastered)"},{"id":555,"title":"Broken Album"}]}`))
		case deezerAlbumPath:
			_, _ = w.Write(albumBytes)
		case deezerAlbumTracksPath:
			_, _ = w.Write(trackBytes)
		case "/album/555":
			_, _ = w.Write([]byte(`{"id":555,"title":"Broken Album","tracklist":"` + server.URL + `/album/555/tracks"}`))
		case "/album/555/tracks":
			http.Error(w, "temporary failure", http.StatusBadGateway)
		default:
			http.NotFound(w, r)
		}
	})
	defer server.Close()

	adapter := newTestAdapter(server)
	results, err := adapter.SearchByMetadata(context.Background(), model.CanonicalAlbum{Title: "Abbey Road", Artists: []string{"The Beatles"}})
	require.NoError(t, err)
	assertSingleCandidate(t, results)
}

func TestSearchByISRCSkipsTracksWithoutAlbumIDs(t *testing.T) {
	server := newJSONRouteServer(map[string]jsonRoute{
		"/track/isrc:" + deezerComeTogetherISRC: jsonOK(mustReadTestFile(t, "testdata/track-without-album-id.json")),
	})
	defer server.Close()

	adapter := newTestAdapter(server)
	results, err := adapter.SearchByISRC(context.Background(), []string{deezerComeTogetherISRC})
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestSearchByMetadataSkipsNonPositiveAlbumIDs(t *testing.T) {
	server := newJSONRouteServer(map[string]jsonRoute{
		deezerAlbumSearchPath: jsonOK(mustReadTestFile(t, "testdata/search-album-non-positive-id.json")),
	})
	defer server.Close()

	adapter := newTestAdapter(server)
	results, err := adapter.SearchByMetadata(context.Background(), model.CanonicalAlbum{Title: "Abbey Road", Artists: []string{"The Beatles"}})
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestSearchSongByMetadataSkipsNonPositiveTrackIDs(t *testing.T) {
	server := newJSONRouteServer(map[string]jsonRoute{
		deezerTrackSearchPath: jsonOK(mustReadTestFile(t, "testdata/search-track-non-positive-id.json")),
	})
	defer server.Close()

	adapter := newTestAdapter(server)
	results, err := adapter.SearchSongByMetadata(context.Background(), model.CanonicalSong{Title: "Come Together", Artists: []string{"The Beatles"}})
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestSearchSongByMetadataReturnsMalformedResponseError(t *testing.T) {
	server := newJSONRouteServer(map[string]jsonRoute{
		deezerTrackSearchPath: jsonOK([]byte("{")),
	})
	defer server.Close()

	adapter := newTestAdapter(server)
	_, err := adapter.SearchSongByMetadata(context.Background(), model.CanonicalSong{Title: "Come Together", Artists: []string{"The Beatles"}})
	require.Error(t, err)
	assert.ErrorIs(t, err, errMalformedDeezerResponse)
}
