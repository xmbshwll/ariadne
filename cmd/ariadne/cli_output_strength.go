package main

import "github.com/xmbshwll/ariadne"

func filterResolutionByStrength(resolution ariadne.Resolution, minStrength ariadne.MatchStrength) ariadne.Resolution {
	filtered := resolution
	filtered.Matches = filterMatchesByStrength(resolution.Matches, minStrength, pruneAlbumMatchByStrength)
	return filtered
}

func filterSongResolutionByStrength(resolution ariadne.SongResolution, minStrength ariadne.MatchStrength) ariadne.SongResolution {
	filtered := resolution
	filtered.Matches = filterMatchesByStrength(resolution.Matches, minStrength, pruneSongMatchByStrength)
	return filtered
}

func filterMatchesByStrength[C any](
	matches map[ariadne.ServiceName]ariadne.MatchResultOf[C],
	minStrength ariadne.MatchStrength,
	prune func(ariadne.MatchResultOf[C], ariadne.MatchStrength) (ariadne.MatchResultOf[C], bool),
) map[ariadne.ServiceName]ariadne.MatchResultOf[C] {
	if minStrength == ariadne.MatchStrengthVeryWeak {
		return matches
	}
	filtered := make(map[ariadne.ServiceName]ariadne.MatchResultOf[C], len(matches))
	for service, match := range matches {
		pruned, ok := prune(match, minStrength)
		if !ok {
			continue
		}
		filtered[service] = pruned
	}
	return filtered
}

func pruneAlbumMatchByStrength(match ariadne.MatchResult, minStrength ariadne.MatchStrength) (ariadne.MatchResult, bool) {
	pruned := match
	pruned.Alternates = filterScoredByStrength(match.Alternates, minStrength)

	if match.Best == nil || !meetsMinimumStrength(match.Best.Score, minStrength) {
		return ariadne.MatchResult{}, false
	}
	return pruned, true
}

func pruneSongMatchByStrength(match ariadne.SongMatchResult, minStrength ariadne.MatchStrength) (ariadne.SongMatchResult, bool) {
	pruned := match
	pruned.Alternates = filterScoredByStrength(match.Alternates, minStrength)

	if match.Best != nil && meetsMinimumStrength(match.Best.Score, minStrength) {
		best := *match.Best
		pruned.Best = &best
		return pruned, true
	}

	// Songs intentionally keep the service when strong alternates remain, even if
	// the original Best candidate falls below the threshold. Album output is
	// stricter and drops the whole service when Best is pruned.
	if len(pruned.Alternates) == 0 {
		return ariadne.SongMatchResult{}, false
	}

	best, alternates := promoteBestAlternate(pruned.Alternates)
	pruned.Best = &best
	pruned.Alternates = alternates
	return pruned, true
}

// promoteBestAlternate requires at least one entry; callers check beforehand.
func promoteBestAlternate[C any](alternates []ariadne.ScoredMatchOf[C]) (ariadne.ScoredMatchOf[C], []ariadne.ScoredMatchOf[C]) {
	bestIndex := 0
	for i := 1; i < len(alternates); i++ {
		if alternates[i].Score > alternates[bestIndex].Score {
			bestIndex = i
		}
	}

	best := alternates[bestIndex]
	remaining := make([]ariadne.ScoredMatchOf[C], 0, len(alternates)-1)
	remaining = append(remaining, alternates[:bestIndex]...)
	remaining = append(remaining, alternates[bestIndex+1:]...)
	return best, remaining
}

func filterScoredByStrength[C any](alternates []ariadne.ScoredMatchOf[C], minStrength ariadne.MatchStrength) []ariadne.ScoredMatchOf[C] {
	filtered := make([]ariadne.ScoredMatchOf[C], 0, len(alternates))
	for _, alternate := range alternates {
		if !meetsMinimumStrength(alternate.Score, minStrength) {
			continue
		}
		filtered = append(filtered, alternate)
	}
	return filtered
}

func meetsMinimumStrength(score int, minStrength ariadne.MatchStrength) bool {
	return matchStrengthRank(ariadne.MatchStrengthForScore(score)) >= matchStrengthRank(minStrength)
}

func matchStrengthRank(strength ariadne.MatchStrength) int {
	switch strength {
	case ariadne.MatchStrengthStrong:
		return 3
	case ariadne.MatchStrengthProbable:
		return 2
	case ariadne.MatchStrengthWeak:
		return 1
	default:
		return 0
	}
}
