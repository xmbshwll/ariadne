package ariadne

import (
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/resolve"
)

func toInternalCanonicalAlbum(album CanonicalAlbum) model.CanonicalAlbum {
	return model.CanonicalAlbum{
		Service:           toInternalServiceName(album.Service),
		SourceID:          album.SourceID,
		SourceURL:         album.SourceURL,
		RegionHint:        album.RegionHint,
		Title:             album.Title,
		NormalizedTitle:   album.NormalizedTitle,
		Artists:           cloneStrings(album.Artists),
		NormalizedArtists: cloneStrings(album.NormalizedArtists),
		ReleaseDate:       album.ReleaseDate,
		Label:             album.Label,
		UPC:               album.UPC,
		TrackCount:        album.TrackCount,
		TotalDurationMS:   album.TotalDurationMS,
		ArtworkURL:        album.ArtworkURL,
		Explicit:          album.Explicit,
		EditionHints:      cloneStrings(album.EditionHints),
		Tracks:            translateSliceToEmpty(album.Tracks, toInternalCanonicalTrack),
	}
}

func fromInternalCanonicalAlbum(album model.CanonicalAlbum) CanonicalAlbum {
	return CanonicalAlbum{
		Service:           fromInternalServiceName(album.Service),
		SourceID:          album.SourceID,
		SourceURL:         album.SourceURL,
		RegionHint:        album.RegionHint,
		Title:             album.Title,
		NormalizedTitle:   album.NormalizedTitle,
		Artists:           cloneStrings(album.Artists),
		NormalizedArtists: cloneStrings(album.NormalizedArtists),
		ReleaseDate:       album.ReleaseDate,
		Label:             album.Label,
		UPC:               album.UPC,
		TrackCount:        album.TrackCount,
		TotalDurationMS:   album.TotalDurationMS,
		ArtworkURL:        album.ArtworkURL,
		Explicit:          album.Explicit,
		EditionHints:      cloneStrings(album.EditionHints),
		Tracks:            translateSliceToEmpty(album.Tracks, fromInternalCanonicalTrack),
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
	return translateNonEmptySlice(albums, toInternalCandidateAlbum)
}

func fromInternalScoredMatch(match resolve.ScoredMatch) ScoredMatch {
	return ScoredMatch{
		URL:       match.URL,
		Score:     match.Score,
		Reasons:   cloneStrings(match.Reasons),
		Candidate: fromInternalCandidateAlbum(match.Candidate),
	}
}

func fromInternalMatchResult(result resolve.MatchResult) MatchResult {
	public := MatchResult{
		Service:    fromInternalServiceName(result.Service),
		Alternates: translateSliceToEmpty(result.Alternates, fromInternalScoredMatch),
	}
	if result.Best != nil {
		best := fromInternalScoredMatch(*result.Best)
		public.Best = &best
	}
	return public
}

func fromInternalResolution(resolution resolve.Resolution) Resolution {
	return Resolution{
		InputURL: resolution.InputURL,
		Parsed:   fromInternalParsedURL(resolution.Parsed),
		Source:   fromInternalCanonicalAlbum(resolution.Source),
		Matches:  translateServiceMap(resolution.Matches, fromInternalMatchResult),
	}
}
