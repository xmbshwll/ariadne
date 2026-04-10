package ariadne

import (
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/resolve"
	"github.com/xmbshwll/ariadne/internal/score"
)

func toInternalServiceName(service ServiceName) model.ServiceName {
	return model.ServiceName(service)
}

func fromInternalServiceName(service model.ServiceName) ServiceName {
	return ServiceName(service)
}

func toInternalScoreWeights(weights ScoreWeights) score.Weights {
	return score.Weights{
		UPCExact:             weights.UPCExact,
		ISRCStrongOverlap:    weights.ISRCStrongOverlap,
		ISRCPartialScale:     weights.ISRCPartialScale,
		TrackTitleStrong:     weights.TrackTitleStrong,
		TrackTitlePartial:    weights.TrackTitlePartial,
		TitleExact:           weights.TitleExact,
		CoreTitleExact:       weights.CoreTitleExact,
		PrimaryArtistExact:   weights.PrimaryArtistExact,
		ArtistOverlap:        weights.ArtistOverlap,
		TrackCountExact:      weights.TrackCountExact,
		TrackCountNear:       weights.TrackCountNear,
		TrackCountMismatch:   weights.TrackCountMismatch,
		ReleaseDateExact:     weights.ReleaseDateExact,
		ReleaseYearExact:     weights.ReleaseYearExact,
		DurationNear:         weights.DurationNear,
		LabelExact:           weights.LabelExact,
		ExplicitMismatch:     weights.ExplicitMismatch,
		EditionMismatch:      weights.EditionMismatch,
		EditionMarkerPenalty: weights.EditionMarkerPenalty,
	}
}

func fromInternalScoreWeights(weights score.Weights) ScoreWeights {
	return ScoreWeights{
		UPCExact:             weights.UPCExact,
		ISRCStrongOverlap:    weights.ISRCStrongOverlap,
		ISRCPartialScale:     weights.ISRCPartialScale,
		TrackTitleStrong:     weights.TrackTitleStrong,
		TrackTitlePartial:    weights.TrackTitlePartial,
		TitleExact:           weights.TitleExact,
		CoreTitleExact:       weights.CoreTitleExact,
		PrimaryArtistExact:   weights.PrimaryArtistExact,
		ArtistOverlap:        weights.ArtistOverlap,
		TrackCountExact:      weights.TrackCountExact,
		TrackCountNear:       weights.TrackCountNear,
		TrackCountMismatch:   weights.TrackCountMismatch,
		ReleaseDateExact:     weights.ReleaseDateExact,
		ReleaseYearExact:     weights.ReleaseYearExact,
		DurationNear:         weights.DurationNear,
		LabelExact:           weights.LabelExact,
		ExplicitMismatch:     weights.ExplicitMismatch,
		EditionMismatch:      weights.EditionMismatch,
		EditionMarkerPenalty: weights.EditionMarkerPenalty,
	}
}

func toInternalSongScoreWeights(weights SongScoreWeights) score.SongWeights {
	return score.SongWeights{
		ISRCExact:            weights.ISRCExact,
		TitleExact:           weights.TitleExact,
		CoreTitleExact:       weights.CoreTitleExact,
		PrimaryArtistExact:   weights.PrimaryArtistExact,
		ArtistOverlap:        weights.ArtistOverlap,
		DurationNear:         weights.DurationNear,
		AlbumTitleExact:      weights.AlbumTitleExact,
		ReleaseDateExact:     weights.ReleaseDateExact,
		ReleaseYearExact:     weights.ReleaseYearExact,
		TrackNumberExact:     weights.TrackNumberExact,
		ExplicitMismatch:     weights.ExplicitMismatch,
		EditionMismatch:      weights.EditionMismatch,
		EditionMarkerPenalty: weights.EditionMarkerPenalty,
	}
}

