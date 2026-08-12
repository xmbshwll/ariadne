package score

import (
	"fmt"
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

// RankSongs scores and sorts target song candidates for a single source song.
func RankSongs(source model.CanonicalSong, candidates []model.CandidateSong, weights SongWeights) Ranking[model.CandidateSong] {
	ranked := make([]Ranked[model.CandidateSong], 0, len(candidates))
	for _, candidate := range candidates {
		ranked = append(ranked, scoreSongCandidate(source, candidate, weights))
	}
	ranked, best := finalizeRanking(ranked, func(r Ranked[model.CandidateSong]) int { return r.Score }, func(r Ranked[model.CandidateSong]) string { return r.Candidate.CandidateID })
	return Ranking[model.CandidateSong]{Best: best, Ranked: ranked}
}

func scoreSongCandidate(source model.CanonicalSong, candidate model.CandidateSong, weights SongWeights) Ranked[model.CandidateSong] {
	song := candidate.CanonicalSong
	titleWeights := titleSignalWeights{
		exact: weights.TitleExact,
		core:  weights.CoreTitleExact,
	}
	artistWeights := artistSignalWeights{
		primaryExact: weights.PrimaryArtistExact,
		overlap:      weights.ArtistOverlap,
	}
	releaseWeights := releaseDateSignalWeights{
		exact: weights.ReleaseDateExact,
		year:  weights.ReleaseYearExact,
	}

	score, reasons, evidence := collectScoreContributions(
		scoreTitleSignal(source.Title, source.NormalizedTitle, song.Title, song.NormalizedTitle, titleWeights),
		scoreArtistSignal(source.Artists, source.NormalizedArtists, song.Artists, song.NormalizedArtists, artistWeights),
		scoreSongISRC(source, song, weights),
		scoreDurationSignal(source.DurationMS, song.DurationMS, weights.DurationNear),
		scoreNormalizedExactSignal(source.AlbumTitle, source.AlbumNormalizedTitle, song.AlbumTitle, song.AlbumNormalizedTitle, weights.AlbumTitleExact, "album title exact match"),
		scoreSongTrackNumber(source, song, weights),
		scoreReleaseDateSignal(source.ReleaseDate, song.ReleaseDate, releaseWeights),
		scoreExplicitSignal(source.Explicit, song.Explicit, weights.ExplicitMismatch),
		scoreEditionHintSignal(source.EditionHints, song.EditionHints, weights.EditionMismatch),
		scoreEditionMarkerSignal(source.Title, song.Title, weights.EditionMarkerPenalty, weights.EditionMismatch),
	)

	return Ranked[model.CandidateSong]{Candidate: candidate, Score: score, Reasons: reasons, Evidence: evidence}
}

func scoreSongISRC(source model.CanonicalSong, candidate model.CanonicalSong, weights SongWeights) scoreContribution {
	if source.ISRC != "" && candidate.ISRC != "" && strings.EqualFold(source.ISRC, candidate.ISRC) {
		return scoreContribution{value: weights.ISRCExact, reason: "isrc exact match"}
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
