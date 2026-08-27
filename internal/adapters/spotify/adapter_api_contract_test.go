package spotify_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	spotify "github.com/xmbshwll/ariadne/internal/adapters/spotify"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xmbshwll/ariadne/internal/model"
)

func TestFetchAlbumBootstrapMapsNotFoundStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	adapter := spotify.New(server.Client(), spotify.WithWebBaseURL(server.URL))
	_, err := adapter.FetchAlbumBootstrap(context.Background(), model.ParsedAlbumURL{
		Service:      model.ServiceSpotify,
		EntityType:   model.EntityTypeAlbum,
		ID:           "missing",
		CanonicalURL: "https://open.spotify.com/album/missing",
	})
	require.ErrorIs(t, err, spotify.ErrSpotifyAlbumNotFound)
}

func TestFetchAlbumHydratesTracksViaSingleTrackEndpointInParallel(t *testing.T) {
	release := make(chan struct{})
	var mu sync.Mutex
	started := 0

	adapter := newSpotifyAPIAdapter(t, func(mux *http.ServeMux) {
		mux.HandleFunc("/albums/album-good", func(w http.ResponseWriter, r *http.Request) {
			requireSpotifyBearerAuth(t, r)
			writeJSON(t, w, spotify.APIAlbumResponse{
				ID:          "album-good",
				Name:        "Abbey Road (Remastered)",
				ReleaseDate: "1969-09-26",
				TotalTracks: 2,
				Artists:     []spotify.APIArtist{{Name: "The Beatles"}},
				Tracks: spotify.APITrackPage{Items: []spotify.APITrack{
					{ID: "track-1", Name: "Come Together", TrackNumber: 1, DiscNumber: 1, DurationMS: 258947, Artists: []spotify.APIArtist{{Name: "The Beatles"}}},
					{ID: "track-2", Name: "Something", TrackNumber: 2, DiscNumber: 1, DurationMS: 182293, Artists: []spotify.APIArtist{{Name: "The Beatles"}}},
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

			track, ok := map[string]spotify.APITrack{
				"track-1": {ID: "track-1", Name: "Come Together", TrackNumber: 1, DiscNumber: 1, DurationMS: 258947, ExternalIDs: spotify.APIExternalIDs{ISRC: "GBAYE0601690"}, Artists: []spotify.APIArtist{{Name: "The Beatles"}}, Album: spotify.APITrackAlbum{ID: "album-good", Name: "Abbey Road (Remastered)", ReleaseDate: "1969-09-26", Artists: []spotify.APIArtist{{Name: "The Beatles"}}}},
				"track-2": {ID: "track-2", Name: "Something", TrackNumber: 2, DiscNumber: 1, DurationMS: 182293, ExternalIDs: spotify.APIExternalIDs{ISRC: "GBAYE0601691"}, Artists: []spotify.APIArtist{{Name: "The Beatles"}}, Album: spotify.APITrackAlbum{ID: "album-good", Name: "Abbey Road (Remastered)", ReleaseDate: "1969-09-26", Artists: []spotify.APIArtist{{Name: "The Beatles"}}}},
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

	album, err := adapter.FetchAlbum(ctx, model.ParsedAlbumURL{Service: model.ServiceSpotify, EntityType: model.EntityTypeAlbum, ID: "album-good", CanonicalURL: "https://open.spotify.com/album/album-good"})
	require.NoError(t, err)
	require.NotNil(t, album)
	require.Len(t, album.Tracks, 2)
	assert.Equal(t, "GBAYE0601690", album.Tracks[0].ISRC)
	assert.Equal(t, "GBAYE0601691", album.Tracks[1].ISRC)
}

func TestFetchAlbumSkipsTransientTrackDetailFailures(t *testing.T) {
	adapter := newSpotifyAPIAdapter(t, func(mux *http.ServeMux) {
		mux.HandleFunc("/albums/album-good", func(w http.ResponseWriter, r *http.Request) {
			requireSpotifyBearerAuth(t, r)
			writeJSON(t, w, spotify.APIAlbumResponse{
				ID:          "album-good",
				Name:        "Abbey Road (Remastered)",
				ReleaseDate: "1969-09-26",
				TotalTracks: 2,
				Artists:     []spotify.APIArtist{{Name: "The Beatles"}},
				Tracks: spotify.APITrackPage{Items: []spotify.APITrack{
					{ID: "track-good", Name: "Come Together", TrackNumber: 1, DiscNumber: 1, DurationMS: 258947, Artists: []spotify.APIArtist{{Name: "The Beatles"}}},
					{ID: "track-flaky", Name: "Something", TrackNumber: 2, DiscNumber: 1, DurationMS: 182293, Artists: []spotify.APIArtist{{Name: "The Beatles"}}},
				}},
			})
		})
		registerSpotifyTrackHandler(t, mux, func(w http.ResponseWriter, r *http.Request, trackID string) {
			if trackID == "track-flaky" {
				http.Error(w, "temporary spotify failure", http.StatusBadGateway)
				return
			}
			writeJSON(t, w, spotify.APITrack{ID: "track-good", Name: "Come Together", TrackNumber: 1, DiscNumber: 1, DurationMS: 258947, ExternalIDs: spotify.APIExternalIDs{ISRC: "GBAYE0601690"}, Artists: []spotify.APIArtist{{Name: "The Beatles"}}, Album: spotify.APITrackAlbum{ID: "album-good", Name: "Abbey Road (Remastered)", ReleaseDate: "1969-09-26", Artists: []spotify.APIArtist{{Name: "The Beatles"}}}})
		})
	})

	album, err := adapter.FetchAlbum(context.Background(), model.ParsedAlbumURL{Service: model.ServiceSpotify, EntityType: model.EntityTypeAlbum, ID: "album-good", CanonicalURL: "https://open.spotify.com/album/album-good"})

	require.NoError(t, err)
	require.NotNil(t, album)
	require.Len(t, album.Tracks, 2)
	assert.Equal(t, "GBAYE0601690", album.Tracks[0].ISRC)
	assert.Empty(t, album.Tracks[1].ISRC)
}

func TestFetchAlbumRetriesTransientSpotifyAPIStatus(t *testing.T) {
	albumRequests := 0
	adapter := newSpotifyAPIAdapter(t, func(mux *http.ServeMux) {
		mux.HandleFunc("/albums/album-good", func(w http.ResponseWriter, r *http.Request) {
			requireSpotifyBearerAuth(t, r)
			albumRequests++
			if albumRequests == 1 {
				http.Error(w, "temporary spotify failure", http.StatusBadGateway)
				return
			}
			writeJSON(t, w, spotify.APIAlbumResponse{
				ID:          "album-good",
				Name:        "Abbey Road (Remastered)",
				ReleaseDate: "1969-09-26",
				TotalTracks: 1,
				Artists:     []spotify.APIArtist{{Name: "The Beatles"}},
				Tracks: spotify.APITrackPage{Items: []spotify.APITrack{{
					ID:          "track-1",
					Name:        "Come Together",
					TrackNumber: 1,
					DiscNumber:  1,
					DurationMS:  258947,
					Artists:     []spotify.APIArtist{{Name: "The Beatles"}},
				}}},
			})
		})
		registerSpotifyTrackEndpoint(t, mux, map[string]spotify.APITrack{
			"track-1": {ID: "track-1", Name: "Come Together", TrackNumber: 1, DiscNumber: 1, DurationMS: 258947, ExternalIDs: spotify.APIExternalIDs{ISRC: "GBAYE0601690"}, Artists: []spotify.APIArtist{{Name: "The Beatles"}}, Album: spotify.APITrackAlbum{ID: "album-good", Name: "Abbey Road (Remastered)", ReleaseDate: "1969-09-26", Artists: []spotify.APIArtist{{Name: "The Beatles"}}}},
		})
	})

	album, err := adapter.FetchAlbum(context.Background(), model.ParsedAlbumURL{Service: model.ServiceSpotify, EntityType: model.EntityTypeAlbum, ID: "album-good", CanonicalURL: "https://open.spotify.com/album/album-good"})

	require.NoError(t, err)
	require.NotNil(t, album)
	assert.Equal(t, "album-good", album.SourceID)
	assert.Equal(t, 2, albumRequests)
}

func TestSearchByMetadataSkipsAlbumsThatDisappearDuringHydration(t *testing.T) {
	adapter := newSpotifyAPIAdapter(t, func(mux *http.ServeMux) {
		mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
			requireSpotifyBearerAuth(t, r)
			writeJSON(t, w, spotify.APIAlbumSearchResponse{Albums: spotify.APIAlbumSearchPage{Items: []spotify.APIAlbumSummary{{ID: "album-good"}, {ID: "album-missing"}}}})
		})
		mux.HandleFunc("/albums/album-good", func(w http.ResponseWriter, r *http.Request) {
			requireSpotifyBearerAuth(t, r)
			writeJSON(t, w, spotify.APIAlbumResponse{
				ID:          "album-good",
				Name:        "Abbey Road (Remastered)",
				ReleaseDate: "1969-09-26",
				TotalTracks: 1,
				Artists:     []spotify.APIArtist{{Name: "The Beatles"}},
				Tracks: spotify.APITrackPage{Items: []spotify.APITrack{{
					ID:          "track-1",
					Name:        "Come Together",
					TrackNumber: 1,
					DiscNumber:  1,
					DurationMS:  258947,
					Artists:     []spotify.APIArtist{{Name: "The Beatles"}},
				}}},
			})
		})
		mux.HandleFunc("/albums/album-missing", func(w http.ResponseWriter, r *http.Request) {
			requireSpotifyBearerAuth(t, r)
			http.NotFound(w, r)
		})
		registerSpotifyTrackEndpoint(t, mux, map[string]spotify.APITrack{
			"track-1": {ID: "track-1", Name: "Come Together", TrackNumber: 1, DiscNumber: 1, DurationMS: 258947, Artists: []spotify.APIArtist{{Name: "The Beatles"}}, Album: spotify.APITrackAlbum{ID: "album-good", Name: "Abbey Road (Remastered)", ReleaseDate: "1969-09-26", Artists: []spotify.APIArtist{{Name: "The Beatles"}}}},
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
			writeJSON(t, w, spotify.APIAlbumSearchResponse{Albums: spotify.APIAlbumSearchPage{Items: []spotify.APIAlbumSummary{{ID: "album-good"}}}})
		})
		mux.HandleFunc("/albums/album-good", func(w http.ResponseWriter, r *http.Request) {
			requireSpotifyBearerAuth(t, r)
			writeJSON(t, w, spotify.APIAlbumResponse{
				ID:          "album-good",
				Name:        "ΘΕΛΗΜΑ",
				ReleaseDate: "2024-01-01",
				TotalTracks: 1,
				Artists:     []spotify.APIArtist{{Name: "DECIPHER"}},
				Tracks:      spotify.APITrackPage{Items: []spotify.APITrack{{ID: "track-1", Name: "ΘΕΛΗΜΑ", TrackNumber: 1, DiscNumber: 1, DurationMS: 200000, Artists: []spotify.APIArtist{{Name: "DECIPHER"}}}}},
			})
		})
		registerSpotifyTrackEndpoint(t, mux, map[string]spotify.APITrack{
			"track-1": {ID: "track-1", Name: "ΘΕΛΗΜΑ", TrackNumber: 1, DiscNumber: 1, DurationMS: 200000, Artists: []spotify.APIArtist{{Name: "DECIPHER"}}, Album: spotify.APITrackAlbum{ID: "album-good", Name: "ΘΕΛΗΜΑ", ReleaseDate: "2024-01-01", Artists: []spotify.APIArtist{{Name: "DECIPHER"}}}},
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
	assert.ErrorIs(t, err, spotify.ErrMalformedSpotifyAPIResponse)
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
			writeJSON(t, w, spotify.APITrackSearchResponse{Tracks: spotify.APITrackSearchPage{Items: []spotify.APITrackSearchItem{{ID: "track-good", Name: "ΘΕΛΗΜΑ", DurationMS: 200000, Artists: []spotify.APIArtist{{Name: "DECIPHER"}}}}}})
		})
		registerSpotifyTrackEndpoint(t, mux, map[string]spotify.APITrack{
			"track-good": {ID: "track-good", Name: "ΘΕΛΗΜΑ", TrackNumber: 1, DiscNumber: 1, DurationMS: 200000, Artists: []spotify.APIArtist{{Name: "DECIPHER"}}, Album: spotify.APITrackAlbum{ID: "album-good", Name: "ΘΕΛΗΜΑ", ReleaseDate: "2024-01-01", Artists: []spotify.APIArtist{{Name: "DECIPHER"}}}},
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
			writeJSON(t, w, spotify.APITrackSearchResponse{Tracks: spotify.APITrackSearchPage{Items: []spotify.APITrackSearchItem{{ID: "track-good", Name: "Come Together", DurationMS: 258947, Artists: []spotify.APIArtist{{Name: "The Beatles"}}}, {ID: "track-bad", Name: "Come Together", DurationMS: 200000, Artists: []spotify.APIArtist{{Name: "Tribute Band"}}}}}})
		})
		registerSpotifyTrackHandler(t, mux, func(w http.ResponseWriter, r *http.Request, trackID string) {
			if trackID == "track-bad" {
				http.Error(w, "broken track hydration", http.StatusBadGateway)
				return
			}
			track, ok := map[string]spotify.APITrack{
				"track-good": {ID: "track-good", Name: "Come Together", TrackNumber: 1, DiscNumber: 1, DurationMS: 258947, Artists: []spotify.APIArtist{{Name: "The Beatles"}}, Album: spotify.APITrackAlbum{ID: "album-good", Name: "Abbey Road", ReleaseDate: "1969-09-26", Artists: []spotify.APIArtist{{Name: "The Beatles"}}}},
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
