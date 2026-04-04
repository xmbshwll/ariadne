package bandcamp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xmbshwll/ariadne/internal/model"
)

func TestExtractAndRankSearchCandidates(t *testing.T) {
	tests := []struct {
		name       string
		fixture    string
		source     model.CanonicalAlbum
		wantCount  int
		wantTitles []string
	}{
		{
			name:    "remaster beats deluxe and unrelated album",
			fixture: "testdata/search-fixture-remaster-vs-deluxe.html",
			source: model.CanonicalAlbum{
				Title:      "Abbey Road (Remastered)",
				Artists:    []string{"The Beatles"},
				TrackCount: 17,
			},
			wantCount:  3,
			wantTitles: []string{"Abbey Road (Remaster)", "Abbey Road (Super Deluxe Edition)", "Revolver"},
		},
		{
			name:    "artist disambiguation still keeps exact title candidates ahead of looser variants",
			fixture: "testdata/search-fixture-artist-disambiguation.html",
			source: model.CanonicalAlbum{
				Title:      "The Abbey Road Session",
				Artists:    []string{"COMRADIATION"},
				TrackCount: 14,
			},
			wantCount:  3,
			wantTitles: []string{"The Abbey Road Session", "The Abbey Road Session", "Abbey Road Live"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := mustReadBandcampFixture(t, tt.fixture)
			candidates := extractSearchCandidates(body)
			if len(candidates) != tt.wantCount {
				t.Fatalf("candidate count = %d, want %d", len(candidates), tt.wantCount)
			}

			ranked := rankSearchCandidates(tt.source, candidates)
			if len(ranked) != tt.wantCount {
				t.Fatalf("ranked count = %d, want %d", len(ranked), tt.wantCount)
			}
			for i, want := range tt.wantTitles {
				if ranked[i].Title != want {
					t.Fatalf("ranked[%d] title = %q, want %q", i, ranked[i].Title, want)
				}
			}
		})
	}
}

func TestExtractSearchCandidatesDeduplicatesURLs(t *testing.T) {
	body := mustReadBandcampFixture(t, "testdata/search-fixture-url-dedup.html")
	candidates := extractSearchCandidates(body)
	if len(candidates) != 1 {
		t.Fatalf("candidate count = %d, want 1", len(candidates))
	}
	if candidates[0].URL != "https://artist.bandcamp.com/album/example-album" {
		t.Fatalf("candidate url = %q, want canonical bandcamp url", candidates[0].URL)
	}
}

func mustReadBandcampFixture(t *testing.T, relativePath string) []byte {
	t.Helper()
	path := filepath.Clean(relativePath)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return content
}