func fromInternalSongScoreWeights(weights score.SongWeights) SongScoreWeights {
	return SongScoreWeights{
		ISRCExact:            weights.ISRCExact,
		TitleExact:           weights.TitleExact,
		CoreTitleExact:       weights.CoreTitleExact,
		PrimaryArtistExact:   weights.PrimaryArtistExact,
		ArtistOverlap:        weights.ArtistOverlap,
		DurationNear:         weights.DurationNear,
		AlbumTitleExact:      weights.AlbumTitleExact,
		ReleaseDateExact:     weights.ReleaseDateExact,
		ReleaseYearExact:     weights.ReleaseYearExact,
		TrackNumberExact:     weights.TrackNumberExact,
		ExplicitMismatch:     weights.ExplicitMismatch,
		EditionMismatch:      weights.EditionMismatch,
		EditionMarkerPenalty: weights.EditionMarkerPenalty,
	}
}

func toInternalParsedAlbumURL(parsed ParsedAlbumURL) model.ParsedAlbumURL {
	return model.ParsedAlbumURL{
		Service:      toInternalServiceName(parsed.Service),
		EntityType:   parsed.EntityType,
		ID:           parsed.ID,
		CanonicalURL: parsed.CanonicalURL,
		RegionHint:   parsed.RegionHint,
		RawURL:       parsed.RawURL,
	}
}

func fromInternalParsedAlbumURL(parsed model.ParsedAlbumURL) ParsedAlbumURL {
	return ParsedAlbumURL{
		Service:      fromInternalServiceName(parsed.Service),
		EntityType:   parsed.EntityType,
		ID:           parsed.ID,
		CanonicalURL: parsed.CanonicalURL,
		RegionHint:   parsed.RegionHint,
		RawURL:       parsed.RawURL,
	}
}

func toInternalCanonicalTrack(track CanonicalTrack) model.CanonicalTrack {
	return model.CanonicalTrack{
		DiscNumber:      track.DiscNumber,
		TrackNumber:     track.TrackNumber,
		Title:           track.Title,
		NormalizedTitle: track.NormalizedTitle,
		DurationMS:      track.DurationMS,
		ISRC:            track.ISRC,
		Artists:         append([]string(nil), track.Artists...),
	}
}

func fromInternalCanonicalTrack(track model.CanonicalTrack) CanonicalTrack {
	return CanonicalTrack{
		DiscNumber:      track.DiscNumber,
		TrackNumber:     track.TrackNumber,
		Title:           track.Title,
		NormalizedTitle: track.NormalizedTitle,
		DurationMS:      track.DurationMS,
		ISRC:            track.ISRC,
		Artists:         append([]string(nil), track.Artists...),
	}
}

func toInternalCanonicalAlbum(album CanonicalAlbum) model.CanonicalAlbum {
	tracks := make([]model.CanonicalTrack, 0, len(album.Tracks))
	for _, track := range album.Tracks {
		tracks = append(tracks, toInternalCanonicalTrack(track))
	}
	return model.CanonicalAlbum{
		Service:           toInternalServiceName(album.Service),
		SourceID:          album.SourceID,
		SourceURL:         album.SourceURL,
		RegionHint:        album.RegionHint,
		Title:             album.Title,
		NormalizedTitle:   album.NormalizedTitle,
		Artists:           append([]string(nil), album.Artists...),
		NormalizedArtists: append([]string(nil), album.NormalizedArtists...),
		ReleaseDate:       album.ReleaseDate,
		Label:             album.Label,
		UPC:               album.UPC,
		TrackCount:        album.TrackCount,
		TotalDurationMS:   album.TotalDurationMS,
		ArtworkURL:        album.ArtworkURL,
		Explicit:          album.Explicit,
		EditionHints:      append([]string(nil), album.EditionHints...),
		Tracks:            tracks,
	}
}

