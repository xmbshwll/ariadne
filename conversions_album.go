package ariadne

import (
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/resolve"
)

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

func fromInternalResolution(resolution resolve.Resolution) Resolution {
	matches := make(map[ServiceName]MatchResult, len(resolution.Matches))
	for service, match := range resolution.Matches {
		matches[fromInternalServiceName(service)] = fromInternalMatchResult(match)
	}
	return Resolution{
		InputURL: resolution.InputURL,
		Parsed:   fromInternalParsedURL(resolution.Parsed),
		Source:   fromInternalCanonicalAlbum(resolution.Source),
		Matches:  matches,
	}
}
