package score

import (
	"fmt"
	"sort"
	"strings"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/normalize"
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
}

// Ranking is the ordered ranking for one target service.
type Ranking struct {
	Best   *RankedCandidate
	Ranked []RankedCandidate
}

type scoreContribution struct {
	value  int
	reason string
}

func collectScoreContributions(contributions ...scoreContribution) (int, []string) {
	score := 0
	reasons := make([]string, 0, len(contributions))
	for _, contribution := range contributions {
		if contribution.reason == "" {
			continue
		}
		score += contribution.value
		reasons = append(reasons, contribution.reason)
	}
	return score, reasons
}

// RankAlbums scores and sorts target candidates for a single source album.
func RankAlbums(source model.CanonicalAlbum, candidates []model.CandidateAlbum, weights Weights) Ranking {
	ranked := make([]RankedCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		ranked = append(ranked, scoreCandidate(source, candidate, weights))
	}

	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].Score == ranked[j].Score {
			return ranked[i].Candidate.CandidateID < ranked[j].Candidate.CandidateID
		}
		return ranked[i].Score > ranked[j].Score
	})

	ranking := Ranking{Ranked: ranked}
	if len(ranked) > 0 {
		best := ranked[0]
		ranking.Best = &best
	}
	return ranking
}

func scoreCandidate(source model.CanonicalAlbum, candidate model.CandidateAlbum, weights Weights) RankedCandidate {
	score, reasons := collectScoreContributions(
		scoreAlbumTitle(source, candidate.CanonicalAlbum, weights),
		scoreAlbumArtists(source, candidate.CanonicalAlbum, weights),
		scoreAlbumUPC(source, candidate.CanonicalAlbum, weights),
		scoreAlbumISRCOverlap(source, candidate.CanonicalAlbum, weights),
		scoreAlbumTrackTitleOverlap(source, candidate.CanonicalAlbum, weights),
		scoreAlbumTrackCount(source, candidate.CanonicalAlbum, weights),
		scoreAlbumReleaseDate(source, candidate.CanonicalAlbum, weights),
		scoreAlbumDuration(source, candidate.CanonicalAlbum, weights),
		scoreAlbumLabel(source, candidate.CanonicalAlbum, weights),
		scoreAlbumExplicit(source, candidate.CanonicalAlbum, weights),
		scoreAlbumEditionHints(source, candidate.CanonicalAlbum, weights),
		scoreAlbumEditionMarkers(source, candidate.CanonicalAlbum, weights),
	)

	return RankedCandidate{
		Candidate: candidate,
		Score:     score,
		Reasons:   reasons,
	}
}

func scoreAlbumTitle(source model.CanonicalAlbum, candidate model.CanonicalAlbum, weights Weights) scoreContribution {
	sourceTitle := normalizedOrDerived(source.Title, source.NormalizedTitle)
	candidateTitle := normalizedOrDerived(candidate.Title, candidate.NormalizedTitle)
	if sourceTitle != "" && sourceTitle == candidateTitle {
		return scoreContribution{value: weights.TitleExact, reason: "title exact match"}
	}

	sourceCoreTitle := coreTitle(source.Title, source.NormalizedTitle)
	candidateCoreTitle := coreTitle(candidate.Title, candidate.NormalizedTitle)
	if sourceCoreTitle != "" && sourceCoreTitle == candidateCoreTitle {
		return scoreContribution{value: weights.CoreTitleExact, reason: "core title match"}
	}

	return scoreContribution{}
}

