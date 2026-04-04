package score

import (
	"fmt"
	"sort"
	"strings"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/normalize"
)

const (
	upcExactScore             = 100
	isrcStrongOverlapScore    = 80
	trackTitleStrongScore     = 30
	trackTitlePartialScale    = 20
	titleExactScore           = 25
	coreTitleExactScore       = 15
	primaryArtistExactScore   = 20
	artistOverlapScore        = 10
	trackCountExactScore      = 15
	trackCountNearScore       = 5
	trackCountMismatchPenalty = -15
	releaseDateExactScore     = 10
	releaseYearExactScore     = 5
	durationNearScore         = 10
	labelExactScore           = 5
	explicitMismatchPenalty   = -10
	editionMismatchPenalty    = -20
	editionMarkerPenalty      = -10
)

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

// RankAlbums scores and sorts target candidates for a single source album.
func RankAlbums(source model.CanonicalAlbum, candidates []model.CandidateAlbum) Ranking {
	ranked := make([]RankedCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		ranked = append(ranked, scoreCandidate(source, candidate))
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

func scoreCandidate(source model.CanonicalAlbum, candidate model.CandidateAlbum) RankedCandidate {
	score := 0
	reasons := make([]string, 0, 8)

	sourceTitle := normalizedOrDerived(source.Title, source.NormalizedTitle)
	candidateTitle := normalizedOrDerived(candidate.Title, candidate.NormalizedTitle)
	sourceCoreTitle := coreTitle(source.Title, source.NormalizedTitle)
	candidateCoreTitle := coreTitle(candidate.Title, candidate.NormalizedTitle)
	if sourceTitle != "" && sourceTitle == candidateTitle {
		score += titleExactScore
		reasons = append(reasons, "title exact match")
	} else if sourceCoreTitle != "" && sourceCoreTitle == candidateCoreTitle {
		score += coreTitleExactScore
		reasons = append(reasons, "core title match")
	}

	sourceArtists := normalizedArtists(source)
	candidateArtists := normalizedArtists(candidate.CanonicalAlbum)
	if len(sourceArtists) > 0 && len(candidateArtists) > 0 {
		if sourceArtists[0] == candidateArtists[0] {
			score += primaryArtistExactScore
			reasons = append(reasons, "primary artist exact match")
		} else if artistOverlap(sourceArtists, candidateArtists) {
			score += artistOverlapScore
			reasons = append(reasons, "artist overlap")
		}
	}

	if source.UPC != "" && candidate.UPC != "" && source.UPC == candidate.UPC {
		score += upcExactScore
		reasons = append(reasons, "upc exact match")
	}

	overlap, sourceISRCCount := isrcOverlap(source, candidate.CanonicalAlbum)
	if sourceISRCCount > 0 && overlap > 0 {
		ratio := float64(overlap) / float64(sourceISRCCount)
		if ratio >= 0.70 {
			score += isrcStrongOverlapScore
			reasons = append(reasons, fmt.Sprintf("strong isrc overlap (%d/%d)", overlap, sourceISRCCount))
		} else {
			partialScore := int(ratio * 60)
			score += partialScore
			reasons = append(reasons, fmt.Sprintf("partial isrc overlap (%d/%d)", overlap, sourceISRCCount))
		}
	}

	trackTitleOverlapCount, sourceTrackTitleCount := trackTitleOverlap(source, candidate.CanonicalAlbum)
	if sourceTrackTitleCount > 0 && trackTitleOverlapCount > 0 {
		ratio := float64(trackTitleOverlapCount) / float64(sourceTrackTitleCount)
		switch {
		case ratio >= 0.70:
			score += trackTitleStrongScore
			reasons = append(reasons, fmt.Sprintf("strong track title overlap (%d/%d)", trackTitleOverlapCount, sourceTrackTitleCount))
		case ratio >= 0.40:
			partialScore := int(ratio * trackTitlePartialScale)
			if partialScore > 0 {
				score += partialScore
				reasons = append(reasons, fmt.Sprintf("partial track title overlap (%d/%d)", trackTitleOverlapCount, sourceTrackTitleCount))
			}
		}
	}

	if source.TrackCount > 0 && candidate.TrackCount > 0 {
		diff := source.TrackCount - candidate.TrackCount
		if diff < 0 {
			diff = -diff
		}
		switch {
		case diff == 0:
			score += trackCountExactScore
			reasons = append(reasons, "track count exact match")
		case diff == 1:
			score += trackCountNearScore
			reasons = append(reasons, "track count near match")
		case diff >= 3:
			score += trackCountMismatchPenalty
			reasons = append(reasons, "track count mismatch")
		}
	}

	if source.ReleaseDate != "" && candidate.ReleaseDate != "" {
		switch {
		case source.ReleaseDate == candidate.ReleaseDate:
			score += releaseDateExactScore
			reasons = append(reasons, "release date exact match")
		case sameReleaseYear(source.ReleaseDate, candidate.ReleaseDate):
			score += releaseYearExactScore
			reasons = append(reasons, "release year match")
		}
	}

	if source.TotalDurationMS > 0 && candidate.TotalDurationMS > 0 && durationNear(source.TotalDurationMS, candidate.TotalDurationMS) {
		score += durationNearScore
		reasons = append(reasons, "duration near match")
	}

	if source.Label != "" && candidate.Label != "" && normalizedOrDerived(source.Label, "") == normalizedOrDerived(candidate.Label, "") {
		score += labelExactScore
		reasons = append(reasons, "label exact match")
	}

	if source.Explicit != candidate.Explicit {
		score += explicitMismatchPenalty
		reasons = append(reasons, "explicit mismatch")
	}

	if editionMismatch(source.EditionHints, candidate.EditionHints) {
		score += editionMismatchPenalty
		reasons = append(reasons, "edition mismatch")
	}

	if penalty, markers := editionMarkerMismatchPenalty(source, candidate.CanonicalAlbum); penalty != 0 {
		score += penalty
		reasons = append(reasons, "edition marker mismatch: "+strings.Join(markers, ", "))
	}

	return RankedCandidate{
		Candidate: candidate,
		Score:     score,
		Reasons:   reasons,
	}
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
	threshold := leftMS / 50
	if threshold < 1000 {
		threshold = 1000
	}
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

func editionMarkerMismatchPenalty(source model.CanonicalAlbum, candidate model.CanonicalAlbum) (int, []string) {
	sourceMarkers := editionMarkers(source.Title)
	candidateMarkers := editionMarkers(candidate.Title)
	if len(sourceMarkers) == 0 && len(candidateMarkers) == 0 {
		return 0, nil
	}

	differences := symmetricMarkerDifference(sourceMarkers, candidateMarkers)
	if len(differences) == 0 {
		return 0, nil
	}
	penalty := len(differences) * editionMarkerPenalty
	if penalty < editionMismatchPenalty {
		penalty = editionMismatchPenalty
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
