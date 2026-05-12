package score

import "sort"

// finalizeRanking sorts ranked candidates by score (highest first, tiebreak by ID) and returns the sorted slice with a pointer to the best.
func finalizeRanking[C any](ranked []C, score func(C) int, id func(C) string) ([]C, *C) {
	sort.SliceStable(ranked, func(i, j int) bool {
		si, sj := score(ranked[i]), score(ranked[j])
		if si == sj {
			return id(ranked[i]) < id(ranked[j])
		}
		return si > sj
	})
	var best *C
	if len(ranked) > 0 {
		b := ranked[0]
		best = &b
	}
	return ranked, best
}
