package deezer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/xmbshwll/ariadne/internal/model"
)

func TestAdapter(t *testing.T) {
	albumBytes := mustReadTestFile(t, "testdata/source-payload.json")
	trackBytes := mustReadTestFile(t, "testdata/tracks.json")
	searchBytes := []byte(`{"data":[{"id":12047952,"title":"Abbey Road (Remastered)"}]}`)

	var album albumResponse
	if err := json.Unmarshal(albumBytes, &album); err != nil {
		t.Fatalf("unmarshal album: %v", err)
	}

	var tracks tracksResponse
	if err := json.Unmarshal(trackBytes, &tracks); err != nil {
		t.Fatalf("unmarshal tracks: %v", err)
	}

	t.Run("to canonical album", func(t *testing.T) {
		adapter := New(nil)
		parsed := model.ParsedAlbumURL{
			Service:      model.ServiceDeezer,
			EntityType:   "album",
			ID:           "12047952",
			CanonicalURL: "https://www.deezer.com/album/12047952",
			RawURL:       "https://www.deezer.com/album/12047952",
		}

		got := adapter.toCanonicalAlbum(parsed, album, tracks)
		if got.Title != "Abbey Road (Remastered)" {
			t.Fatalf("title = %q", got.Title)
		}
		if got.UPC != "602547670342" {
			t.Fatalf("upc = %q", got.UPC)
		}
		if got.Label != "EMI Catalogue" {
			t.Fatalf("label = %q", got.Label)
		}
		if got.TrackCount != 17 {
			t.Fatalf("track count = %d", got.TrackCount)
		}
		if len(got.Tracks) == 0 {
			t.Fatalf("expected tracks")
		}
		if got.Tracks[0].ISRC != "GBAYE0601690" {
			t.Fatalf("first track isrc = %q", got.Tracks[0].ISRC)
		}
		if got.Tracks[0].DurationMS != 258000 {
			t.Fatalf("first track duration = %d", got.Tracks[0].DurationMS)
		}
		if got.Artists[0] != "The Beatles" {
			t.Fatalf("artist = %q", got.Artists[0])
		}
	})

	t.Run("target search", func(t *testing.T) {
		server := newTestServer(t, albumBytes, trackBytes, searchBytes)
		defer server.Close()

		adapter := New(server.Client())
		adapter.baseURL = server.URL
		ctx := context.Background()

		t.Run("search by upc", func(t *testing.T) {
			results, err := adapter.SearchByUPC(ctx, "602547670342")
			if err != nil {
				t.Fatalf("SearchByUPC error: %v", err)
			}
			assertSingleCandidate(t, results)
		})

		t.Run("search by isrc", func(t *testing.T) {
			results, err := adapter.SearchByISRC(ctx, []string{"GBAYE0601690", "GBAYE0601691"})
			if err != nil {
				t.Fatalf("SearchByISRC error: %v", err)
			}
			assertSingleCandidate(t, results)
		})

		t.Run("search by metadata", func(t *testing.T) {
			results, err := adapter.SearchByMetadata(ctx, model.CanonicalAlbum{
				Title:   "Abbey Road (Remastered)",
				Artists: []string{"The Beatles"},
			})
			if err != nil {
				t.Fatalf("SearchByMetadata error: %v", err)
			}
			assertSingleCandidate(t, results)
		})
	})
}

func newTestServer(t *testing.T, albumBytes, trackBytes, searchBytes []byte) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/album/12047952":
			_, _ = w.Write(albumBytes)
		case "/album/upc:602547670342":
			_, _ = w.Write(albumBytes)
		case "/album/12047952/tracks":
			_, _ = w.Write(trackBytes)
		case "/search/album":
			_, _ = w.Write(searchBytes)
		case "/track/isrc:GBAYE0601690":
			_, _ = w.Write([]byte(`{"id":116348128,"title":"Come Together (Remastered 2009)","isrc":"GBAYE0601690","album":{"id":12047952,"title":"Abbey Road (Remastered)","link":"https://www.deezer.com/album/12047952"}}`))
		case "/track/isrc:GBAYE0601691":
			_, _ = w.Write([]byte(`{"id":116348454,"title":"Something (Remastered 2009)","isrc":"GBAYE0601691","album":{"id":12047952,"title":"Abbey Road (Remastered)","link":"https://www.deezer.com/album/12047952"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
}

func assertSingleCandidate(t *testing.T, results []model.CandidateAlbum) {
	t.Helper()
	if len(results) != 1 {
		t.Fatalf("result count = %d, want 1", len(results))
	}
	if results[0].CandidateID != "12047952" {
		t.Fatalf("candidate id = %q, want 12047952", results[0].CandidateID)
	}
	if results[0].MatchURL != "https://www.deezer.com/album/12047952" {
		t.Fatalf("match url = %q", results[0].MatchURL)
	}
	if results[0].UPC != "602547670342" {
		t.Fatalf("upc = %q", results[0].UPC)
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
