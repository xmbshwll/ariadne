package spotify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
)

func TestFetchAlbumBootstrapMapsNotFoundStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	adapter := New(server.Client(), WithWebBaseURL(server.URL))
	_, err := adapter.fetchAlbumBootstrap(context.Background(), model.ParsedAlbumURL{
		Service:      model.ServiceSpotify,
		EntityType:   "album",
		ID:           "missing",
		CanonicalURL: "https://open.spotify.com/album/missing",
	})
	require.ErrorIs(t, err, errSpotifyAlbumNotFound)
}

func TestSearchByMetadataSkipsAlbumsThatDisappearDuringHydration(t *testing.T) {
	adapter := newSpotifyAPIAdapter(t, func(mux *http.ServeMux) {
		mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
			requireSpotifyBearerAuth(t, r)
			writeJSON(t, w, apiAlbumSearchResponse{Albums: apiAlbumSearchPage{Items: []apiAlbumSummary{{ID: "album-good"}, {ID: "album-missing"}}}})
		})
		mux.HandleFunc("/albums/album-good", func(w http.ResponseWriter, r *http.Request) {
			requireSpotifyBearerAuth(t, r)
			writeJSON(t, w, apiAlbumResponse{
				ID:          "album-good",
				Name:        "Abbey Road (Remastered)",
				ReleaseDate: "1969-09-26",
				TotalTracks: 1,
				Artists:     []apiArtist{{Name: "The Beatles"}},
				Tracks: apiTrackPage{Items: []apiTrack{{
					ID:          "track-1",
					Name:        "Come Together",
					TrackNumber: 1,
					DiscNumber:  1,
					DurationMS:  258947,
					Artists:     []apiArtist{{Name: "The Beatles"}},
				}}},
			})
		})
		mux.HandleFunc("/albums/album-missing", func(w http.ResponseWriter, r *http.Request) {
			requireSpotifyBearerAuth(t, r)
			http.NotFound(w, r)
		})
		mux.HandleFunc("/tracks", func(w http.ResponseWriter, r *http.Request) {
			requireSpotifyBearerAuth(t, r)
			writeSpotifyTrackBatchJSON(t, w, r, map[string]apiTrack{
				"track-1": {ID: "track-1", Name: "Come Together", TrackNumber: 1, DiscNumber: 1, DurationMS: 258947, Artists: []apiArtist{{Name: "The Beatles"}}, Album: apiTrackAlbum{ID: "album-good", Name: "Abbey Road (Remastered)", ReleaseDate: "1969-09-26", Artists: []apiArtist{{Name: "The Beatles"}}}},
			})
		})
	})

	results, err := adapter.SearchByMetadata(context.Background(), model.CanonicalAlbum{Title: "Abbey Road", Artists: []string{"The Beatles"}})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "album-good", results[0].CandidateID)
}

func TestSearchByMetadataKeepsEarlierResultsWhenLaterQueriesFail(t *testing.T) {
	searchRequests := 0
	adapter := newSpotifyAPIAdapter(t, func(mux *http.ServeMux) {
		mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
			requireSpotifyBearerAuth(t, r)
			searchRequests++
			if searchRequests > 1 {
				http.Error(w, "temporary spotify failure", http.StatusBadGateway)
				return
			}
			writeJSON(t, w, apiAlbumSearchResponse{Albums: apiAlbumSearchPage{Items: []apiAlbumSummary{{ID: "album-good"}}}})
		})
		mux.HandleFunc("/albums/album-good", func(w http.ResponseWriter, r *http.Request) {
			requireSpotifyBearerAuth(t, r)
			writeJSON(t, w, apiAlbumResponse{
				ID:          "album-good",
				Name:        "ΘΕΛΗΜΑ",
				ReleaseDate: "2024-01-01",
				TotalTracks: 1,
				Artists:     []apiArtist{{Name: "DECIPHER"}},
				Tracks:      apiTrackPage{Items: []apiTrack{{ID: "track-1", Name: "ΘΕΛΗΜΑ", TrackNumber: 1, DiscNumber: 1, DurationMS: 200000, Artists: []apiArtist{{Name: "DECIPHER"}}}}},
			})
		})
		mux.HandleFunc("/tracks", func(w http.ResponseWriter, r *http.Request) {
			requireSpotifyBearerAuth(t, r)
			writeSpotifyTrackBatchJSON(t, w, r, map[string]apiTrack{
				"track-1": {ID: "track-1", Name: "ΘΕΛΗΜΑ", TrackNumber: 1, DiscNumber: 1, DurationMS: 200000, Artists: []apiArtist{{Name: "DECIPHER"}}, Album: apiTrackAlbum{ID: "album-good", Name: "ΘΕΛΗΜΑ", ReleaseDate: "2024-01-01", Artists: []apiArtist{{Name: "DECIPHER"}}}},
			})
		})
	})

	results, err := adapter.SearchByMetadata(context.Background(), model.CanonicalAlbum{Title: "ΘΕΛΗΜΑ (Thelema)", Artists: []string{"DECIPHER"}})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "album-good", results[0].CandidateID)
	assert.Greater(t, searchRequests, 1)
}

