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

// pruneMatchByStrength filters alternates below the threshold, then applies the
// entity shape's keep policy for a best that no longer qualifies: album output
// drops the whole service, while songs keep the service when strong alternates
// remain by promoting the best alternate into the best slot.
func pruneMatchByStrength[C any](match ariadne.MatchResultOf[C], minStrength ariadne.MatchStrength, promoteAlternate bool) (ariadne.MatchResultOf[C], bool) {
	pruned := match
	pruned.Alternates = filterScoredByStrength(match.Alternates, minStrength)

	if match.Best != nil && meetsMinimumStrength(match.Best.Score, minStrength) {
		best := *match.Best
		pruned.Best = &best
		return pruned, true
	}
	if !promoteAlternate || len(pruned.Alternates) == 0 {
		return ariadne.MatchResultOf[C]{}, false
	}

	best, alternates := promoteBestAlternate(pruned.Alternates)
	pruned.Best = &best
	pruned.Alternates = alternates
	return pruned, true
}

func pruneAlbumMatchByStrength(match ariadne.MatchResult, minStrength ariadne.MatchStrength) (ariadne.MatchResult, bool) {
	return pruneMatchByStrength(match, minStrength, false)
}

func pruneSongMatchByStrength(match ariadne.SongMatchResult, minStrength ariadne.MatchStrength) (ariadne.SongMatchResult, bool) {
	return pruneMatchByStrength(match, minStrength, true)
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