func fromInternalCanonicalAlbum(album model.CanonicalAlbum) CanonicalAlbum {
	tracks := make([]CanonicalTrack, 0, len(album.Tracks))
	for _, track := range album.Tracks {
		tracks = append(tracks, fromInternalCanonicalTrack(track))
	}
	return CanonicalAlbum{
		Service:           fromInternalServiceName(album.Service),
		SourceID:          album.SourceID,
		SourceURL:         album.SourceURL,
		RegionHint:        album.RegionHint,
		Title:             album.Title,
		NormalizedTitle:   album.NormalizedTitle,
		Artists:           append([]string(nil), album.Artists...),
		NormalizedArtists: append([]string(nil), album.NormalizedArtists...),
		ReleaseDate:       album.ReleaseDate,
		Label:             album.Label,
		UPC:               album.UPC,
		TrackCount:        album.TrackCount,
		TotalDurationMS:   album.TotalDurationMS,
		ArtworkURL:        album.ArtworkURL,
		Explicit:          album.Explicit,
		EditionHints:      append([]string(nil), album.EditionHints...),
		Tracks:            tracks,
	}
}

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

func toInternalCandidateAlbum(album CandidateAlbum) model.CandidateAlbum {
	return model.CandidateAlbum{
		CanonicalAlbum: toInternalCanonicalAlbum(album.CanonicalAlbum),
		CandidateID:    album.CandidateID,
		MatchURL:       album.MatchURL,
	}
}

func fromInternalCandidateAlbum(album model.CandidateAlbum) CandidateAlbum {
	return CandidateAlbum{
		CanonicalAlbum: fromInternalCanonicalAlbum(album.CanonicalAlbum),
		CandidateID:    album.CandidateID,
		MatchURL:       album.MatchURL,
	}
}

func toInternalCandidateAlbums(albums []CandidateAlbum) []model.CandidateAlbum {
	if len(albums) == 0 {
		return nil
	}
	internal := make([]model.CandidateAlbum, 0, len(albums))
	for _, album := range albums {
		internal = append(internal, toInternalCandidateAlbum(album))
	}
	return internal
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

func fromInternalScoredMatch(match resolve.ScoredMatch) ScoredMatch {
	return ScoredMatch{
		URL:       match.URL,
		Score:     match.Score,
		Reasons:   append([]string(nil), match.Reasons...),
		Candidate: fromInternalCandidateAlbum(match.Candidate),
	}
}

func fromInternalMatchResult(result resolve.MatchResult) MatchResult {
	public := MatchResult{
		Service:    fromInternalServiceName(result.Service),
		Alternates: make([]ScoredMatch, 0, len(result.Alternates)),
	}
	if result.Best != nil {
		best := fromInternalScoredMatch(*result.Best)
		public.Best = &best
	}
	for _, alternate := range result.Alternates {
		public.Alternates = append(public.Alternates, fromInternalScoredMatch(alternate))
	}
	return public
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

func fromInternalResolution(resolution resolve.Resolution) Resolution {
	matches := make(map[ServiceName]MatchResult, len(resolution.Matches))
	for service, match := range resolution.Matches {
		matches[fromInternalServiceName(service)] = fromInternalMatchResult(match)
	}
	return Resolution{
		InputURL: resolution.InputURL,
		Parsed:   fromInternalParsedAlbumURL(resolution.Parsed),
		Source:   fromInternalCanonicalAlbum(resolution.Source),
		Matches:  matches,
	}
}

func fromInternalSongResolution(resolution resolve.SongResolution) SongResolution {
	matches := make(map[ServiceName]SongMatchResult, len(resolution.Matches))
	for service, match := range resolution.Matches {
		matches[fromInternalServiceName(service)] = fromInternalSongMatchResult(match)
	}
	return SongResolution{
		InputURL: resolution.InputURL,
		Parsed:   fromInternalParsedAlbumURL(resolution.Parsed),
		Source:   fromInternalCanonicalSong(resolution.Source),
		Matches:  matches,
	}
}