func TestSearchByMetadataReturnsMalformedResponseError(t *testing.T) {
	adapter := newSpotifyAPIAdapter(t, func(mux *http.ServeMux) {
		mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
			requireSpotifyBearerAuth(t, r)
			_, _ = w.Write([]byte("{"))
		})
	})

	_, err := adapter.SearchByMetadata(context.Background(), model.CanonicalAlbum{Title: "Abbey Road", Artists: []string{"The Beatles"}})
	require.Error(t, err)
	assert.ErrorIs(t, err, errMalformedSpotifyAPIResponse)
}

func TestSearchSongByMetadataKeepsEarlierResultsWhenLaterQueriesFail(t *testing.T) {
	searchRequests := 0
	adapter := newSpotifyAPIAdapter(t, func(mux *http.ServeMux) {
		mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
			requireSpotifyBearerAuth(t, r)
			searchRequests++
			if searchRequests > 1 {
				http.Error(w, "temporary spotify failure", http.StatusBadGateway)
				return
			}
			writeJSON(t, w, apiTrackSearchResponse{Tracks: apiTrackSearchPage{Items: []apiTrackSearchItem{{ID: "track-good", Name: "ΘΕΛΗΜΑ", DurationMS: 200000, Artists: []apiArtist{{Name: "DECIPHER"}}}}}})
		})
		mux.HandleFunc("/tracks", func(w http.ResponseWriter, r *http.Request) {
			requireSpotifyBearerAuth(t, r)
			writeSpotifyTrackBatchJSON(t, w, r, map[string]apiTrack{
				"track-good": {ID: "track-good", Name: "ΘΕΛΗΜΑ", TrackNumber: 1, DiscNumber: 1, DurationMS: 200000, Artists: []apiArtist{{Name: "DECIPHER"}}, Album: apiTrackAlbum{ID: "album-good", Name: "ΘΕΛΗΜΑ", ReleaseDate: "2024-01-01", Artists: []apiArtist{{Name: "DECIPHER"}}}},
			})
		})
	})

	results, err := adapter.SearchSongByMetadata(context.Background(), model.CanonicalSong{Title: "ΘΕΛΗΜΑ (Thelema)", Artists: []string{"DECIPHER"}})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "track-good", results[0].CandidateID)
	assert.Greater(t, searchRequests, 1)
}

func TestSearchSongByMetadataKeepsPartialResultsWhenLaterHydrationFails(t *testing.T) {
	adapter := newSpotifyAPIAdapter(t, func(mux *http.ServeMux) {
		mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
			requireSpotifyBearerAuth(t, r)
			writeJSON(t, w, apiTrackSearchResponse{Tracks: apiTrackSearchPage{Items: []apiTrackSearchItem{{ID: "track-good", Name: "Come Together", DurationMS: 258947, Artists: []apiArtist{{Name: "The Beatles"}}}, {ID: "track-bad", Name: "Come Together", DurationMS: 200000, Artists: []apiArtist{{Name: "Tribute Band"}}}}}})
		})
		mux.HandleFunc("/tracks", func(w http.ResponseWriter, r *http.Request) {
			requireSpotifyBearerAuth(t, r)
			if r.URL.Query().Get("ids") == "track-bad" {
				http.Error(w, "broken track hydration", http.StatusBadGateway)
				return
			}
			writeSpotifyTrackBatchJSON(t, w, r, map[string]apiTrack{
				"track-good": {ID: "track-good", Name: "Come Together", TrackNumber: 1, DiscNumber: 1, DurationMS: 258947, Artists: []apiArtist{{Name: "The Beatles"}}, Album: apiTrackAlbum{ID: "album-good", Name: "Abbey Road", ReleaseDate: "1969-09-26", Artists: []apiArtist{{Name: "The Beatles"}}}},
			})
		})
	})

	results, err := adapter.SearchSongByMetadata(context.Background(), model.CanonicalSong{Title: "Come Together", Artists: []string{"The Beatles"}})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "track-good", results[0].CandidateID)
}
