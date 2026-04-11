package spotify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
)

func TestAPIBackedAlbumAndSongOperations(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		requireSpotifyTokenRequest(t, r)
		_ = json.NewEncoder(w).Encode(tokenResponse{AccessToken: "token-123", TokenType: "Bearer", ExpiresIn: 3600})
	})
	mux.HandleFunc("/albums/album-good", func(w http.ResponseWriter, r *http.Request) {
		requireSpotifyBearerAuth(t, r)
		writeJSON(t, w, apiAlbumResponse{
			ID:          "album-good",
			Name:        "Abbey Road (Remastered)",
			ReleaseDate: "1969-09-26",
			Label:       "EMI Catalogue",
			TotalTracks: 17,
			Images:      []apiImage{{URL: "https://i.scdn.co/image/best", Width: 640}},
			Artists:     []apiArtist{{Name: "The Beatles"}},
			ExternalIDs: apiExternalIDs{UPC: "602547670342"},
			Tracks: apiTrackPage{Items: []apiTrack{
				{ID: "track-1", Name: "Come Together", TrackNumber: 1, DiscNumber: 1, DurationMS: 258947, Artists: []apiArtist{{Name: "The Beatles"}}},
				{ID: "track-2", Name: "Something", TrackNumber: 2, DiscNumber: 1, DurationMS: 182293, Artists: []apiArtist{{Name: "The Beatles"}}},
			}},
		})
	})
	mux.HandleFunc("/albums/album-weak", func(w http.ResponseWriter, r *http.Request) {
		requireSpotifyBearerAuth(t, r)
		writeJSON(t, w, apiAlbumResponse{
			ID:          "album-weak",
			Name:        "Abbey Road",
			ReleaseDate: "2020-01-01",
			Label:       "Other Label",
			TotalTracks: 17,
			Images:      []apiImage{{URL: "https://i.scdn.co/image/weak", Width: 640}},
			Artists:     []apiArtist{{Name: "The Beatles Complete On Ukulele"}},
			Tracks: apiTrackPage{Items: []apiTrack{
				{ID: "track-weak-1", Name: "Come Together", TrackNumber: 1, DiscNumber: 1, DurationMS: 200000, Artists: []apiArtist{{Name: "The Beatles Complete On Ukulele"}}},
			}},
		})
	})
	mux.HandleFunc("/tracks/track-1", func(w http.ResponseWriter, r *http.Request) {
		requireSpotifyBearerAuth(t, r)
		writeJSON(t, w, apiTrack{ID: "track-1", Name: "Come Together", TrackNumber: 1, DiscNumber: 1, DurationMS: 258947, ExternalIDs: apiExternalIDs{ISRC: "GBAYE0601690"}, Artists: []apiArtist{{Name: "The Beatles"}}, Album: apiTrackAlbum{ID: "album-good", Name: "Abbey Road (Remastered)", ReleaseDate: "1969-09-26", Images: []apiImage{{URL: "https://i.scdn.co/image/best", Width: 640}}, Artists: []apiArtist{{Name: "The Beatles"}}}})
	})
	mux.HandleFunc("/tracks/track-2", func(w http.ResponseWriter, r *http.Request) {
		requireSpotifyBearerAuth(t, r)
		writeJSON(t, w, apiTrack{ID: "track-2", Name: "Something", TrackNumber: 2, DiscNumber: 1, DurationMS: 182293, ExternalIDs: apiExternalIDs{ISRC: "GBAYE0601691"}, Artists: []apiArtist{{Name: "The Beatles"}}, Album: apiTrackAlbum{ID: "album-good", Name: "Abbey Road (Remastered)", ReleaseDate: "1969-09-26", Images: []apiImage{{URL: "https://i.scdn.co/image/best", Width: 640}}, Artists: []apiArtist{{Name: "The Beatles"}}}})
	})
	mux.HandleFunc("/tracks/track-weak-1", func(w http.ResponseWriter, r *http.Request) {
		requireSpotifyBearerAuth(t, r)
		writeJSON(t, w, apiTrack{ID: "track-weak-1", Name: "Come Together", TrackNumber: 1, DiscNumber: 1, DurationMS: 200000, ExternalIDs: apiExternalIDs{ISRC: "OTHER0001"}, Artists: []apiArtist{{Name: "The Beatles Complete On Ukulele"}}, Album: apiTrackAlbum{ID: "album-weak", Name: "Abbey Road", ReleaseDate: "2020-01-01", Images: []apiImage{{URL: "https://i.scdn.co/image/weak", Width: 640}}, Artists: []apiArtist{{Name: "The Beatles Complete On Ukulele"}}}})
	})
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		requireSpotifyBearerAuth(t, r)
		query := r.URL.Query().Get("q")
		switch {
		case strings.Contains(query, "upc:602547670342"):
			writeJSON(t, w, apiAlbumSearchResponse{Albums: apiAlbumSearchPage{Items: []apiAlbumSummary{{ID: "album-good"}}}})
		case strings.Contains(query, "isrc:GBAYE0601690"):
			writeJSON(t, w, apiTrackSearchResponse{Tracks: apiTrackSearchPage{Items: []apiTrackSearchItem{{ID: "track-1", Name: "Come Together", DurationMS: 258947, Artists: []apiArtist{{Name: "The Beatles"}}, ExternalIDs: apiExternalIDs{ISRC: "GBAYE0601690"}, Album: apiTrackAlbum{ID: "album-good", Name: "Abbey Road (Remastered)", ReleaseDate: "1969-09-26", Images: []apiImage{{URL: "https://i.scdn.co/image/best", Width: 640}}, Artists: []apiArtist{{Name: "The Beatles"}}}}}}})
		case strings.Contains(query, "isrc:GBAYE0601691"):
			writeJSON(t, w, apiTrackSearchResponse{Tracks: apiTrackSearchPage{Items: []apiTrackSearchItem{{ID: "track-2", Name: "Something", DurationMS: 182293, Artists: []apiArtist{{Name: "The Beatles"}}, ExternalIDs: apiExternalIDs{ISRC: "GBAYE0601691"}, Album: apiTrackAlbum{ID: "album-good", Name: "Abbey Road (Remastered)", ReleaseDate: "1969-09-26", Images: []apiImage{{URL: "https://i.scdn.co/image/best", Width: 640}}, Artists: []apiArtist{{Name: "The Beatles"}}}}}}})
		case strings.Contains(query, "album:Abbey Road (Remastered)"), strings.Contains(query, "album:Abbey Road artist:The Beatles"), strings.Contains(query, "album:Abbey Road"):
			writeJSON(t, w, apiAlbumSearchResponse{Albums: apiAlbumSearchPage{Items: []apiAlbumSummary{{ID: "album-good"}, {ID: "album-weak"}}}})
		case strings.Contains(query, "track:Come Together artist:The Beatles"), strings.Contains(query, "track:Come Together"):
			writeJSON(t, w, apiTrackSearchResponse{Tracks: apiTrackSearchPage{Items: []apiTrackSearchItem{{ID: "track-1", Name: "Come Together", DurationMS: 258947, Artists: []apiArtist{{Name: "The Beatles"}}, ExternalIDs: apiExternalIDs{ISRC: "GBAYE0601690"}, Album: apiTrackAlbum{ID: "album-good", Name: "Abbey Road (Remastered)", ReleaseDate: "1969-09-26", Images: []apiImage{{URL: "https://i.scdn.co/image/best", Width: 640}}, Artists: []apiArtist{{Name: "The Beatles"}}}}, {ID: "track-weak-1", Name: "Come Together", DurationMS: 200000, Artists: []apiArtist{{Name: "The Beatles Complete On Ukulele"}}, ExternalIDs: apiExternalIDs{ISRC: "OTHER0001"}, Album: apiTrackAlbum{ID: "album-weak", Name: "Abbey Road", ReleaseDate: "2020-01-01", Images: []apiImage{{URL: "https://i.scdn.co/image/weak", Width: 640}}, Artists: []apiArtist{{Name: "The Beatles Complete On Ukulele"}}}}}}})
		default:
			http.NotFound(w, r)
		}
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	adapter := New(
		server.Client(),
		WithCredentials("client-id", "client-secret"),
		WithAPIBaseURL(server.URL),
		WithAuthBaseURL(server.URL),
	)

	parsed := model.ParsedAlbumURL{Service: model.ServiceSpotify, EntityType: "album", ID: "album-good", CanonicalURL: "https://open.spotify.com/album/album-good"}
	album, err := adapter.FetchAlbum(context.Background(), parsed)
	require.NoError(t, err)
	require.NotNil(t, album)
	require.NotEmpty(t, album.UPC)
	require.NotEmpty(t, album.Tracks)
	assert.Equal(t, "602547670342", album.UPC)
	assert.Equal(t, "GBAYE0601690", album.Tracks[0].ISRC)

	upcResults, err := adapter.SearchByUPC(context.Background(), "602547670342")
	require.NoError(t, err)
	assertSingleAlbum(t, upcResults, "album-good")

	isrcResults, err := adapter.SearchByISRC(context.Background(), []string{"GBAYE0601690", "GBAYE0601691"})
	require.NoError(t, err)
	assertSingleAlbum(t, isrcResults, "album-good")

	metadataResults, err := adapter.SearchByMetadata(context.Background(), model.CanonicalAlbum{Title: "Abbey Road (Remastered)", Artists: []string{"The Beatles"}})
	require.NoError(t, err)
	require.Len(t, metadataResults, 2)
	assert.Equal(t, "album-good", metadataResults[0].CandidateID)

	song, err := adapter.FetchSong(context.Background(), model.ParsedAlbumURL{Service: model.ServiceSpotify, EntityType: "song", ID: "track-1", CanonicalURL: "https://open.spotify.com/track/track-1"})
	require.NoError(t, err)
	require.NotNil(t, song)
	require.NotEmpty(t, song.ISRC)
	require.NotEmpty(t, song.AlbumTitle)
	assert.Equal(t, "GBAYE0601690", song.ISRC)
	assert.Equal(t, "Abbey Road (Remastered)", song.AlbumTitle)

	songISRCResults, err := adapter.SearchSongByISRC(context.Background(), "GBAYE0601690")
	require.NoError(t, err)
	assertSingleSong(t, songISRCResults, "track-1")

	songMetadataResults, err := adapter.SearchSongByMetadata(context.Background(), model.CanonicalSong{Title: "Come Together", Artists: []string{"The Beatles"}})
	require.NoError(t, err)
	require.Len(t, songMetadataResults, 2)
	assert.Equal(t, "track-1", songMetadataResults[0].CandidateID)
}

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
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		requireSpotifyTokenRequest(t, r)
		_ = json.NewEncoder(w).Encode(tokenResponse{AccessToken: "token-123", TokenType: "Bearer", ExpiresIn: 3600})
	})
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
	mux.HandleFunc("/tracks/track-1", func(w http.ResponseWriter, r *http.Request) {
		requireSpotifyBearerAuth(t, r)
		writeJSON(t, w, apiTrack{ID: "track-1", Name: "Come Together", TrackNumber: 1, DiscNumber: 1, DurationMS: 258947, Artists: []apiArtist{{Name: "The Beatles"}}, Album: apiTrackAlbum{ID: "album-good", Name: "Abbey Road (Remastered)", ReleaseDate: "1969-09-26", Artists: []apiArtist{{Name: "The Beatles"}}}})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	adapter := New(server.Client(), WithCredentials("client-id", "client-secret"), WithAPIBaseURL(server.URL), WithAuthBaseURL(server.URL))
	results, err := adapter.SearchByMetadata(context.Background(), model.CanonicalAlbum{Title: "Abbey Road", Artists: []string{"The Beatles"}})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "album-good", results[0].CandidateID)
}

func requireSpotifyTokenRequest(t *testing.T, r *http.Request) {
	t.Helper()
	require.Equal(t, http.MethodPost, r.Method)
	require.NoError(t, r.ParseForm())
	assert.Equal(t, "client_credentials", r.Form.Get("grant_type"))
	assert.True(t, strings.HasPrefix(r.Header.Get("Authorization"), "Basic "))
}

func requireSpotifyBearerAuth(t *testing.T, r *http.Request) {
	t.Helper()
	assert.Equal(t, "Bearer token-123", r.Header.Get("Authorization"))
}
