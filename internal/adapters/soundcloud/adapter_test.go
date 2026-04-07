package soundcloud

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/xmbshwll/ariadne/internal/model"
)

func TestAdapter(t *testing.T) {
	sourcePayload := mustReadSoundCloudFixture(t, filepath.Join("testdata", "source-payload.json"))
	searchPayload := mustReadSoundCloudFixture(t, filepath.Join("testdata", "search-results.json"))
	clientID := "qNxp6KCjufkNWMIclTv0O4ycYGY0eFFX"

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			_, _ = fmt.Fprintf(w, `<html><body><script src="%s/assets/app.js"></script></body></html>`, server.URL)
		case "/assets/app.js":
			_, _ = w.Write([]byte(`window.__sc_config={client_id:"` + clientID + `"};`))
		case "/album":
			_, _ = fmt.Fprintf(w, `<html><body><script>window.__sc_hydration = [{"hydratable":"playlist","data":%s}];</script></body></html>`, sourcePayload)
		case "/search/playlists":
			if r.URL.Query().Get("client_id") != clientID {
				http.Error(w, "missing client id", http.StatusUnauthorized)
				return
			}
			_, _ = w.Write(searchPayload)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := New(server.Client(), WithSiteBaseURL(server.URL), WithAPIBaseURL(server.URL))
	parsed := model.ParsedAlbumURL{
		Service:      model.ServiceSoundCloud,
		EntityType:   "set",
		ID:           "evidence-official/sets/cats-dogs-6",
		CanonicalURL: server.URL + "/album",
		RawURL:       server.URL + "/album",
	}

	t.Run("fetch album from hydration", func(t *testing.T) {
		album, err := adapter.FetchAlbum(context.Background(), parsed)
		if err != nil {
			t.Fatalf("FetchAlbum error: %v", err)
		}
		if album.Title != "Cats & Dogs" {
			t.Fatalf("title = %q", album.Title)
		}
		if album.SourceURL != "https://soundcloud.com/evidence-official/sets/cats-dogs-6" {
			t.Fatalf("source url = %q", album.SourceURL)
		}
		if album.TrackCount != 17 {
			t.Fatalf("track count = %d, want 17", album.TrackCount)
		}
		if album.UPC != "826257014467" {
			t.Fatalf("upc = %q, want 826257014467", album.UPC)
		}
		if len(album.Tracks) == 0 {
			t.Fatalf("expected tracks")
		}
		if album.Tracks[0].ISRC != "USBWK1100093" {
			t.Fatalf("first track isrc = %q", album.Tracks[0].ISRC)
		}
		if album.Label != "Rhymesayers" {
			t.Fatalf("label = %q, want Rhymesayers", album.Label)
		}
	})

	t.Run("search by metadata via api v2", func(t *testing.T) {
		results, err := adapter.SearchByMetadata(context.Background(), model.CanonicalAlbum{
			Title:   "Cats & Dogs",
			Artists: []string{"Evidence"},
		})
		if err != nil {
			t.Fatalf("SearchByMetadata error: %v", err)
		}
		if len(results) != 5 {
			t.Fatalf("result count = %d, want 5", len(results))
		}
		if results[0].CandidateID != "evidence-official/sets/cats-dogs-3" {
			t.Fatalf("first candidate id = %q", results[0].CandidateID)
		}
		if results[1].CandidateID != "evidence-official/sets/cats-dogs-6" {
			t.Fatalf("second candidate id = %q", results[1].CandidateID)
		}
		if results[1].Tracks[0].ISRC != "USBWK1100093" {
			t.Fatalf("second candidate first isrc = %q", results[1].Tracks[0].ISRC)
		}
	})

	t.Run("identifier search unsupported", func(t *testing.T) {
		upcResults, err := adapter.SearchByUPC(context.Background(), "826257014467")
		if err != nil {
			t.Fatalf("SearchByUPC error: %v", err)
		}
		if len(upcResults) != 0 {
			t.Fatalf("upc results = %d, want 0", len(upcResults))
		}
		isrcResults, err := adapter.SearchByISRC(context.Background(), []string{"USBWK1100093"})
		if err != nil {
			t.Fatalf("SearchByISRC error: %v", err)
		}
		if len(isrcResults) != 0 {
			t.Fatalf("isrc results = %d, want 0", len(isrcResults))
		}
	})
}

func mustReadSoundCloudFixture(t *testing.T, relativePath string) []byte {
	t.Helper()
	path := filepath.Clean(relativePath)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return content
}
