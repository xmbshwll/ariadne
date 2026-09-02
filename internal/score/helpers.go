package score

import (
	"sort"
	"strings"

	"github.com/xmbshwll/ariadne/internal/normalize"
)

// editionMarkerCandidates is the comparison subset of the edition vocabulary:
// content flags stay out so explicit and clean editions keep distinct titles.
var editionMarkerCandidates = normalize.EditionComparisonMarkers()

// MatchEvidence describes structured match signals used by resolvers for behavioral decisions.
type MatchEvidence struct {
	Title  bool
	Artist bool
}

// HasTitleOrArtist reports whether a candidate has album/song title or artist evidence.
func (e MatchEvidence) HasTitleOrArtist() bool {
	return e.Title || e.Artist
}

type scoreContribution struct {
	Value    int
	Reason   string
	Evidence MatchEvidence
}

func collectScoreContributions(contributions ...scoreContribution) (int, []string, MatchEvidence) {
	score := 0
	reasons := make([]string, 0, len(contributions))
	Evidence := MatchEvidence{}
	for _, contribution := range contributions {
		if contribution.Reason == "" {
			continue
		}
		score += contribution.Value
		reasons = append(reasons, contribution.Reason)
		Evidence.Title = Evidence.Title || contribution.Evidence.Title
		Evidence.Artist = Evidence.Artist || contribution.Evidence.Artist
	}
	return score, reasons, Evidence
}

func normalizedOrDerived(raw string, normalized string) string {
	if normalized != "" {
		return normalized
	}
	return normalize.Text(raw)
}

func coreTitle(raw string, normalized string) string {
	base := normalizedOrDerived(raw, normalized)
	tokens := strings.Fields(base)
	if len(tokens) == 0 {
		return ""
	}

	markers := editionMarkers(base)
	if len(markers) == 0 {
		return strings.Join(tokens, " ")
	}

	markerTokenSpans := make([][]string, 0, len(markers))
	for _, marker := range markers {
		markerTokens := strings.Fields(marker)
		if len(markerTokens) == 0 {
			continue
		}
		markerTokenSpans = append(markerTokenSpans, markerTokens)
	}
	if len(markerTokenSpans) == 0 {
		return strings.Join(tokens, " ")
	}

	cleaned := make([]string, 0, len(tokens))
	for i := 0; i < len(tokens); {
		markerLength := matchedMarkerLength(tokens[i:], markerTokenSpans)
		if markerLength > 0 {
			i += markerLength
			continue
		}
		cleaned = append(cleaned, tokens[i])
		i++
	}
	return strings.Join(cleaned, " ")
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

func editionMarkerPenalty(sourceTitle string, candidateTitle string, markerPenalty int, mismatchCap int) (int, []string) {
	sourceMarkers := editionMarkers(sourceTitle)
	candidateMarkers := editionMarkers(candidateTitle)
	if len(sourceMarkers) == 0 && len(candidateMarkers) == 0 {
		return 0, nil
	}

	differences := symmetricMarkerDifference(sourceMarkers, candidateMarkers)
	if len(differences) == 0 {
		return 0, nil
	}

	penalty := len(differences) * markerPenalty
	if markerPenalty < 0 && penalty < mismatchCap {
		penalty = mismatchCap
	}
	if markerPenalty > 0 && penalty > mismatchCap {
		penalty = mismatchCap
	}
	return penalty, differences
}

func editionMarkers(title string) []string {
	normalized := normalize.Text(title)
	if normalized == "" {
		return nil
	}

	markers := make([]string, 0, len(editionMarkerCandidates))
	padded := " " + normalized + " "
	for _, candidate := range editionMarkerCandidates {
		needle := " " + candidate + " "
		index := strings.Index(padded, needle)
		if index == -1 {
			continue
		}
		markers = append(markers, candidate)
		padded = padded[:index] + strings.Repeat(" ", len(needle)) + padded[index+len(needle):]
	}
	return markers
}

func matchedMarkerLength(tokens []string, markers [][]string) int {
	for _, marker := range markers {
		if len(tokens) < len(marker) {
			continue
		}
		matched := true
		for i := range marker {
			if tokens[i] != marker[i] {
				matched = false
				break
			}
		}
		if matched {
			return len(marker)
		}
	}
	return 0
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
