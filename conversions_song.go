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
		Artists:                copyStrings(song.Artists),
		NormalizedArtists:      copyStrings(song.NormalizedArtists),
		DurationMS:             song.DurationMS,
		ISRC:                   song.ISRC,
		Explicit:               song.Explicit,
		DiscNumber:             song.DiscNumber,
		TrackNumber:            song.TrackNumber,
		AlbumID:                song.AlbumID,
		AlbumTitle:             song.AlbumTitle,
		AlbumNormalizedTitle:   song.AlbumNormalizedTitle,
		AlbumArtists:           copyStrings(song.AlbumArtists),
		AlbumNormalizedArtists: copyStrings(song.AlbumNormalizedArtists),
		ReleaseDate:            song.ReleaseDate,
		ArtworkURL:             song.ArtworkURL,
		EditionHints:           copyStrings(song.EditionHints),
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
		Artists:                copyStrings(song.Artists),
		NormalizedArtists:      copyStrings(song.NormalizedArtists),
		DurationMS:             song.DurationMS,
		ISRC:                   song.ISRC,
		Explicit:               song.Explicit,
		DiscNumber:             song.DiscNumber,
		TrackNumber:            song.TrackNumber,
		AlbumID:                song.AlbumID,
		AlbumTitle:             song.AlbumTitle,
		AlbumNormalizedTitle:   song.AlbumNormalizedTitle,
		AlbumArtists:           copyStrings(song.AlbumArtists),
		AlbumNormalizedArtists: copyStrings(song.AlbumNormalizedArtists),
		ReleaseDate:            song.ReleaseDate,
		ArtworkURL:             song.ArtworkURL,
		EditionHints:           copyStrings(song.EditionHints),
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
	return translateCompactSlice(songs, toInternalCandidateSong)
}

func fromInternalSongScoredMatch(match resolve.SongScoredMatch) SongScoredMatch {
	return SongScoredMatch{
		URL:       match.URL,
		Score:     match.Score,
		Reasons:   copyStrings(match.Reasons),
		Candidate: fromInternalCandidateSong(match.Candidate),
	}
}

func fromInternalSongMatchResult(result resolve.SongMatchResult) SongMatchResult {
	public := SongMatchResult{
		Service:    fromInternalServiceName(result.Service),
		Alternates: translateStableSlice(result.Alternates, fromInternalSongScoredMatch),
	}
	if result.Best != nil {
		best := fromInternalSongScoredMatch(*result.Best)
		public.Best = &best
	}
	return public
}

func fromInternalSongResolution(resolution resolve.SongResolution) SongResolution {
	return SongResolution{
		InputURL: resolution.InputURL,
		Parsed:   fromInternalParsedURL(resolution.Parsed),
		Source:   fromInternalCanonicalSong(resolution.Source),
		Matches:  translateStableServiceMap(resolution.Matches, fromInternalSongMatchResult),
	}
}
