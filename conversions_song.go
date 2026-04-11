package ariadne

import (
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/resolve"
)

func toInternalCanonicalSong(song CanonicalSong) model.CanonicalSong {
	return model.CanonicalSong{
		Service:                toInternalServiceName(song.Service),
		SourceID:               song.SourceID,
		SourceURL:              song.SourceURL,
		RegionHint:             song.RegionHint,
		Title:                  song.Title,
		NormalizedTitle:        song.NormalizedTitle,
		Artists:                append([]string(nil), song.Artists...),
		NormalizedArtists:      append([]string(nil), song.NormalizedArtists...),
		DurationMS:             song.DurationMS,
		ISRC:                   song.ISRC,
		Explicit:               song.Explicit,
		DiscNumber:             song.DiscNumber,
		TrackNumber:            song.TrackNumber,
		AlbumID:                song.AlbumID,
		AlbumTitle:             song.AlbumTitle,
		AlbumNormalizedTitle:   song.AlbumNormalizedTitle,
		AlbumArtists:           append([]string(nil), song.AlbumArtists...),
		AlbumNormalizedArtists: append([]string(nil), song.AlbumNormalizedArtists...),
		ReleaseDate:            song.ReleaseDate,
		ArtworkURL:             song.ArtworkURL,
		EditionHints:           append([]string(nil), song.EditionHints...),
	}
}

func fromInternalCanonicalSong(song model.CanonicalSong) CanonicalSong {
	return CanonicalSong{
		Service:                fromInternalServiceName(song.Service),
		SourceID:               song.SourceID,
		SourceURL:              song.SourceURL,
		RegionHint:             song.RegionHint,
		Title:                  song.Title,
		NormalizedTitle:        song.NormalizedTitle,
		Artists:                append([]string(nil), song.Artists...),
		NormalizedArtists:      append([]string(nil), song.NormalizedArtists...),
		DurationMS:             song.DurationMS,
		ISRC:                   song.ISRC,
		Explicit:               song.Explicit,
		DiscNumber:             song.DiscNumber,
		TrackNumber:            song.TrackNumber,
		AlbumID:                song.AlbumID,
		AlbumTitle:             song.AlbumTitle,
		AlbumNormalizedTitle:   song.AlbumNormalizedTitle,
		AlbumArtists:           append([]string(nil), song.AlbumArtists...),
		AlbumNormalizedArtists: append([]string(nil), song.AlbumNormalizedArtists...),
		ReleaseDate:            song.ReleaseDate,
		ArtworkURL:             song.ArtworkURL,
		EditionHints:           append([]string(nil), song.EditionHints...),
	}
}

func toInternalCandidateSong(song CandidateSong) model.CandidateSong {
	return model.CandidateSong{
		CanonicalSong: toInternalCanonicalSong(song.CanonicalSong),
		CandidateID:   song.CandidateID,
		MatchURL:      song.MatchURL,
	}
}

func fromInternalCandidateSong(song model.CandidateSong) CandidateSong {
	return CandidateSong{
		CanonicalSong: fromInternalCanonicalSong(song.CanonicalSong),
		CandidateID:   song.CandidateID,
		MatchURL:      song.MatchURL,
	}
}

func toInternalCandidateSongs(songs []CandidateSong) []model.CandidateSong {
	if len(songs) == 0 {
		return nil
	}
	internal := make([]model.CandidateSong, 0, len(songs))
	for _, song := range songs {
		internal = append(internal, toInternalCandidateSong(song))
	}
	return internal
}

func fromInternalSongScoredMatch(match resolve.SongScoredMatch) SongScoredMatch {
	return SongScoredMatch{
		URL:       match.URL,
		Score:     match.Score,
		Reasons:   append([]string(nil), match.Reasons...),
		Candidate: fromInternalCandidateSong(match.Candidate),
	}
}

func fromInternalSongMatchResult(result resolve.SongMatchResult) SongMatchResult {
	public := SongMatchResult{
		Service:    fromInternalServiceName(result.Service),
		Alternates: make([]SongScoredMatch, 0, len(result.Alternates)),
	}
	if result.Best != nil {
		best := fromInternalSongScoredMatch(*result.Best)
		public.Best = &best
	}
	for _, alternate := range result.Alternates {
		public.Alternates = append(public.Alternates, fromInternalSongScoredMatch(alternate))
	}
	return public
}

func fromInternalSongResolution(resolution resolve.SongResolution) SongResolution {
	matches := make(map[ServiceName]SongMatchResult, len(resolution.Matches))
	for service, match := range resolution.Matches {
		matches[fromInternalServiceName(service)] = fromInternalSongMatchResult(match)
	}
	return SongResolution{
		InputURL: resolution.InputURL,
		Parsed:   fromInternalParsedURL(resolution.Parsed),
		Source:   fromInternalCanonicalSong(resolution.Source),
		Matches:  matches,
	}
}
