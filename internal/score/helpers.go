package score

import (
	"sort"
	"strings"

	"github.com/xmbshwll/ariadne/internal/normalize"
)

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

func normalizeArtistNames(values []string) []string {
	items := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		normalized := normalizedOrDerived(value, "")
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		items = append(items, normalized)
	}
	return items
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
