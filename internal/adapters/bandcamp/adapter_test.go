package bandcamp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xmbshwll/ariadne/internal/model"
)

const (
	bandcampSearchPath = "/search"
	lonAbatyAbbeyRoad  = "Lôn Abaty / Abbey Road"
)

func TestAdapter(t *testing.T) {
	sourcePage := mustReadTestFile(t, "testdata/source-page.html")

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/album/l-n-abaty-abbey-road":
			_, _ = w.Write(sourcePage)
		case bandcampSearchPath:
			searchHTML := fmt.Sprintf(`
				<html><body>
					<li class="searchresult data-search">
					  <div class="itemtype">ALBUM</div>
					  <div class="heading"><a href="%s/album/l-n-abaty-abbey-road?from=search">Lôn Abaty / Abbey Road</a></div>
					  <div class="subhead">by COMRADIATION</div>
					  <div class="length">14 tracks, 60 minutes</div>
					  <div class="released">released December 2, 2021</div>
					</li>
					<li class="searchresult data-search">
					  <div class="itemtype">ALBUM</div>
					  <div class="heading"><a href="%s/album/after-abbey-road">After Abbey Road</a></div>
					  <div class="subhead">by Mike Westbrook</div>
					  <div class="length">17 tracks, 94 minutes</div>
					  <div class="released">released September 27, 2019</div>
					</li>
				</body></html>
			`, server.URL, server.URL)
			_, _ = w.Write([]byte(searchHTML))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := New(server.Client(), WithSearchBaseURL(server.URL))
	parsed := model.ParsedAlbumURL{
		Service:      model.ServiceBandcamp,
		EntityType:   "album",
		ID:           "l-n-abaty-abbey-road",
		CanonicalURL: server.URL + "/album/l-n-abaty-abbey-road",
		RawURL:       server.URL + "/album/l-n-abaty-abbey-road",
	}

	t.Run("fetch album", func(t *testing.T) {
		album, err := adapter.FetchAlbum(context.Background(), parsed)
		if err != nil {
			t.Fatalf("FetchAlbum error: %v", err)
		}
		if album.Title != lonAbatyAbbeyRoad {
			t.Fatalf("title = %q", album.Title)
		}
		if album.SourceID != "l-n-abaty-abbey-road" {
			t.Fatalf("source id = %q", album.SourceID)
		}
		if album.SourceURL != parsed.CanonicalURL {
			t.Fatalf("source url = %q", album.SourceURL)
		}
		if album.TrackCount != 14 {
			t.Fatalf("track count = %d", album.TrackCount)
		}
		if len(album.Tracks) != 14 {
			t.Fatalf("tracks len = %d", len(album.Tracks))
		}
		if album.Tracks[0].Title == "" {
			t.Fatalf("expected first track title")
		}
		if album.TotalDurationMS <= 0 {
			t.Fatalf("expected total duration ms > 0")
		}
		if album.ArtworkURL == "" {
			t.Fatalf("expected artwork url")
		}
		if album.ReleaseDate != "2021-12-02" {
			t.Fatalf("release date = %q, want 2021-12-02", album.ReleaseDate)
		}
	})

	t.Run("search by metadata", func(t *testing.T) {
		results, err := adapter.SearchByMetadata(context.Background(), model.CanonicalAlbum{
			Title:   "Abbey Road",
			Artists: []string{"COMRADIATION"},
		})
		if err != nil {
			t.Fatalf("SearchByMetadata error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("result count = %d, want 1", len(results))
		}
		if results[0].CandidateID != "l-n-abaty-abbey-road" {
			t.Fatalf("candidate id = %q", results[0].CandidateID)
		}
		if !strings.Contains(results[0].MatchURL, "/album/l-n-abaty-abbey-road") {
			t.Fatalf("candidate url = %q", results[0].MatchURL)
		}
	})

	t.Run("search by upc unsupported", func(t *testing.T) {
		results, err := adapter.SearchByUPC(context.Background(), "123")
		if err != nil {
			t.Fatalf("SearchByUPC error: %v", err)
		}
		if len(results) != 0 {
			t.Fatalf("result count = %d, want 0", len(results))
		}
	})
}

func TestSearchByMetadataReranksHydratedCandidates(t *testing.T) {
	source := model.CanonicalAlbum{
		Title:      "Live at KEXP",
		Artists:    []string{"Sea Lemon"},
		TrackCount: 4,
		Tracks: []model.CanonicalTrack{
			{Title: "Stay", NormalizedTitle: "stay"},
			{Title: "Cellar", NormalizedTitle: "cellar"},
			{Title: "Vaporized", NormalizedTitle: "vaporized"},
			{Title: "Give In", NormalizedTitle: "give in"},
		},
	}

	lowOverlapPage := mustBandcampAlbumPage(t, "Live at KEXP", "Sea Lemon", "2024-01-10", []string{"Stay", "Blue Moon", "Drive", "Night Swim"})
	highOverlapPage := mustBandcampAlbumPage(t, "Live at KEXP", "Sea Lemon", "2024-01-10", []string{"Stay", "Cellar", "Vaporized", "Give In"})

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case bandcampSearchPath:
			searchHTML := fmt.Sprintf(`
				<html><body>
					<li class="searchresult data-search">
					  <div class="itemtype">ALBUM</div>
					  <div class="heading"><a href="%s/album/live-at-kexp-low?from=search">Live at KEXP</a></div>
					  <div class="subhead">by Sea Lemon</div>
					  <div class="length">4 tracks, 12 minutes</div>
					  <div class="released">released January 10, 2024</div>
					</li>
					<li class="searchresult data-search">
					  <div class="itemtype">ALBUM</div>
					  <div class="heading"><a href="%s/album/live-at-kexp-high?from=search">Live at KEXP</a></div>
					  <div class="subhead">by Sea Lemon</div>
					  <div class="length">4 tracks, 12 minutes</div>
					  <div class="released">released January 10, 2024</div>
					</li>
				</body></html>
			`, server.URL, server.URL)
			_, _ = w.Write([]byte(searchHTML))
		case "/album/live-at-kexp-low":
			_, _ = w.Write(lowOverlapPage)
		case "/album/live-at-kexp-high":
			_, _ = w.Write(highOverlapPage)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := New(server.Client(), WithSearchBaseURL(server.URL))
	results, err := adapter.SearchByMetadata(context.Background(), source)
	if err != nil {
		t.Fatalf("SearchByMetadata error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("result count = %d, want 2", len(results))
	}
	if results[0].CandidateID != "live-at-kexp-high" {
		t.Fatalf("first candidate = %q, want live-at-kexp-high", results[0].CandidateID)
	}
	if results[1].CandidateID != "live-at-kexp-low" {
		t.Fatalf("second candidate = %q, want live-at-kexp-low", results[1].CandidateID)
	}
}

func TestSongAdapter(t *testing.T) {
	trackPage := mustBandcampTrackPage(t, "Come Together", "COMRADIATION", lonAbatyAbbeyRoad, "2021-12-02", 251000)

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/track/come-together":
			_, _ = w.Write(trackPage)
		case bandcampSearchPath:
			searchHTML := fmt.Sprintf(`
				<html><body>
					<li class="searchresult data-search">
					  <div class="itemtype">TRACK</div>
					  <div class="heading"><a href="%s/track/come-together?from=search">Come Together</a></div>
					  <div class="subhead">by COMRADIATION</div>
					  <div class="released">released December 2, 2021</div>
					</li>
					<li class="searchresult data-search">
					  <div class="itemtype">TRACK</div>
					  <div class="heading"><a href="%s/track/come-together-live">Come Together (Live)</a></div>
					  <div class="subhead">by Tribute Band</div>
					  <div class="released">released January 1, 2020</div>
					</li>
				</body></html>
			`, server.URL, server.URL)
			_, _ = w.Write([]byte(searchHTML))
		case "/track/come-together-live":
			_, _ = w.Write(mustBandcampTrackPage(t, "Come Together (Live)", "Tribute Band", "Abbey Road Live", "2020-01-01", 300000))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := New(server.Client(), WithSearchBaseURL(server.URL))
	parsed := model.ParsedAlbumURL{
		Service:      model.ServiceBandcamp,
		EntityType:   "song",
		ID:           "come-together",
		CanonicalURL: server.URL + "/track/come-together",
		RawURL:       server.URL + "/track/come-together",
	}

	t.Run("fetch song", func(t *testing.T) {
		song, err := adapter.FetchSong(context.Background(), parsed)
		if err != nil {
			t.Fatalf("FetchSong error: %v", err)
		}
		if song.Title != "Come Together" {
			t.Fatalf("title = %q", song.Title)
		}
		if song.AlbumTitle != lonAbatyAbbeyRoad {
			t.Fatalf("album title = %q", song.AlbumTitle)
		}
		if song.DurationMS != 251000 {
			t.Fatalf("duration = %d", song.DurationMS)
		}
	})

	t.Run("search song by metadata", func(t *testing.T) {
		results, err := adapter.SearchSongByMetadata(context.Background(), model.CanonicalSong{
			Title:      "Come Together",
			Artists:    []string{"COMRADIATION"},
			DurationMS: 251000,
			AlbumTitle: lonAbatyAbbeyRoad,
		})
		if err != nil {
			t.Fatalf("SearchSongByMetadata error: %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("result count = %d, want 2", len(results))
		}
		if results[0].CandidateID != "come-together" {
			t.Fatalf("candidate id = %q", results[0].CandidateID)
		}
		if results[0].AlbumTitle != lonAbatyAbbeyRoad {
			t.Fatalf("album title = %q", results[0].AlbumTitle)
		}
		if results[1].CandidateID != "come-together-live" {
			t.Fatalf("second candidate id = %q", results[1].CandidateID)
		}
	})
}

func TestRealSavedPages(t *testing.T) {
	tests := []struct {
		name        string
		fixture     string
		path        string
		wantTitle   string
		wantArtist  string
		wantTracks  int
		wantDate    string
		wantArtwork bool
	}{
		{
			name:        "after abbey road",
			fixture:     "testdata/real-after-abbey-road.html",
			path:        "/album/after-abbey-road",
			wantTitle:   "After Abbey Road",
			wantArtist:  "Mike Westbrook",
			wantTracks:  17,
			wantDate:    "2019-09-27",
			wantArtwork: true,
		},
		{
			name:        "morningrise abbey road remaster",
			fixture:     "testdata/real-morningrise-abbey-road-remaster.html",
			path:        "/album/morningrise-abbey-road-remaster",
			wantTitle:   "Morningrise (Abbey Road Remaster)",
			wantArtist:  "Opeth",
			wantTracks:  5,
			wantDate:    "2023-06-02",
			wantArtwork: true,
		},
		{
			name:        "for those that wish to exist at abbey road",
			fixture:     "testdata/real-for-those-that-wish-to-exist-at-abbey-road.html",
			path:        "/album/for-those-that-wish-to-exist-at-abbey-road",
			wantTitle:   "For Those That Wish To Exist At Abbey Road",
			wantArtist:  "Architects",
			wantTracks:  15,
			wantDate:    "2022-03-25",
			wantArtwork: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := mustReadTestFile(t, tt.fixture)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tt.path {
					http.NotFound(w, r)
					return
				}
				_, _ = w.Write(page)
			}))
			defer server.Close()

			adapter := New(server.Client())
			parsed := model.ParsedAlbumURL{
				Service:      model.ServiceBandcamp,
				EntityType:   "album",
				ID:           strings.TrimPrefix(tt.path, "/album/"),
				CanonicalURL: server.URL + tt.path,
				RawURL:       server.URL + tt.path,
			}

			album, err := adapter.FetchAlbum(context.Background(), parsed)
			if err != nil {
				t.Fatalf("FetchAlbum error: %v", err)
			}
			if album.Title != tt.wantTitle {
				t.Fatalf("title = %q, want %q", album.Title, tt.wantTitle)
			}
			if len(album.Artists) == 0 || album.Artists[0] != tt.wantArtist {
				t.Fatalf("artist = %v, want %q", album.Artists, tt.wantArtist)
			}
			if album.TrackCount != tt.wantTracks {
				t.Fatalf("track count = %d, want %d", album.TrackCount, tt.wantTracks)
			}
			if album.ReleaseDate != tt.wantDate {
				t.Fatalf("release date = %q, want %q", album.ReleaseDate, tt.wantDate)
			}
			if tt.wantArtwork && album.ArtworkURL == "" {
				t.Fatalf("expected artwork url")
			}
		})
	}
}

