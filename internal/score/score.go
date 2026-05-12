package score

import (
	"fmt"
	"strings"

	"github.com/xmbshwll/ariadne/internal/model"
)

// Weights configures how ranking signals contribute to the final score.
type Weights struct {
	UPCExact             int
	ISRCStrongOverlap    int
	ISRCPartialScale     int
	TrackTitleStrong     int
	TrackTitlePartial    int
	TitleExact           int
	CoreTitleExact       int
	PrimaryArtistExact   int
	ArtistOverlap        int
	TrackCountExact      int
	TrackCountNear       int
	TrackCountMismatch   int
	ReleaseDateExact     int
	ReleaseYearExact     int
	DurationNear         int
	LabelExact           int
	ExplicitMismatch     int
	EditionMismatch      int
	EditionMarkerPenalty int
}

// DefaultWeights returns the built-in scoring weights.
func DefaultWeights() Weights {
	return Weights{
		UPCExact:             100,
		ISRCStrongOverlap:    80,
		ISRCPartialScale:     60,
		TrackTitleStrong:     30,
		TrackTitlePartial:    20,
		TitleExact:           25,
		CoreTitleExact:       15,
		PrimaryArtistExact:   20,
		ArtistOverlap:        10,
		TrackCountExact:      15,
		TrackCountNear:       5,
		TrackCountMismatch:   -15,
		ReleaseDateExact:     10,
		ReleaseYearExact:     5,
		DurationNear:         10,
		LabelExact:           5,
		ExplicitMismatch:     -10,
		EditionMismatch:      -20,
		EditionMarkerPenalty: -10,
	}
}

// RankedCandidate is one candidate plus its computed score and explanation.
type RankedCandidate struct {
	Candidate model.CandidateAlbum
	Score     int
	Reasons   []string
	Evidence  MatchEvidence
}

// Ranking is the ordered ranking for one target service.
type Ranking struct {
	Best   *RankedCandidate
	Ranked []RankedCandidate
}

// RankAlbums scores and sorts target candidates for a single source album.
func RankAlbums(source model.CanonicalAlbum, candidates []model.CandidateAlbum, weights Weights) Ranking {
	ranked := make([]RankedCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		ranked = append(ranked, scoreCandidate(source, candidate, weights))
	}
	ranked, best := finalizeRanking(ranked, func(r RankedCandidate) int { return r.Score }, func(r RankedCandidate) string { return r.Candidate.CandidateID })
	return Ranking{Best: best, Ranked: ranked}
}

func scoreCandidate(source model.CanonicalAlbum, candidate model.CandidateAlbum, weights Weights) RankedCandidate {
	album := candidate.CanonicalAlbum
	titleWeights := titleSignalWeights{
		exact: weights.TitleExact,
		core:  weights.CoreTitleExact,
	}
	artistWeights := artistSignalWeights{
		primaryExact: weights.PrimaryArtistExact,
		overlap:      weights.ArtistOverlap,
	}
	releaseWeights := releaseDateSignalWeights{
		exact: weights.ReleaseDateExact,
		year:  weights.ReleaseYearExact,
	}

	score, reasons, evidence := collectScoreContributions(
		scoreTitleSignal(source.Title, source.NormalizedTitle, album.Title, album.NormalizedTitle, titleWeights),
		scoreArtistSignal(source.Artists, source.NormalizedArtists, album.Artists, album.NormalizedArtists, artistWeights),
		scoreAlbumUPC(source, album, weights),
		scoreAlbumISRCOverlap(source, album, weights),
		scoreAlbumTrackTitleOverlap(source, album, weights),
		scoreAlbumTrackCount(source, album, weights),
		scoreReleaseDateSignal(source.ReleaseDate, album.ReleaseDate, releaseWeights),
		scoreDurationSignal(source.TotalDurationMS, album.TotalDurationMS, weights.DurationNear),
		scoreAlbumLabel(source, album, weights),
		scoreExplicitSignal(source.Explicit, album.Explicit, weights.ExplicitMismatch),
		scoreEditionHintSignal(source.EditionHints, album.EditionHints, weights.EditionMismatch),
		scoreEditionMarkerSignal(source.Title, album.Title, weights.EditionMarkerPenalty, weights.EditionMismatch),
	)

	return RankedCandidate{
		Candidate: candidate,
		Score:     score,
		Reasons:   reasons,
		Evidence:  evidence,
	}
}

func scoreAlbumUPC(source model.CanonicalAlbum, candidate model.CanonicalAlbum, weights Weights) scoreContribution {
	if source.UPC != "" && candidate.UPC != "" && source.UPC == candidate.UPC {
		return scoreContribution{value: weights.UPCExact, reason: "upc exact match"}
	}
	return scoreContribution{}
}

