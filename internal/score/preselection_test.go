package score_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xmbshwll/ariadne/internal/score"
)

var preSelectionMarkers = []string{"super deluxe", "remastered", "remix", "mix", "deluxe", "live"}

func TestPreSelectionTitleRank(t *testing.T) {
	tests := []struct {
		name           string
		sourceTitle    string
		candidateTitle string
		want           int
	}{
		{name: "exact normalized match", sourceTitle: "Abbey Road", candidateTitle: "abbey road", want: score.PreSelectionTitleExact},
		{name: "edition-stripped match", sourceTitle: "Abbey Road", candidateTitle: "Abbey Road (Remastered)", want: score.PreSelectionTitleCore},
		{name: "substring containment", sourceTitle: "Abbey Road", candidateTitle: "The Beatles Abbey Road Live", want: score.PreSelectionTitlePart},
		{name: "no match", sourceTitle: "Abbey Road", candidateTitle: "Let It Be", want: 0},
		{name: "empty candidate", sourceTitle: "Abbey Road", candidateTitle: "", want: score.PreSelectionTitlePart},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, score.PreSelectionTitleRank(tt.sourceTitle, tt.candidateTitle, preSelectionMarkers), tt.name)
		})
	}
}

func TestPreSelectionCoreTitleStripsMarkersAsWholeWords(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "strips a marked suffix", value: "abbey road remastered", want: "abbey road"},
		{name: "strips a marked phrase", value: "abbey road super deluxe", want: "abbey road"},
		{name: "keeps words containing a marker", value: "livewire", want: "livewire"},
		{name: "keeps unlisted markers", value: "abbey road anniversary", want: "abbey road anniversary"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, score.PreSelectionCoreTitle(tt.value, preSelectionMarkers), tt.name)
		})
	}
}

func TestPreSelectionArtistRank(t *testing.T) {
	tests := []struct {
		name            string
		sourceArtists   []string
		candidateArtist string
		want            int
	}{
		{name: "exact match", sourceArtists: []string{"The Beatles"}, candidateArtist: "the beatles", want: score.PreSelectionArtistExact},
		{name: "contained match", sourceArtists: []string{"The Beatles"}, candidateArtist: "The Beatles Band", want: score.PreSelectionArtistContained},
		{name: "no match", sourceArtists: []string{"The Beatles"}, candidateArtist: "Tribute Band", want: 0},
		{name: "no source artist", sourceArtists: nil, candidateArtist: "The Beatles", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, score.PreSelectionArtistRank(tt.sourceArtists, tt.candidateArtist), tt.name)
		})
	}
}

func TestPreSelectionReleaseYearRank(t *testing.T) {
	tests := []struct {
		name       string
		sourceDate string
		candidate  string
		want       int
	}{
		{name: "same year", sourceDate: "1969-09-26", candidate: "1969-10-01", want: score.PreSelectionReleaseYearMatch},
		{name: "different year", sourceDate: "1969-09-26", candidate: "2020-01-01", want: 0},
		{name: "missing source date", sourceDate: "", candidate: "1969-09-26", want: 0},
		{name: "a bare year counts as a year match", sourceDate: "1969-09-26", candidate: "1969", want: score.PreSelectionReleaseYearMatch},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, score.PreSelectionReleaseYearRank(tt.sourceDate, tt.candidate), tt.name)
		})
	}
}
