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
		if err != nil {
			t.Fatalf("FetchAlbum error: %v", err)
		}
		if album.Title != "Abbey Road (Remastered)" {
			t.Fatalf("title = %q", album.Title)
		}
		if album.Label == "" {
			t.Fatalf("expected label")
		}
		if album.TrackCount != 17 {
			t.Fatalf("track count = %d", album.TrackCount)
		}
		if len(album.Tracks) != 17 {
			t.Fatalf("tracks len = %d", len(album.Tracks))
		}
		if album.Tracks[0].Title != "Come Together - Remastered 2009" {
			t.Fatalf("first track title = %q", album.Tracks[0].Title)
		}
		if album.Tracks[0].DurationMS != 259946 {
			t.Fatalf("first track duration = %d", album.Tracks[0].DurationMS)
		}
		if album.ArtworkURL == "" {
			t.Fatalf("expected artwork url")
		}
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
			writeJSON(t, w, apiTrack{ID: "track-1", Name: "Come Together", TrackNumber: 1, DiscNumber: 1, DurationMS: 258947, ExternalIDs: apiExternalIDs{ISRC: "GBAYE0601690"}, Artists: []apiArtist{{Name: "The Beatles"}}})
		})
		mux.HandleFunc("/tracks/track-2", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, apiTrack{ID: "track-2", Name: "Something", TrackNumber: 2, DiscNumber: 1, DurationMS: 182293, ExternalIDs: apiExternalIDs{ISRC: "GBAYE0601691"}, Artists: []apiArtist{{Name: "The Beatles"}}})
		})
		mux.HandleFunc("/tracks/track-weak-1", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, apiTrack{ID: "track-weak-1", Name: "Come Together", TrackNumber: 1, DiscNumber: 1, DurationMS: 200000, ExternalIDs: apiExternalIDs{ISRC: "OTHER0001"}, Artists: []apiArtist{{Name: "The Beatles Complete On Ukulele"}}})
		})
		mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query().Get("q")
			switch {
			case strings.Contains(query, "upc:602547670342"):
				writeJSON(t, w, apiAlbumSearchResponse{Albums: apiAlbumSearchPage{Items: []apiAlbumSummary{{ID: "album-good"}}}})
			case strings.Contains(query, "isrc:GBAYE0601690"):
				writeJSON(t, w, apiTrackSearchResponse{Tracks: apiTrackSearchPage{Items: []apiTrackSearchItem{{ID: "track-1", Album: apiTrackAlbum{ID: "album-good"}}}}})
			case strings.Contains(query, "isrc:GBAYE0601691"):
				writeJSON(t, w, apiTrackSearchResponse{Tracks: apiTrackSearchPage{Items: []apiTrackSearchItem{{ID: "track-2", Album: apiTrackAlbum{ID: "album-good"}}}}})
			case strings.Contains(query, "album:Abbey Road (Remastered)"), strings.Contains(query, "album:Abbey Road artist:The Beatles"), strings.Contains(query, "album:Abbey Road"):
				writeJSON(t, w, apiAlbumSearchResponse{Albums: apiAlbumSearchPage{Items: []apiAlbumSummary{{ID: "album-good"}, {ID: "album-weak"}}}})
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
		if err != nil {
			t.Fatalf("FetchAlbum api error: %v", err)
		}
		if album.UPC != "602547670342" {
			t.Fatalf("upc = %q", album.UPC)
		}
		if album.Tracks[0].ISRC != "GBAYE0601690" {
			t.Fatalf("first track isrc = %q", album.Tracks[0].ISRC)
		}

		upcResults, err := adapter.SearchByUPC(context.Background(), "602547670342")
		if err != nil {
			t.Fatalf("SearchByUPC error: %v", err)
		}
		assertSingleAlbum(t, upcResults, "album-good")

		isrcResults, err := adapter.SearchByISRC(context.Background(), []string{"GBAYE0601690", "GBAYE0601691"})
		if err != nil {
			t.Fatalf("SearchByISRC error: %v", err)
		}
		assertSingleAlbum(t, isrcResults, "album-good")

		metadataResults, err := adapter.SearchByMetadata(context.Background(), model.CanonicalAlbum{Title: "Abbey Road (Remastered)", Artists: []string{"The Beatles"}})
		if err != nil {
			t.Fatalf("SearchByMetadata error: %v", err)
		}
		if len(metadataResults) != 2 {
			t.Fatalf("metadata result count = %d, want 2", len(metadataResults))
		}
		if metadataResults[0].CandidateID != "album-good" {
			t.Fatalf("first metadata candidate = %q", metadataResults[0].CandidateID)
		}
	})
}

func assertSingleAlbum(t *testing.T, candidates []model.CandidateAlbum, wantID string) {
	t.Helper()
	if len(candidates) != 1 {
		t.Fatalf("candidate count = %d, want 1", len(candidates))
	}
	if candidates[0].CandidateID != wantID {
		t.Fatalf("candidate id = %q, want %q", candidates[0].CandidateID, wantID)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("encode json response: %v", err)
	}
}

func mustReadTestFile(t *testing.T, relativePath string) []byte {
	t.Helper()
	path := filepath.Clean(relativePath)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return content
}