func mustBandcampAlbumPage(t *testing.T, title string, artist string, releaseDate string, tracks []string) []byte {
	t.Helper()
	items := make([]map[string]any, 0, len(tracks))
	for i, track := range tracks {
		items = append(items, map[string]any{
			"@type":    "ListItem",
			"position": i + 1,
			"item": map[string]any{
				"@type":    "MusicRecording",
				"name":     track,
				"duration": "P00H03M00S",
			},
		})
	}
	payload := map[string]any{
		"@context":      "https://schema.org",
		"@type":         "MusicAlbum",
		"@id":           "https://example.bandcamp.com/album/test",
		"name":          title,
		"datePublished": releaseDate + " 00:00:00 GMT",
		"image":         "https://f4.bcbits.com/img/example.jpg",
		"byArtist": map[string]any{
			"@type": "MusicGroup",
			"name":  artist,
		},
		"publisher": map[string]any{
			"@type": "MusicGroup",
			"name":  artist,
		},
		"track": map[string]any{
			"@type":           "ItemList",
			"numberOfItems":   len(tracks),
			"itemListElement": items,
		},
	}
	content, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal bandcamp album payload: %v", err)
	}
	page := fmt.Sprintf("<html><body><script type=\"application/ld+json\">%s</script></body></html>", content)
	return []byte(page)
}

func mustBandcampTrackPage(t *testing.T, title string, artist string, albumTitle string, releaseDate string, durationMS int) []byte {
	t.Helper()
	payload := map[string]any{
		"@context":      "https://schema.org",
		"@type":         "MusicRecording",
		"@id":           "https://example.bandcamp.com/track/test",
		"name":          title,
		"datePublished": releaseDate + " 00:00:00 GMT",
		"duration":      fmt.Sprintf("PT%dM%dS", durationMS/60000, (durationMS/1000)%60),
		"image":         "https://f4.bcbits.com/img/example-track.jpg",
		"byArtist": map[string]any{
			"@type": "MusicGroup",
			"name":  artist,
		},
		"publisher": map[string]any{
			"@type": "MusicGroup",
			"name":  artist,
		},
		"inAlbum": map[string]any{
			"@type": "MusicAlbum",
			"@id":   "https://example.bandcamp.com/album/example-album",
			"name":  albumTitle,
		},
	}
	content, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal bandcamp track payload: %v", err)
	}
	page := fmt.Sprintf("<html><body><script type=\"application/ld+json\">%s</script></body></html>", content)
	return []byte(page)
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
