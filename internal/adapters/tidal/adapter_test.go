package tidal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xmbshwll/ariadne/internal/model"
)

func TestAdapter(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth2/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		_ = json.NewEncoder(w).Encode(tokenResponse{AccessToken: "token-123", TokenType: "Bearer", ExpiresIn: 3600})
	})
	mux.HandleFunc("/albums/156205493", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, apiDocument{
			Data: apiResource{
				ID:   "156205493",
				Type: "albums",
				Attributes: resourceAttributes{
					Title:         "Shadows among trees",
					BarcodeID:     "053000502692",
					ReleaseDate:   "2020-10-02",
					Duration:      "PT35M",
					Explicit:      false,
					NumberOfItems: 5,
					Copyright:     resourceCopyright{Text: "Posev"},
				},
				Relationships: resourceRelationships{
					Artists:  relationship{Data: []relationshipData{{ID: "4152940", Type: "artists"}}},
					CoverArt: relationship{Data: []relationshipData{{ID: "art-1", Type: "artworks"}}},
					Items:    relationship{Data: []relationshipData{{ID: "156205494", Type: "tracks", Meta: relationshipMeta{TrackNumber: 1, VolumeNumber: 1}}, {ID: "156205495", Type: "tracks", Meta: relationshipMeta{TrackNumber: 2, VolumeNumber: 1}}}},
				},
			},
			Included: []apiResource{
				{ID: "4152940", Type: "artists", Attributes: resourceAttributes{Name: "Fetch"}},
				{ID: "art-1", Type: "artworks", Attributes: resourceAttributes{Files: []resourceFile{{Href: "https://resources.tidal.test/1280.jpg", Meta: fileMeta{Width: 1280, Height: 1280}}}}},
				{ID: "156205494", Type: "tracks", Attributes: resourceAttributes{Title: "Kings of mist", Duration: "PT6M30S", ISRC: "QZMHK2043414"}, Relationships: resourceRelationships{Artists: relationship{Data: []relationshipData{{ID: "4152940", Type: "artists"}}}}},
				{ID: "156205495", Type: "tracks", Attributes: resourceAttributes{Title: "Something unspeakable", Duration: "PT7M00S", ISRC: "QZMHK2043415"}, Relationships: resourceRelationships{Artists: relationship{Data: []relationshipData{{ID: "4152940", Type: "artists"}}}}},
			},
		})
	})
	mux.HandleFunc("/albums", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("filter[barcodeId]") != "053000502692" {
			http.NotFound(w, r)
			return
		}
		writeJSON(t, w, apiDocument{Data: []apiResource{{ID: "156205493", Type: "albums"}}})
	})
	mux.HandleFunc("/tracks", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("filter[isrc]") != "QZMHK2043414" {
			http.NotFound(w, r)
			return
		}
		writeJSON(t, w, apiDocument{
			Data:     []apiResource{{ID: "156205494", Type: "tracks"}},
			Included: []apiResource{{ID: "156205493", Type: "albums"}},
		})
	})
	mux.HandleFunc("/searchResults/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/searchResults/Shadows among trees Fetch/relationships/albums" && r.URL.Path != "/searchResults/Shadows%20among%20trees%20Fetch/relationships/albums" {
			http.NotFound(w, r)
			return
		}
		writeJSON(t, w, apiDocument{Data: []apiResource{{ID: "156205493", Type: "albums"}}})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	adapter := New(
		server.Client(),
		WithCredentials("tidal-client", "tidal-secret"),
		WithAPIBaseURL(server.URL),
		WithAuthBaseURL(server.URL),
	)

	parsed := model.ParsedAlbumURL{Service: model.ServiceTIDAL, EntityType: "album", ID: "156205493", CanonicalURL: "https://tidal.com/album/156205493"}
	album, err := adapter.FetchAlbum(context.Background(), parsed)
	if err != nil {
		t.Fatalf("FetchAlbum error: %v", err)
	}
	if album.Title != "Shadows among trees" {
		t.Fatalf("title = %q", album.Title)
	}
	if album.UPC != "053000502692" {
		t.Fatalf("upc = %q", album.UPC)
	}
	if len(album.Tracks) != 2 {
		t.Fatalf("tracks len = %d, want 2", len(album.Tracks))
	}
	if album.Tracks[0].ISRC != "QZMHK2043414" {
		t.Fatalf("first track isrc = %q", album.Tracks[0].ISRC)
	}
	if album.ArtworkURL == "" {
		t.Fatalf("expected artwork url")
	}

	upcResults, err := adapter.SearchByUPC(context.Background(), "053000502692")
	if err != nil {
		t.Fatalf("SearchByUPC error: %v", err)
	}
	assertSingleAlbum(t, upcResults, "156205493")

	isrcResults, err := adapter.SearchByISRC(context.Background(), []string{"QZMHK2043414"})
	if err != nil {
		t.Fatalf("SearchByISRC error: %v", err)
	}
	assertSingleAlbum(t, isrcResults, "156205493")

	metadataResults, err := adapter.SearchByMetadata(context.Background(), model.CanonicalAlbum{Title: "Shadows among trees", Artists: []string{"Fetch"}})
	if err != nil {
		t.Fatalf("SearchByMetadata error: %v", err)
	}
	assertSingleAlbum(t, metadataResults, "156205493")
}

func TestAdapterRequiresCredentialsForSourceAndSearch(t *testing.T) {
	adapter := New(nil)

	if _, err := adapter.FetchAlbum(context.Background(), model.ParsedAlbumURL{Service: model.ServiceTIDAL, ID: "156205493", CanonicalURL: "https://tidal.com/album/156205493"}); err == nil {
		t.Fatalf("expected credentials error for source fetch")
	}
	if _, err := adapter.SearchByMetadata(context.Background(), model.CanonicalAlbum{Title: "Album"}); err == nil {
		t.Fatalf("expected credentials error for metadata search")
	}
}

func assertSingleAlbum(t *testing.T, candidates []model.CandidateAlbum, wantID string) {
	t.Helper()
	if len(candidates) != 1 {
		t.Fatalf("candidate count = %d, want 1", len(candidates))
	}
	if candidates[0].CandidateID != wantID {
		t.Fatalf("candidate id = %q, want %q", candidates[0].CandidateID, wantID)
	}
	if !strings.Contains(candidates[0].MatchURL, wantID) {
		t.Fatalf("candidate url = %q, want id %q in url", candidates[0].MatchURL, wantID)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("encode json response: %v", err)
	}
}