func scoreAlbumArtists(source model.CanonicalAlbum, candidate model.CanonicalAlbum, weights Weights) scoreContribution {
	sourceArtists := normalizedArtists(source)
	candidateArtists := normalizedArtists(candidate)
	if len(sourceArtists) == 0 || len(candidateArtists) == 0 {
		return scoreContribution{}
	}
	if sourceArtists[0] == candidateArtists[0] {
		return scoreContribution{value: weights.PrimaryArtistExact, reason: "primary artist exact match"}
	}
	if artistOverlap(sourceArtists, candidateArtists) {
		return scoreContribution{value: weights.ArtistOverlap, reason: "artist overlap"}
	}
	return scoreContribution{}
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

func scoreAlbumReleaseDate(source model.CanonicalAlbum, candidate model.CanonicalAlbum, weights Weights) scoreContribution {
	if source.ReleaseDate == "" || candidate.ReleaseDate == "" {
		return scoreContribution{}
	}
	if source.ReleaseDate == candidate.ReleaseDate {
		return scoreContribution{value: weights.ReleaseDateExact, reason: "release date exact match"}
	}
	if sameReleaseYear(source.ReleaseDate, candidate.ReleaseDate) {
		return scoreContribution{value: weights.ReleaseYearExact, reason: "release year match"}
	}
	return scoreContribution{}
}

func scoreAlbumDuration(source model.CanonicalAlbum, candidate model.CanonicalAlbum, weights Weights) scoreContribution {
	if source.TotalDurationMS > 0 && candidate.TotalDurationMS > 0 && durationNear(source.TotalDurationMS, candidate.TotalDurationMS) {
		return scoreContribution{value: weights.DurationNear, reason: "duration near match"}
	}
	return scoreContribution{}
}

func scoreAlbumLabel(source model.CanonicalAlbum, candidate model.CanonicalAlbum, weights Weights) scoreContribution {
	if source.Label != "" && candidate.Label != "" && normalizedOrDerived(source.Label, "") == normalizedOrDerived(candidate.Label, "") {
		return scoreContribution{value: weights.LabelExact, reason: "label exact match"}
	}
	return scoreContribution{}
}

func scoreAlbumExplicit(source model.CanonicalAlbum, candidate model.CanonicalAlbum, weights Weights) scoreContribution {
	if source.Explicit != candidate.Explicit {
		return scoreContribution{value: weights.ExplicitMismatch, reason: "explicit mismatch"}
	}
	return scoreContribution{}
}

func scoreAlbumEditionHints(source model.CanonicalAlbum, candidate model.CanonicalAlbum, weights Weights) scoreContribution {
	if editionMismatch(source.EditionHints, candidate.EditionHints) {
		return scoreContribution{value: weights.EditionMismatch, reason: "edition mismatch"}
	}
	return scoreContribution{}
}

func scoreAlbumEditionMarkers(source model.CanonicalAlbum, candidate model.CanonicalAlbum, weights Weights) scoreContribution {
	penalty, markers := editionMarkerMismatchPenalty(source, candidate, weights)
	if penalty == 0 {
		return scoreContribution{}
	}
	return scoreContribution{value: penalty, reason: "edition marker mismatch: " + strings.Join(markers, ", ")}
}

func normalizedOrDerived(raw string, normalized string) string {
	if normalized != "" {
		return normalized
	}
	return normalize.Text(raw)
}

func coreTitle(raw string, normalized string) string {
	base := normalizedOrDerived(raw, normalized)
	base = strings.ReplaceAll(base, "(", " ")
	base = strings.ReplaceAll(base, ")", " ")
	base = strings.ReplaceAll(base, "[", " ")
	base = strings.ReplaceAll(base, "]", " ")
	for _, marker := range editionMarkers(raw) {
		base = strings.ReplaceAll(base, marker, " ")
	}
	return strings.Join(strings.Fields(base), " ")
}

func normalizedArtists(album model.CanonicalAlbum) []string {
	if len(album.NormalizedArtists) > 0 {
		return album.NormalizedArtists
	}
	return normalize.Artists(album.Artists)
}

func artistOverlap(left []string, right []string) bool {
	seen := make(map[string]struct{}, len(left))
	for _, value := range left {
		seen[value] = struct{}{}
	}
	for _, value := range right {
		if _, ok := seen[value]; ok {
			return true
		}
	}
	return false
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

func sameReleaseYear(left string, right string) bool {
	if len(left) < 4 || len(right) < 4 {
		return false
	}
	return left[:4] == right[:4]
}

func durationNear(leftMS int, rightMS int) bool {
	delta := leftMS - rightMS
	if delta < 0 {
		delta = -delta
	}
	threshold := max(leftMS/50, 1000)
	return delta <= threshold
}

func editionMismatch(left []string, right []string) bool {
	if len(left) == 0 || len(right) == 0 {
		return false
	}
	leftSet := make(map[string]struct{}, len(left))
	for _, value := range left {
		leftSet[value] = struct{}{}
	}
	for _, value := range right {
		if _, ok := leftSet[value]; ok {
			return false
		}
	}
	return true
}

func editionMarkerMismatchPenalty(source model.CanonicalAlbum, candidate model.CanonicalAlbum, weights Weights) (int, []string) {
	sourceMarkers := editionMarkers(source.Title)
	candidateMarkers := editionMarkers(candidate.Title)
	if len(sourceMarkers) == 0 && len(candidateMarkers) == 0 {
		return 0, nil
	}

	differences := symmetricMarkerDifference(sourceMarkers, candidateMarkers)
	if len(differences) == 0 {
		return 0, nil
	}
	penalty := len(differences) * weights.EditionMarkerPenalty
	if weights.EditionMarkerPenalty < 0 && penalty < weights.EditionMismatch {
		penalty = weights.EditionMismatch
	}
	if weights.EditionMarkerPenalty > 0 && penalty > weights.EditionMismatch {
		penalty = weights.EditionMismatch
	}
	return penalty, differences
}

func editionMarkers(title string) []string {
	normalized := normalize.Text(title)
	candidates := []string{"super deluxe", "deluxe", "remix", "mix", "anniversary", "live", "acoustic"}
	markers := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if strings.Contains(normalized, candidate) {
			markers = append(markers, candidate)
		}
	}
	return markers
}

func symmetricMarkerDifference(left []string, right []string) []string {
	leftSet := make(map[string]struct{}, len(left))
	rightSet := make(map[string]struct{}, len(right))
	for _, value := range left {
		leftSet[value] = struct{}{}
	}
	for _, value := range right {
		rightSet[value] = struct{}{}
	}

	differences := make([]string, 0)
	for _, value := range left {
		if _, ok := rightSet[value]; !ok {
			differences = append(differences, value)
		}
	}
	for _, value := range right {
		if _, ok := leftSet[value]; !ok {
			differences = append(differences, value)
		}
	}
	sort.Strings(differences)
	return differences
}
