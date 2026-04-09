package score

import (
	"fmt"
	"sort"
	"strings"

	"github.com/xmbshwll/ariadne/internal/model"
)

// SongWeights configures how ranking signals contribute to song match scores.
type SongWeights struct {
	ISRCExact            int
	TitleExact           int
	CoreTitleExact       int
	PrimaryArtistExact   int
	ArtistOverlap        int
	DurationNear         int
	AlbumTitleExact      int
	ReleaseDateExact     int
	ReleaseYearExact     int
	TrackNumberExact     int
	ExplicitMismatch     int
	EditionMismatch      int
	EditionMarkerPenalty int
}

// DefaultSongWeights returns the built-in song scoring weights.
func DefaultSongWeights() SongWeights {
	return SongWeights{
		ISRCExact:            100,
		TitleExact:           25,
		CoreTitleExact:       15,
		PrimaryArtistExact:   20,
		ArtistOverlap:        10,
		DurationNear:         15,
		AlbumTitleExact:      5,
		ReleaseDateExact:     5,
		ReleaseYearExact:     3,
		TrackNumberExact:     3,
		ExplicitMismatch:     -10,
		EditionMismatch:      -20,
		EditionMarkerPenalty: -10,
	}
}

// RankedSongCandidate is one song candidate plus its computed score and explanation.
type RankedSongCandidate struct {
	Candidate model.CandidateSong
	Score     int
	Reasons   []string
}

// SongRanking is the ordered song ranking for one target service.
type SongRanking struct {
	Best   *RankedSongCandidate
	Ranked []RankedSongCandidate
}

// RankSongs scores and sorts target song candidates for a single source song.
func RankSongs(source model.CanonicalSong, candidates []model.CandidateSong, weights SongWeights) SongRanking {
	ranked := make([]RankedSongCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		ranked = append(ranked, scoreSongCandidate(source, candidate, weights))
	}

	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].Score == ranked[j].Score {
			return ranked[i].Candidate.CandidateID < ranked[j].Candidate.CandidateID
		}
		return ranked[i].Score > ranked[j].Score
	})

	ranking := SongRanking{Ranked: ranked}
	if len(ranked) > 0 {
		best := ranked[0]
		ranking.Best = &best
	}
	return ranking
}

func scoreSongCandidate(source model.CanonicalSong, candidate model.CandidateSong, weights SongWeights) RankedSongCandidate {
	score := 0
	reasons := make([]string, 0, 8)

	sourceTitle := normalizedOrDerived(source.Title, source.NormalizedTitle)
	candidateTitle := normalizedOrDerived(candidate.Title, candidate.NormalizedTitle)
	sourceCoreTitle := coreTitle(source.Title, source.NormalizedTitle)
	candidateCoreTitle := coreTitle(candidate.Title, candidate.NormalizedTitle)
	if sourceTitle != "" && sourceTitle == candidateTitle {
		score += weights.TitleExact
		reasons = append(reasons, "title exact match")
	} else if sourceCoreTitle != "" && sourceCoreTitle == candidateCoreTitle {
		score += weights.CoreTitleExact
		reasons = append(reasons, "core title match")
	}

	sourceArtists := source.NormalizedArtists
	if len(sourceArtists) == 0 {
		sourceArtists = normalizeArtists(source.Artists)
	}
	candidateArtists := candidate.NormalizedArtists
	if len(candidateArtists) == 0 {
		candidateArtists = normalizeArtists(candidate.Artists)
	}
	if len(sourceArtists) > 0 && len(candidateArtists) > 0 {
		if sourceArtists[0] == candidateArtists[0] {
			score += weights.PrimaryArtistExact
			reasons = append(reasons, "primary artist exact match")
		} else if artistOverlap(sourceArtists, candidateArtists) {
			score += weights.ArtistOverlap
			reasons = append(reasons, "artist overlap")
		}
	}

	if source.ISRC != "" && candidate.ISRC != "" && strings.EqualFold(source.ISRC, candidate.ISRC) {
		score += weights.ISRCExact
		reasons = append(reasons, "isrc exact match")
	}

	if source.DurationMS > 0 && candidate.DurationMS > 0 && durationNear(source.DurationMS, candidate.DurationMS) {
		score += weights.DurationNear
		reasons = append(reasons, "duration near match")
	}

	if source.AlbumTitle != "" && candidate.AlbumTitle != "" {
		sourceAlbumTitle := normalizedOrDerived(source.AlbumTitle, source.AlbumNormalizedTitle)
		candidateAlbumTitle := normalizedOrDerived(candidate.AlbumTitle, candidate.AlbumNormalizedTitle)
		if sourceAlbumTitle != "" && sourceAlbumTitle == candidateAlbumTitle {
			score += weights.AlbumTitleExact
			reasons = append(reasons, "album title exact match")
		}
	}

	if source.TrackNumber > 0 && candidate.TrackNumber > 0 && source.TrackNumber == candidate.TrackNumber {
		score += weights.TrackNumberExact
		reasons = append(reasons, fmt.Sprintf("track number exact match (%d)", source.TrackNumber))
	}

	if source.ReleaseDate != "" && candidate.ReleaseDate != "" {
		switch {
		case source.ReleaseDate == candidate.ReleaseDate:
			score += weights.ReleaseDateExact
			reasons = append(reasons, "release date exact match")
		case sameReleaseYear(source.ReleaseDate, candidate.ReleaseDate):
			score += weights.ReleaseYearExact
			reasons = append(reasons, "release year match")
		}
	}

	if source.Explicit != candidate.Explicit {
		score += weights.ExplicitMismatch
		reasons = append(reasons, "explicit mismatch")
	}

	if editionMismatch(source.EditionHints, candidate.EditionHints) {
		score += weights.EditionMismatch
		reasons = append(reasons, "edition mismatch")
	}

	if penalty, markers := editionMarkerMismatchPenaltySongs(source, candidate.CanonicalSong, weights); penalty != 0 {
		score += penalty
		reasons = append(reasons, "edition marker mismatch: "+strings.Join(markers, ", "))
	}

	return RankedSongCandidate{Candidate: candidate, Score: score, Reasons: reasons}
}

func normalizeArtists(values []string) []string {
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

func editionMarkerMismatchPenaltySongs(source model.CanonicalSong, candidate model.CanonicalSong, weights SongWeights) (int, []string) {
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
