package spotify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

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

func TestFetchAlbumHydratesTracksViaSingleTrackEndpointInParallel(t *testing.T) {
	release := make(chan struct{})
	var mu sync.Mutex
	started := 0

	adapter := newSpotifyAPIAdapter(t, func(mux *http.ServeMux) {
		mux.HandleFunc("/albums/album-good", func(w http.ResponseWriter, r *http.Request) {
			requireSpotifyBearerAuth(t, r)
			writeJSON(t, w, apiAlbumResponse{
				ID:          "album-good",
				Name:        "Abbey Road (Remastered)",
				ReleaseDate: "1969-09-26",
				TotalTracks: 2,
				Artists:     []apiArtist{{Name: "The Beatles"}},
				Tracks: apiTrackPage{Items: []apiTrack{
					{ID: "track-1", Name: "Come Together", TrackNumber: 1, DiscNumber: 1, DurationMS: 258947, Artists: []apiArtist{{Name: "The Beatles"}}},
					{ID: "track-2", Name: "Something", TrackNumber: 2, DiscNumber: 1, DurationMS: 182293, Artists: []apiArtist{{Name: "The Beatles"}}},
				}},
			})
		})
		registerSpotifyTrackHandler(t, mux, func(w http.ResponseWriter, r *http.Request, trackID string) {
			mu.Lock()
			started++
			if started == 2 {
				close(release)
			}
			mu.Unlock()

			select {
			case <-release:
			case <-time.After(250 * time.Millisecond):
				http.Error(w, "expected parallel track hydration", http.StatusGatewayTimeout)
				return
			}

			track, ok := map[string]apiTrack{
				"track-1": {ID: "track-1", Name: "Come Together", TrackNumber: 1, DiscNumber: 1, DurationMS: 258947, ExternalIDs: apiExternalIDs{ISRC: "GBAYE0601690"}, Artists: []apiArtist{{Name: "The Beatles"}}, Album: apiTrackAlbum{ID: "album-good", Name: "Abbey Road (Remastered)", ReleaseDate: "1969-09-26", Artists: []apiArtist{{Name: "The Beatles"}}}},
				"track-2": {ID: "track-2", Name: "Something", TrackNumber: 2, DiscNumber: 1, DurationMS: 182293, ExternalIDs: apiExternalIDs{ISRC: "GBAYE0601691"}, Artists: []apiArtist{{Name: "The Beatles"}}, Album: apiTrackAlbum{ID: "album-good", Name: "Abbey Road (Remastered)", ReleaseDate: "1969-09-26", Artists: []apiArtist{{Name: "The Beatles"}}}},
			}[trackID]
			if !ok {
				http.NotFound(w, r)
				return
			}
			writeJSON(t, w, track)
		})
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	album, err := adapter.FetchAlbum(ctx, model.ParsedAlbumURL{Service: model.ServiceSpotify, EntityType: "album", ID: "album-good", CanonicalURL: "https://open.spotify.com/album/album-good"})
	require.NoError(t, err)
	require.NotNil(t, album)
	require.Len(t, album.Tracks, 2)
	assert.Equal(t, "GBAYE0601690", album.Tracks[0].ISRC)
	assert.Equal(t, "GBAYE0601691", album.Tracks[1].ISRC)
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
		registerSpotifyTrackEndpoint(t, mux, map[string]apiTrack{
			"track-1": {ID: "track-1", Name: "Come Together", TrackNumber: 1, DiscNumber: 1, DurationMS: 258947, Artists: []apiArtist{{Name: "The Beatles"}}, Album: apiTrackAlbum{ID: "album-good", Name: "Abbey Road (Remastered)", ReleaseDate: "1969-09-26", Artists: []apiArtist{{Name: "The Beatles"}}}},
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
		registerSpotifyTrackEndpoint(t, mux, map[string]apiTrack{
			"track-1": {ID: "track-1", Name: "ΘΕΛΗΜΑ", TrackNumber: 1, DiscNumber: 1, DurationMS: 200000, Artists: []apiArtist{{Name: "DECIPHER"}}, Album: apiTrackAlbum{ID: "album-good", Name: "ΘΕΛΗΜΑ", ReleaseDate: "2024-01-01", Artists: []apiArtist{{Name: "DECIPHER"}}}},
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
		registerSpotifyTrackEndpoint(t, mux, map[string]apiTrack{
			"track-good": {ID: "track-good", Name: "ΘΕΛΗΜΑ", TrackNumber: 1, DiscNumber: 1, DurationMS: 200000, Artists: []apiArtist{{Name: "DECIPHER"}}, Album: apiTrackAlbum{ID: "album-good", Name: "ΘΕΛΗΜΑ", ReleaseDate: "2024-01-01", Artists: []apiArtist{{Name: "DECIPHER"}}}},
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
		registerSpotifyTrackHandler(t, mux, func(w http.ResponseWriter, r *http.Request, trackID string) {
			if trackID == "track-bad" {
				http.Error(w, "broken track hydration", http.StatusBadGateway)
				return
			}
			track, ok := map[string]apiTrack{
				"track-good": {ID: "track-good", Name: "Come Together", TrackNumber: 1, DiscNumber: 1, DurationMS: 258947, Artists: []apiArtist{{Name: "The Beatles"}}, Album: apiTrackAlbum{ID: "album-good", Name: "Abbey Road", ReleaseDate: "1969-09-26", Artists: []apiArtist{{Name: "The Beatles"}}}},
			}[trackID]
			if !ok {
				http.NotFound(w, r)
				return
			}
			writeJSON(t, w, track)
		})
	})

	results, err := adapter.SearchSongByMetadata(context.Background(), model.CanonicalSong{Title: "Come Together", Artists: []string{"The Beatles"}})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "track-good", results[0].CandidateID)
}
