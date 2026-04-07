package youtubemusic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/xmbshwll/ariadne/internal/model"
)

func TestAdapter(t *testing.T) {
	sourcePage := mustReadYouTubeMusicFixture(t, filepath.Join("testdata", "source-page.html"))
	searchPage := mustReadYouTubeMusicFixture(t, filepath.Join("testdata", "search-page.html"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/browse/MPREb_tQfaWH32ovE":
			_, _ = w.Write(sourcePage)
		case "/playlist":
			_, _ = w.Write(sourcePage)
		case "/search":
			_, _ = w.Write(searchPage)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := New(server.Client(), WithBaseURL(server.URL))

	t.Run("fetch album from browse page", func(t *testing.T) {
		album, err := adapter.FetchAlbum(context.Background(), model.ParsedAlbumURL{
			Service:      model.ServiceYouTubeMusic,
			EntityType:   "album",
			ID:           "MPREb_tQfaWH32ovE",
			CanonicalURL: server.URL + "/browse/MPREb_tQfaWH32ovE",
		})
		if err != nil {
			t.Fatalf("FetchAlbum error: %v", err)
		}
		if album.Title != "Abbey Road (Super Deluxe Edition)" {
			t.Fatalf("title = %q", album.Title)
		}
		if album.SourceURL != "https://music.youtube.com/playlist?list=OLAK5uy_lqcFZTOPHGwcnP0nYMzNuY0IES0fl7Fe4" {
			t.Fatalf("source url = %q", album.SourceURL)
		}
		if album.SourceID != "OLAK5uy_lqcFZTOPHGwcnP0nYMzNuY0IES0fl7Fe4" {
			t.Fatalf("source id = %q", album.SourceID)
		}
		if len(album.Artists) != 1 || album.Artists[0] != "The Beatles" {
			t.Fatalf("artists = %#v", album.Artists)
		}
		if album.TrackCount == 0 {
			t.Fatalf("expected track count")
		}
		if album.Tracks[0].Title != "Come Together (2019 Mix)" {
			t.Fatalf("first track title = %q", album.Tracks[0].Title)
		}
		if album.ArtworkURL == "" {
			t.Fatalf("expected artwork url")
		}
	})

	t.Run("search by metadata hydrates browse result", func(t *testing.T) {
		results, err := adapter.SearchByMetadata(context.Background(), model.CanonicalAlbum{
			Title:   "Abbey Road",
			Artists: []string{"The Beatles"},
		})
		if err != nil {
			t.Fatalf("SearchByMetadata error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("result count = %d, want 1", len(results))
		}
		if results[0].CandidateID != "OLAK5uy_lqcFZTOPHGwcnP0nYMzNuY0IES0fl7Fe4" {
			t.Fatalf("candidate id = %q", results[0].CandidateID)
		}
		if results[0].Title != "Abbey Road (Super Deluxe Edition)" {
			t.Fatalf("candidate title = %q", results[0].Title)
		}
		if len(results[0].Tracks) == 0 {
			t.Fatalf("expected hydrated track list")
		}
	})

	t.Run("identifier search unsupported", func(t *testing.T) {
		upcResults, err := adapter.SearchByUPC(context.Background(), "123")
		if err != nil {
			t.Fatalf("SearchByUPC error: %v", err)
		}
		if len(upcResults) != 0 {
			t.Fatalf("upc results = %d, want 0", len(upcResults))
		}
		isrcResults, err := adapter.SearchByISRC(context.Background(), []string{"ABC"})
		if err != nil {
			t.Fatalf("SearchByISRC error: %v", err)
		}
		if len(isrcResults) != 0 {
			t.Fatalf("isrc results = %d, want 0", len(isrcResults))
		}
	})
}

func mustReadYouTubeMusicFixture(t *testing.T, relativePath string) []byte {
	t.Helper()
	path := filepath.Clean(relativePath)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return content
}
