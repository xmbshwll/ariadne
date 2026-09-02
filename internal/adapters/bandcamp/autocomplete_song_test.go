package bandcamp

// White-box: the song half of autocomplete extraction is unexported, and the
// export seam (BandcampTargetSearch) only reaches it through a live search. The
// extraction rules are worth pinning directly.

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractAutocompleteSongSearchCandidates(t *testing.T) {
	tests := []struct {
		name        string
		results     []fuzzySearchResult
		wantURLs    []string
		wantTitles  []string
		wantArtists []string
	}{
		{
			name: "track results become song candidates with canonical urls",
			results: []fuzzySearchResult{
				{Type: "t", Name: "Come Together", BandName: "The Beatles", URL: "https://artist.bandcamp.com/track/come-together"},
				{Type: "t", Name: "Something", BandName: "The Beatles", URL: "https://artist.bandcamp.com/track/something"},
			},
			wantURLs:    []string{"https://artist.bandcamp.com/track/come-together", "https://artist.bandcamp.com/track/something"},
			wantTitles:  []string{"Come Together", "Something"},
			wantArtists: []string{"The Beatles", "The Beatles"},
		},
		{
			name: "album, artist, and unknown result types are skipped",
			results: []fuzzySearchResult{
				{Type: "a", Name: "Abbey Road", BandName: "The Beatles", URL: "https://artist.bandcamp.com/album/abbey-road"},
				{Type: "b", Name: "The Beatles", BandName: "The Beatles", URL: "https://artist.bandcamp.com"},
				{Type: "t", Name: "Come Together", BandName: "The Beatles", URL: "https://artist.bandcamp.com/track/come-together"},
			},
			wantURLs:    []string{"https://artist.bandcamp.com/track/come-together"},
			wantTitles:  []string{"Come Together"},
			wantArtists: []string{"The Beatles"},
		},
		{
			name: "a track result on an unsupported host is dropped",
			results: []fuzzySearchResult{
				{Type: "t", Name: "Off Site", BandName: "Someone", URL: "https://example.com/track/off-site"},
			},
			wantURLs: nil,
		},
		{
			name:     "no results yields no candidates",
			results:  nil,
			wantURLs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidates := extractAutocompleteSongSearchCandidates(fuzzySearchResponse{Results: tt.results})

			var urls, titles, artists []string
			for _, candidate := range candidates {
				urls = append(urls, candidate.URL)
				titles = append(titles, candidate.Title)
				artists = append(artists, candidate.Artist)
			}
			assert.Equal(t, tt.wantURLs, urls, tt.name)
			assert.Equal(t, tt.wantTitles, titles, tt.name)
			assert.Equal(t, tt.wantArtists, artists, tt.name)
		})
	}
}
