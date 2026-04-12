package main

import "github.com/xmbshwll/ariadne"

func filterResolutionByStrength(resolution ariadne.Resolution, minStrength ariadne.MatchStrength) ariadne.Resolution {
	if minStrength == ariadne.MatchStrengthVeryWeak {
		return resolution
	}
	filtered := resolution
	filtered.Matches = make(map[ariadne.ServiceName]ariadne.MatchResult, len(resolution.Matches))
	for service, match := range resolution.Matches {
		pruned, ok := pruneAlbumMatchByStrength(match, minStrength)
		if !ok {
			continue
		}
		filtered.Matches[service] = pruned
	}
	return filtered
}

func pruneAlbumMatchByStrength(match ariadne.MatchResult, minStrength ariadne.MatchStrength) (ariadne.MatchResult, bool) {
	pruned := match
	pruned.Alternates = filterAlternatesByStrength(match.Alternates, minStrength)

	if match.Best == nil || !meetsMinimumStrength(match.Best.Score, minStrength) {
		return ariadne.MatchResult{}, false
	}
	return pruned, true
}

func filterAlternatesByStrength(alternates []ariadne.ScoredMatch, minStrength ariadne.MatchStrength) []ariadne.ScoredMatch {
	filtered := make([]ariadne.ScoredMatch, 0, len(alternates))
	for _, alternate := range alternates {
		if !meetsMinimumStrength(alternate.Score, minStrength) {
			continue
		}
		filtered = append(filtered, alternate)
	}
	return filtered
}

func filterSongResolutionByStrength(resolution ariadne.SongResolution, minStrength ariadne.MatchStrength) ariadne.SongResolution {
	if minStrength == ariadne.MatchStrengthVeryWeak {
		return resolution
	}

	filtered := resolution
	filtered.Matches = make(map[ariadne.ServiceName]ariadne.SongMatchResult, len(resolution.Matches))
	for service, match := range resolution.Matches {
		pruned, ok := pruneSongMatchByStrength(match, minStrength)
		if !ok {
			continue
		}
		filtered.Matches[service] = pruned
	}
	return filtered
}

func pruneSongMatchByStrength(match ariadne.SongMatchResult, minStrength ariadne.MatchStrength) (ariadne.SongMatchResult, bool) {
	pruned := match
	pruned.Alternates = filterSongAlternatesByStrength(match.Alternates, minStrength)

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

	best, alternates := promoteBestSongAlternate(pruned.Alternates)
	pruned.Best = &best
	pruned.Alternates = alternates
	return pruned, true
}

// promoteBestSongAlternate assumes alternates contains at least one entry.
func promoteBestSongAlternate(alternates []ariadne.SongScoredMatch) (ariadne.SongScoredMatch, []ariadne.SongScoredMatch) {
	if len(alternates) == 0 {
		return ariadne.SongScoredMatch{}, []ariadne.SongScoredMatch{}
	}

	bestIndex := 0
	for i := 1; i < len(alternates); i++ {
		if alternates[i].Score > alternates[bestIndex].Score {
			bestIndex = i
		}
	}

	best := alternates[bestIndex]
	remaining := make([]ariadne.SongScoredMatch, 0, len(alternates)-1)
	remaining = append(remaining, alternates[:bestIndex]...)
	remaining = append(remaining, alternates[bestIndex+1:]...)
	return best, remaining
}

func filterSongAlternatesByStrength(alternates []ariadne.SongScoredMatch, minStrength ariadne.MatchStrength) []ariadne.SongScoredMatch {
	filtered := make([]ariadne.SongScoredMatch, 0, len(alternates))
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