func scoreAlbumISRCOverlap(source model.CanonicalAlbum, candidate model.CanonicalAlbum, weights Weights) scoreContribution {
	overlap, sourceISRCCount := isrcOverlap(source, candidate)
	if sourceISRCCount == 0 || overlap == 0 {
		return scoreContribution{}
	}

	ratio := float64(overlap) / float64(sourceISRCCount)
	if ratio >= 0.70 {
		return scoreContribution{
			value:  weights.ISRCStrongOverlap,
			reason: fmt.Sprintf("strong isrc overlap (%d/%d)", overlap, sourceISRCCount),
		}
	}

	return scoreContribution{
		value:  int(ratio * float64(weights.ISRCPartialScale)),
		reason: fmt.Sprintf("partial isrc overlap (%d/%d)", overlap, sourceISRCCount),
	}
}

func scoreAlbumTrackTitleOverlap(source model.CanonicalAlbum, candidate model.CanonicalAlbum, weights Weights) scoreContribution {
	overlap, sourceTrackTitleCount := trackTitleOverlap(source, candidate)
	if sourceTrackTitleCount == 0 || overlap == 0 {
		return scoreContribution{}
	}

	ratio := float64(overlap) / float64(sourceTrackTitleCount)
	if ratio >= 0.70 {
		return scoreContribution{
			value:  weights.TrackTitleStrong,
			reason: fmt.Sprintf("strong track title overlap (%d/%d)", overlap, sourceTrackTitleCount),
		}
	}
	if ratio < 0.40 {
		return scoreContribution{}
	}

	partialScore := int(ratio * float64(weights.TrackTitlePartial))
	if partialScore == 0 {
		return scoreContribution{}
	}
	return scoreContribution{
		value:  partialScore,
		reason: fmt.Sprintf("partial track title overlap (%d/%d)", overlap, sourceTrackTitleCount),
	}
}

func scoreAlbumTrackCount(source model.CanonicalAlbum, candidate model.CanonicalAlbum, weights Weights) scoreContribution {
	if source.TrackCount == 0 || candidate.TrackCount == 0 {
		return scoreContribution{}
	}

	diff := source.TrackCount - candidate.TrackCount
	if diff < 0 {
		diff = -diff
	}
	if diff == 0 {
		return scoreContribution{value: weights.TrackCountExact, reason: "track count exact match"}
	}
	if diff == 1 {
		return scoreContribution{value: weights.TrackCountNear, reason: "track count near match"}
	}
	if diff >= 3 {
		return scoreContribution{value: weights.TrackCountMismatch, reason: "track count mismatch"}
	}
	return scoreContribution{}
}

func scoreAlbumLabel(source model.CanonicalAlbum, candidate model.CanonicalAlbum, weights Weights) scoreContribution {
	if source.Label != "" && candidate.Label != "" && normalizedOrDerived(source.Label, "") == normalizedOrDerived(candidate.Label, "") {
		return scoreContribution{value: weights.LabelExact, reason: "label exact match"}
	}
	return scoreContribution{}
}

func isrcOverlap(source model.CanonicalAlbum, candidate model.CanonicalAlbum) (int, int) {
	sourceISRCs := make(map[string]struct{}, len(source.Tracks))
	for _, track := range source.Tracks {
		if track.ISRC == "" {
			continue
		}
		sourceISRCs[strings.ToUpper(track.ISRC)] = struct{}{}
	}
	if len(sourceISRCs) == 0 {
		return 0, 0
	}

	overlap := 0
	seen := make(map[string]struct{}, len(candidate.Tracks))
	for _, track := range candidate.Tracks {
		if track.ISRC == "" {
			continue
		}
		key := strings.ToUpper(track.ISRC)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if _, ok := sourceISRCs[key]; ok {
			overlap++
		}
	}
	return overlap, len(sourceISRCs)
}

func trackTitleOverlap(source model.CanonicalAlbum, candidate model.CanonicalAlbum) (int, int) {
	sourceTitles := make(map[string]struct{}, len(source.Tracks))
	for _, track := range source.Tracks {
		title := normalizedOrDerived(track.Title, track.NormalizedTitle)
		if title == "" {
			continue
		}
		sourceTitles[title] = struct{}{}
	}
	if len(sourceTitles) == 0 {
		return 0, 0
	}

	overlap := 0
	seen := make(map[string]struct{}, len(candidate.Tracks))
	for _, track := range candidate.Tracks {
		title := normalizedOrDerived(track.Title, track.NormalizedTitle)
		if title == "" {
			continue
		}
		if _, ok := seen[title]; ok {
			continue
		}
		seen[title] = struct{}{}
		if _, ok := sourceTitles[title]; ok {
			overlap++
		}
	}
	return overlap, len(sourceTitles)
}
