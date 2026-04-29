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
	Evidence  MatchEvidence
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
	score, reasons, evidence := collectScoreContributions(
		scoreSongTitle(source, candidate.CanonicalSong, weights),
		scoreSongArtists(source, candidate.CanonicalSong, weights),
		scoreSongISRC(source, candidate.CanonicalSong, weights),
		scoreSongDuration(source, candidate.CanonicalSong, weights),
		scoreSongAlbumTitle(source, candidate.CanonicalSong, weights),
		scoreSongTrackNumber(source, candidate.CanonicalSong, weights),
		scoreSongReleaseDate(source, candidate.CanonicalSong, weights),
		scoreSongExplicit(source, candidate.CanonicalSong, weights),
		scoreSongEditionHints(source, candidate.CanonicalSong, weights),
		scoreSongEditionMarkers(source, candidate.CanonicalSong, weights),
	)

	return RankedSongCandidate{Candidate: candidate, Score: score, Reasons: reasons, Evidence: evidence}
}

func scoreSongTitle(source model.CanonicalSong, candidate model.CanonicalSong, weights SongWeights) scoreContribution {
	sourceTitle := normalizedOrDerived(source.Title, source.NormalizedTitle)
	candidateTitle := normalizedOrDerived(candidate.Title, candidate.NormalizedTitle)
	if sourceTitle != "" && sourceTitle == candidateTitle {
		return scoreContribution{value: weights.TitleExact, reason: "title exact match", evidence: MatchEvidence{Title: true}}
	}

	sourceCoreTitle := coreTitle(source.Title, source.NormalizedTitle)
	candidateCoreTitle := coreTitle(candidate.Title, candidate.NormalizedTitle)
	if sourceCoreTitle != "" && sourceCoreTitle == candidateCoreTitle {
		return scoreContribution{value: weights.CoreTitleExact, reason: "core title match", evidence: MatchEvidence{Title: true}}
	}

	return scoreContribution{}
}

func scoreSongArtists(source model.CanonicalSong, candidate model.CanonicalSong, weights SongWeights) scoreContribution {
	sourceArtists := source.NormalizedArtists
	if len(sourceArtists) == 0 {
		sourceArtists = normalizeArtistNames(source.Artists)
	}
	candidateArtists := candidate.NormalizedArtists
	if len(candidateArtists) == 0 {
		candidateArtists = normalizeArtistNames(candidate.Artists)
	}
	if len(sourceArtists) == 0 || len(candidateArtists) == 0 {
		return scoreContribution{}
	}
	if sourceArtists[0] == candidateArtists[0] {
		return scoreContribution{value: weights.PrimaryArtistExact, reason: "primary artist exact match", evidence: MatchEvidence{Artist: true}}
	}
	if artistOverlap(sourceArtists, candidateArtists) {
		return scoreContribution{value: weights.ArtistOverlap, reason: "artist overlap", evidence: MatchEvidence{Artist: true}}
	}
	return scoreContribution{}
}

func scoreSongISRC(source model.CanonicalSong, candidate model.CanonicalSong, weights SongWeights) scoreContribution {
	if source.ISRC != "" && candidate.ISRC != "" && strings.EqualFold(source.ISRC, candidate.ISRC) {
		return scoreContribution{value: weights.ISRCExact, reason: "isrc exact match"}
	}
	return scoreContribution{}
}

func scoreSongDuration(source model.CanonicalSong, candidate model.CanonicalSong, weights SongWeights) scoreContribution {
	if source.DurationMS > 0 && candidate.DurationMS > 0 && durationNear(source.DurationMS, candidate.DurationMS) {
		return scoreContribution{value: weights.DurationNear, reason: "duration near match"}
	}
	return scoreContribution{}
}

func scoreSongAlbumTitle(source model.CanonicalSong, candidate model.CanonicalSong, weights SongWeights) scoreContribution {
	if source.AlbumTitle == "" || candidate.AlbumTitle == "" {
		return scoreContribution{}
	}

	sourceAlbumTitle := normalizedOrDerived(source.AlbumTitle, source.AlbumNormalizedTitle)
	candidateAlbumTitle := normalizedOrDerived(candidate.AlbumTitle, candidate.AlbumNormalizedTitle)
	if sourceAlbumTitle != "" && sourceAlbumTitle == candidateAlbumTitle {
		return scoreContribution{value: weights.AlbumTitleExact, reason: "album title exact match"}
	}
	return scoreContribution{}
}

func scoreSongTrackNumber(source model.CanonicalSong, candidate model.CanonicalSong, weights SongWeights) scoreContribution {
	if source.TrackNumber > 0 && candidate.TrackNumber > 0 && source.TrackNumber == candidate.TrackNumber {
		return scoreContribution{
			value:  weights.TrackNumberExact,
			reason: fmt.Sprintf("track number exact match (%d)", source.TrackNumber),
		}
	}
	return scoreContribution{}
}

func scoreSongReleaseDate(source model.CanonicalSong, candidate model.CanonicalSong, weights SongWeights) scoreContribution {
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

func scoreSongExplicit(source model.CanonicalSong, candidate model.CanonicalSong, weights SongWeights) scoreContribution {
	if source.Explicit != candidate.Explicit {
		return scoreContribution{value: weights.ExplicitMismatch, reason: "explicit mismatch"}
	}
	return scoreContribution{}
}

func scoreSongEditionHints(source model.CanonicalSong, candidate model.CanonicalSong, weights SongWeights) scoreContribution {
	if editionMismatch(source.EditionHints, candidate.EditionHints) {
		return scoreContribution{value: weights.EditionMismatch, reason: "edition mismatch"}
	}
	return scoreContribution{}
}

func scoreSongEditionMarkers(source model.CanonicalSong, candidate model.CanonicalSong, weights SongWeights) scoreContribution {
	penalty, markers := editionMarkerPenalty(source.Title, candidate.Title, weights.EditionMarkerPenalty, weights.EditionMismatch)
	if penalty == 0 {
		return scoreContribution{}
	}
	return scoreContribution{value: penalty, reason: "edition marker mismatch: " + strings.Join(markers, ", ")}
}
