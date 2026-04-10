package bandcamp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			require.Len(t, candidates, tt.wantCount)

			ranked := rankSearchCandidates(tt.source, candidates)
			require.Len(t, ranked, tt.wantCount)
			for i, want := range tt.wantTitles {
				assert.Equal(t, want, ranked[i].Title)
			}
		})
	}
}

func TestExtractSearchCandidatesDeduplicatesURLs(t *testing.T) {
	body := mustReadBandcampFixture(t, "testdata/search-fixture-url-dedup.html")
	candidates := extractSearchCandidates(body)
	require.Len(t, candidates, 1)
	assert.Equal(t, "https://artist.bandcamp.com/album/example-album", candidates[0].URL)
}

func TestExtractSongSearchCandidatesCanonicalizesAndDeduplicatesURLs(t *testing.T) {
	body := mustReadBandcampFixture(t, "testdata/search-fixture-track.html")
	candidates := extractSongSearchCandidates(body)
	require.Len(t, candidates, 2)

	assert.Equal(t, "Come Together", candidates[0].Title)
	assert.Equal(t, "https://comradiation.bandcamp.com/track/come-together", candidates[0].URL)

	assert.Equal(t, "Something", candidates[1].Title)
	assert.Equal(t, "https://comradiation.bandcamp.com/track/something", candidates[1].URL)
}

func mustReadBandcampFixture(t *testing.T, relativePath string) []byte {
	t.Helper()
	path := filepath.Clean(relativePath)
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return content
}
