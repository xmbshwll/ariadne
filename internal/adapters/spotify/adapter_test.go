package spotify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
)

func TestAdapter(t *testing.T) {
	html := mustReadTestFile(t, "testdata/source-page.html")

	t.Run("fetch album via bootstrap", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/album/0ETFjACtuP2ADo6LFhL6HN" {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write(html)
		}))
		defer server.Close()

		adapter := New(server.Client(), WithWebBaseURL(server.URL))
		parsed := model.ParsedAlbumURL{
			Service:      model.ServiceSpotify,
			EntityType:   "album",
			ID:           "0ETFjACtuP2ADo6LFhL6HN",
			CanonicalURL: "https://open.spotify.com/album/0ETFjACtuP2ADo6LFhL6HN",
			RawURL:       "https://open.spotify.com/album/0ETFjACtuP2ADo6LFhL6HN",
		}

		album, err := adapter.FetchAlbum(context.Background(), parsed)
		require.NoError(t, err)
		assert.Equal(t, "Abbey Road (Remastered)", album.Title)
		assert.NotEmpty(t, album.Label)
		assert.Equal(t, 17, album.TrackCount)
		require.Len(t, album.Tracks, 17)
		assert.Equal(t, "Come Together - Remastered 2009", album.Tracks[0].Title)
		assert.Equal(t, 259946, album.Tracks[0].DurationMS)
		assert.NotEmpty(t, album.ArtworkURL)
	})

	t.Run("api fetch and target search", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			_ = json.NewEncoder(w).Encode(tokenResponse{AccessToken: "token-123", TokenType: "Bearer", ExpiresIn: 3600})
		})
		mux.HandleFunc("/albums/album-good", func(w http.ResponseWriter, r *http.Request) {
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
			writeJSON(t, w, apiTrack{ID: "track-1", Name: "Come Together", TrackNumber: 1, DiscNumber: 1, DurationMS: 258947, ExternalIDs: apiExternalIDs{ISRC: "GBAYE0601690"}, Artists: []apiArtist{{Name: "The Beatles"}}, Album: apiTrackAlbum{ID: "album-good", Name: "Abbey Road (Remastered)", ReleaseDate: "1969-09-26", Images: []apiImage{{URL: "https://i.scdn.co/image/best", Width: 640}}, Artists: []apiArtist{{Name: "The Beatles"}}}})
		})
		mux.HandleFunc("/tracks/track-2", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, apiTrack{ID: "track-2", Name: "Something", TrackNumber: 2, DiscNumber: 1, DurationMS: 182293, ExternalIDs: apiExternalIDs{ISRC: "GBAYE0601691"}, Artists: []apiArtist{{Name: "The Beatles"}}, Album: apiTrackAlbum{ID: "album-good", Name: "Abbey Road (Remastered)", ReleaseDate: "1969-09-26", Images: []apiImage{{URL: "https://i.scdn.co/image/best", Width: 640}}, Artists: []apiArtist{{Name: "The Beatles"}}}})
		})
		mux.HandleFunc("/tracks/track-weak-1", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, apiTrack{ID: "track-weak-1", Name: "Come Together", TrackNumber: 1, DiscNumber: 1, DurationMS: 200000, ExternalIDs: apiExternalIDs{ISRC: "OTHER0001"}, Artists: []apiArtist{{Name: "The Beatles Complete On Ukulele"}}, Album: apiTrackAlbum{ID: "album-weak", Name: "Abbey Road", ReleaseDate: "2020-01-01", Images: []apiImage{{URL: "https://i.scdn.co/image/weak", Width: 640}}, Artists: []apiArtist{{Name: "The Beatles Complete On Ukulele"}}}})
		})
		mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
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
		require.NotNil(t, &album.Tracks[0])
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
	})
}

func assertSingleAlbum(t *testing.T, candidates []model.CandidateAlbum, wantID string) {
	t.Helper()
	require.Len(t, candidates, 1)
	assert.Equal(t, wantID, candidates[0].CandidateID)
}

func assertSingleSong(t *testing.T, candidates []model.CandidateSong, wantID string) {
	t.Helper()
	require.Len(t, candidates, 1)
	assert.Equal(t, wantID, candidates[0].CandidateID)
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	require.NoError(t, json.NewEncoder(w).Encode(payload))
}

func mustReadTestFile(t *testing.T, relativePath string) []byte {
	t.Helper()
	path := filepath.Clean(relativePath)
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return content
}
