package main

import "github.com/xmbshwll/ariadne"

func newCLIAlbum(album ariadne.CanonicalAlbum) cliAlbum {
	return cliAlbum{
		Service:      string(album.Service),
		ID:           album.SourceID,
		URL:          album.SourceURL,
		RegionHint:   album.RegionHint,
		Title:        album.Title,
		Artists:      append([]string(nil), album.Artists...),
		ReleaseDate:  album.ReleaseDate,
		Label:        album.Label,
		UPC:          album.UPC,
		TrackCount:   album.TrackCount,
		ArtworkURL:   album.ArtworkURL,
		EditionHints: append([]string(nil), album.EditionHints...),
	}
}

func newCLISong(song ariadne.CanonicalSong) cliSong {
	return cliSong{
		Service:      string(song.Service),
		ID:           song.SourceID,
		URL:          song.SourceURL,
		RegionHint:   song.RegionHint,
		Title:        song.Title,
		Artists:      append([]string(nil), song.Artists...),
		DurationMS:   song.DurationMS,
		ISRC:         song.ISRC,
		Explicit:     song.Explicit,
		DiscNumber:   song.DiscNumber,
		TrackNumber:  song.TrackNumber,
		AlbumID:      song.AlbumID,
		AlbumTitle:   song.AlbumTitle,
		ReleaseDate:  song.ReleaseDate,
		ArtworkURL:   song.ArtworkURL,
		EditionHints: append([]string(nil), song.EditionHints...),
	}
}

func newCLIMatchResult(result ariadne.MatchResult) cliMatchResult {
	output := cliMatchResult{
		Found:      result.Best != nil,
		Summary:    "not_found",
		Alternates: make([]cliMatch, 0, len(result.Alternates)),
	}
	if result.Best != nil {
		best := newCLIMatch(*result.Best)
		output.Best = &best
		output.Summary = scoreSummary(result.Best.Score)
	}
	for _, alternate := range result.Alternates {
		output.Alternates = append(output.Alternates, newCLIMatch(alternate))
	}
	return output
}

func newCLISongMatchResult(result ariadne.SongMatchResult) cliSongMatchResult {
	output := cliSongMatchResult{
		Found:      result.Best != nil,
		Summary:    "not_found",
		Alternates: make([]cliSongMatch, 0, len(result.Alternates)),
	}
	if result.Best != nil {
		best := newCLISongMatch(*result.Best)
		output.Best = &best
		output.Summary = scoreSummary(result.Best.Score)
	}
	for _, alternate := range result.Alternates {
		output.Alternates = append(output.Alternates, newCLISongMatch(alternate))
	}
	return output
}

func scoreSummary(score int) string {
	return string(ariadne.MatchStrengthForScore(score))
}

func newCLIMatch(match ariadne.ScoredMatch) cliMatch {
	return cliMatch{
		URL:         match.URL,
		Score:       match.Score,
		Reasons:     append([]string(nil), match.Reasons...),
		AlbumID:     match.Candidate.CandidateID,
		RegionHint:  match.Candidate.RegionHint,
		Title:       match.Candidate.Title,
		Artists:     append([]string(nil), match.Candidate.Artists...),
		ReleaseDate: match.Candidate.ReleaseDate,
		UPC:         match.Candidate.UPC,
	}
}

func newCLISongMatch(match ariadne.SongScoredMatch) cliSongMatch {
	return cliSongMatch{
		URL:         match.URL,
		Score:       match.Score,
		Reasons:     append([]string(nil), match.Reasons...),
		SongID:      match.Candidate.CandidateID,
		RegionHint:  match.Candidate.RegionHint,
		Title:       match.Candidate.Title,
		Artists:     append([]string(nil), match.Candidate.Artists...),
		DurationMS:  match.Candidate.DurationMS,
		ISRC:        match.Candidate.ISRC,
		AlbumTitle:  match.Candidate.AlbumTitle,
		TrackNumber: match.Candidate.TrackNumber,
		ReleaseDate: match.Candidate.ReleaseDate,
	}
}
