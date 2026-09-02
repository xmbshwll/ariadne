package score

import (
	"strings"

	"github.com/xmbshwll/ariadne/internal/normalize"
)

type titleSignalWeights struct {
	Exact int
	Core  int
}

type artistSignalWeights struct {
	PrimaryExact int
	Overlap      int
}

type releaseDateSignalWeights struct {
	Exact int
	Year  int
}

func scoreTitleSignal(sourceTitle string, sourceNormalizedTitle string, candidateTitle string, candidateNormalizedTitle string, weights titleSignalWeights) scoreContribution {
	source := normalizedOrDerived(sourceTitle, sourceNormalizedTitle)
	candidate := normalizedOrDerived(candidateTitle, candidateNormalizedTitle)
	if source != "" && source == candidate {
		return scoreContribution{Value: weights.Exact, Reason: "title exact match", Evidence: MatchEvidence{Title: true}}
	}

	sourceCore := coreTitle(sourceTitle, sourceNormalizedTitle)
	candidateCore := coreTitle(candidateTitle, candidateNormalizedTitle)
	if sourceCore != "" && sourceCore == candidateCore {
		return scoreContribution{Value: weights.Core, Reason: "core title match", Evidence: MatchEvidence{Title: true}}
	}
	return scoreContribution{}
}

func scoreArtistSignal(sourceArtists []string, sourceNormalizedArtists []string, candidateArtists []string, candidateNormalizedArtists []string, weights artistSignalWeights) scoreContribution {
	source := normalizedArtists(sourceArtists, sourceNormalizedArtists)
	candidate := normalizedArtists(candidateArtists, candidateNormalizedArtists)
	if len(source) == 0 || len(candidate) == 0 {
		return scoreContribution{}
	}
	if source[0] == candidate[0] {
		return scoreContribution{Value: weights.PrimaryExact, Reason: "primary artist exact match", Evidence: MatchEvidence{Artist: true}}
	}
	if artistOverlap(source, candidate) {
		return scoreContribution{Value: weights.Overlap, Reason: "artist overlap", Evidence: MatchEvidence{Artist: true}}
	}
	return scoreContribution{}
}

func scoreReleaseDateSignal(sourceDate string, candidateDate string, weights releaseDateSignalWeights) scoreContribution {
	if sourceDate == "" || candidateDate == "" {
		return scoreContribution{}
	}
	if sourceDate == candidateDate {
		return scoreContribution{Value: weights.Exact, Reason: "release date exact match"}
	}
	if sameReleaseYear(sourceDate, candidateDate) {
		return scoreContribution{Value: weights.Year, Reason: "release year match"}
	}
	return scoreContribution{}
}

func scoreDurationSignal(sourceMS int, candidateMS int, weight int) scoreContribution {
	if sourceMS > 0 && candidateMS > 0 && durationNear(sourceMS, candidateMS) {
		return scoreContribution{Value: weight, Reason: "duration near match"}
	}
	return scoreContribution{}
}

func scoreExplicitSignal(sourceExplicit bool, candidateExplicit bool, weight int) scoreContribution {
	if sourceExplicit != candidateExplicit {
		return scoreContribution{Value: weight, Reason: "explicit mismatch"}
	}
	return scoreContribution{}
}

func scoreEditionHintSignal(sourceHints []string, candidateHints []string, weight int) scoreContribution {
	if editionMismatch(sourceHints, candidateHints) {
		return scoreContribution{Value: weight, Reason: "edition mismatch"}
	}
	return scoreContribution{}
}

func scoreEditionMarkerSignal(sourceTitle string, candidateTitle string, markerPenalty int, mismatchCap int) scoreContribution {
	penalty, markers := editionMarkerPenalty(sourceTitle, candidateTitle, markerPenalty, mismatchCap)
	if penalty == 0 {
		return scoreContribution{}
	}
	return scoreContribution{Value: penalty, Reason: "edition marker mismatch: " + strings.Join(markers, ", ")}
}

func scoreNormalizedExactSignal(sourceRaw string, sourceNormalized string, candidateRaw string, candidateNormalized string, weight int, reason string) scoreContribution {
	source := normalizedOrDerived(sourceRaw, sourceNormalized)
	candidate := normalizedOrDerived(candidateRaw, candidateNormalized)
	if source != "" && source == candidate {
		return scoreContribution{Value: weight, Reason: reason}
	}
	return scoreContribution{}
}

func normalizedArtists(raw []string, normalized []string) []string {
	if len(normalized) > 0 {
		return normalized
	}
	return normalize.Artists(raw)
}
