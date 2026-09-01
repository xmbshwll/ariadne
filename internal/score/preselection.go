package score

// PreSelection tier: the plain-int title/artist/release-year ranks a Music
// Service adapter uses to pick which wire candidates are worth hydrating
// before Entity Resolution ranks them with the weighted Score Signals. These
// are coarse tiers, not the final score; they exist so the title/artist
// matching rules are stated once for both layers.

import (
	"strings"

	"github.com/xmbshwll/ariadne/internal/normalize"
)

// PreSelectionTitleRank scores a wire candidate title against the source
// title: exact normalized match beats edition-stripped match beats substring
// containment.
const (
	PreSelectionTitleExact = 40
	PreSelectionTitleCore  = 25
	PreSelectionTitlePart  = 10
)

// PreSelectionArtistRank scores a wire candidate artist against the source
// artists.
const (
	PreSelectionArtistExact     = 45
	PreSelectionArtistContained = 20
)

// PreSelectionReleaseYearMatch is the rank a candidate release date earns when
// its year matches the source's.
const PreSelectionReleaseYearMatch = 5

// PreSelectionCoreTitle strips the given edition markers from a normalized
// title and collapses whitespace, so "Abbey Road (Remastered)" and
// "Abbey Road" compare equal. The marker subset is the caller's choice: what
// counts as an edition word differs between the weighted Signals and a
// provider's pre-selection.
func PreSelectionCoreTitle(value string, markers []string) string {
	normalized := normalize.Text(value)
	for _, marker := range markers {
		normalized = strings.ReplaceAll(normalized, " "+marker, "")
	}
	return strings.Join(strings.Fields(normalized), " ")
}

// PreSelectionTitleRank scores candidateTitle against sourceTitle.
func PreSelectionTitleRank(sourceTitle string, candidateTitle string, markers []string) int {
	sourceTitle = normalize.Text(sourceTitle)
	candidateTitle = normalize.Text(candidateTitle)
	sourceCore := PreSelectionCoreTitle(sourceTitle, markers)
	candidateCore := PreSelectionCoreTitle(candidateTitle, markers)
	switch {
	case sourceTitle != "" && sourceTitle == candidateTitle:
		return PreSelectionTitleExact
	case sourceCore != "" && sourceCore == candidateCore:
		return PreSelectionTitleCore
	case strings.Contains(candidateTitle, sourceTitle) || strings.Contains(sourceTitle, candidateTitle):
		return PreSelectionTitlePart
	default:
		return 0
	}
}

// PreSelectionArtistRank scores a single candidate artist against the source
// artist list's first entry.
func PreSelectionArtistRank(sourceArtists []string, candidateArtist string) int {
	sourceArtist := ""
	if len(sourceArtists) > 0 {
		sourceArtist = normalize.Text(sourceArtists[0])
	}
	candidateArtist = normalize.Text(candidateArtist)
	switch {
	case sourceArtist != "" && sourceArtist == candidateArtist:
		return PreSelectionArtistExact
	case sourceArtist != "" && strings.Contains(candidateArtist, sourceArtist):
		return PreSelectionArtistContained
	default:
		return 0
	}
}

// PreSelectionReleaseYearRank scores a candidate release date that shares the
// source's year; either side missing the date scores zero.
func PreSelectionReleaseYearRank(sourceReleaseDate string, candidateReleaseDate string) int {
	if sourceReleaseDate == "" || candidateReleaseDate == "" || len(sourceReleaseDate) < 4 || len(candidateReleaseDate) < 4 {
		return 0
	}
	if sourceReleaseDate[:4] == candidateReleaseDate[:4] {
		return PreSelectionReleaseYearMatch
	}
	return 0
}
