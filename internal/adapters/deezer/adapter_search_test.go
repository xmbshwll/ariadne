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

func TestAlbumSearches(t *testing.T) {
	albumBytes := mustReadTestFile(t, "testdata/source-payload.json")
	trackBytes := mustReadTestFile(t, "testdata/tracks.json")
	searchBytes := []byte(`{"data":[{"id":12047952,"title":"Abbey Road (Remastered)"}]}`)

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
	searchBytes := []byte(`{"data":[{"id":12047952,"title":"Abbey Road (Remastered)"}]}`)

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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/track/isrc:" + deezerComeTogetherISRC:
			_, _ = w.Write([]byte(deezerComeTogetherTrackPayload))
		case "/track/isrc:BADISRC":
			http.Error(w, "temporary failure", http.StatusBadGateway)
		case deezerAlbumPath:
			_, _ = w.Write(albumBytes)
		case deezerAlbumTracksPath:
			_, _ = w.Write(trackBytes)
		default:
			http.NotFound(w, r)
		}
	}))
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
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/search/album":
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
	}))
	defer server.Close()

	adapter := newTestAdapter(server)

	results, err := adapter.SearchByMetadata(context.Background(), model.CanonicalAlbum{Title: "Abbey Road", Artists: []string{"The Beatles"}})
	require.NoError(t, err)
	assertSingleCandidate(t, results)
}

func TestSearchSongByMetadataReturnsMalformedResponseError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/search/track" {
			_, _ = w.Write([]byte("{"))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	adapter := newTestAdapter(server)

	_, err := adapter.SearchSongByMetadata(context.Background(), model.CanonicalSong{Title: "Come Together", Artists: []string{"The Beatles"}})
	require.Error(t, err)
	assert.ErrorIs(t, err, errMalformedDeezerResponse)
}
