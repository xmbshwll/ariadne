package score

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScoreTitleSignal(t *testing.T) {
	weights := titleSignalWeights{Exact: 25, Core: 15}

	tests := []struct {
		name          string
		title         string
		coreTitle     string
		candidate     string
		candidateCore string
		wantValue     int
		wantReason    string
	}{
		{name: "exact match", title: "Abbey Road", candidate: "Abbey Road", wantValue: 25, wantReason: "title exact match"},
		{name: "core title match", title: "Abbey Road (Live)", candidate: "Abbey Road", wantValue: 15, wantReason: "core title match"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scoreTitleSignal(tt.title, tt.coreTitle, tt.candidate, tt.candidateCore, weights)
			assert.Equal(t, tt.wantValue, got.Value)
			assert.Equal(t, tt.wantReason, got.Reason)
			assert.True(t, got.Evidence.Title)
		})
	}
}

func TestScoreArtistSignal(t *testing.T) {
	weights := artistSignalWeights{PrimaryExact: 20, Overlap: 10}

	tests := []struct {
		name                string
		sourceArtists       []string
		sourceNormalized    []string
		candidateArtists    []string
		candidateNormalized []string
		wantValue           int
		wantReason          string
	}{
		{name: "primary artist exact match", sourceArtists: []string{"The Beatles"}, candidateArtists: []string{"the beatles"}, wantValue: 20, wantReason: "primary artist exact match"},
		{name: "artist overlap", sourceArtists: []string{"Paul McCartney", "The Beatles"}, candidateArtists: []string{"The Beatles"}, wantValue: 10, wantReason: "artist overlap"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scoreArtistSignal(tt.sourceArtists, tt.sourceNormalized, tt.candidateArtists, tt.candidateNormalized, weights)
			assert.Equal(t, tt.wantValue, got.Value)
			assert.Equal(t, tt.wantReason, got.Reason)
			assert.True(t, got.Evidence.Artist)
		})
	}
}

func TestSharedScoreSignalsPreserveReasonText(t *testing.T) {
	assert.Equal(t, "release date exact match", scoreReleaseDateSignal("2024-01-01", "2024-01-01", releaseDateSignalWeights{Exact: 10, Year: 5}).Reason)
	assert.Equal(t, "release year match", scoreReleaseDateSignal("2024-01-01", "2024-12-31", releaseDateSignalWeights{Exact: 10, Year: 5}).Reason)
	assert.Equal(t, "duration near match", scoreDurationSignal(100_000, 101_000, 10).Reason)
	assert.Equal(t, "explicit mismatch", scoreExplicitSignal(true, false, -10).Reason)
	assert.Equal(t, "edition mismatch", scoreEditionHintSignal([]string{"deluxe"}, []string{"standard"}, -20).Reason)
	assert.Equal(t, "album title exact match", scoreNormalizedExactSignal("Album", "", "Album", "", 5, "album title exact match").Reason)
}
