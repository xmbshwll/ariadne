package bandcamp

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/xmbshwll/ariadne/internal/adapters/adapterutil"
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/score"
)

const (
	searchLimit          = 5
	searchHydrationLimit = 8
)

// SearchByMetadata searches Bandcamp HTML results and hydrates matching album pages.
func (a *Adapter) SearchByMetadata(ctx context.Context, album model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
	results, err := searchBandcampCandidates(
		ctx,
		a,
		metadataQuery(album),
		func(body []byte) []searchCandidate {
			return rankSearchCandidates(album, extractSearchCandidates(body))
		},
		a.hydrateBandcampAlbumSearchCandidate,
		"collect bandcamp album candidates",
	)
	if err != nil {
		return nil, err
	}

	ranking := score.RankAlbums(album, results, score.DefaultWeights())
	return topRankedCandidates(ranking.Ranked, func(candidate score.RankedCandidate) model.CandidateAlbum {
		return candidate.Candidate
	}), nil
}

// SearchSongByMetadata searches Bandcamp HTML results and hydrates matching track pages.
func (a *Adapter) SearchSongByMetadata(ctx context.Context, song model.CanonicalSong) ([]model.CandidateSong, error) {
	results, err := searchBandcampCandidates(
		ctx,
		a,
		songMetadataQuery(song),
		func(body []byte) []searchCandidate {
			return rankSongSearchCandidates(song, extractSongSearchCandidates(body))
		},
		a.hydrateBandcampSongSearchCandidate,
		"collect bandcamp song candidates",
	)
	if err != nil {
		return nil, err
	}

	ranking := score.RankSongs(song, results, score.DefaultSongWeights())
	return topRankedCandidates(ranking.Ranked, func(candidate score.RankedSongCandidate) model.CandidateSong {
		return candidate.Candidate
	}), nil
}

func bandcampSearchCandidateURL(candidate searchCandidate) string {
	return candidate.URL
}

func (a *Adapter) hydrateBandcampAlbumSearchCandidate(ctx context.Context, candidate searchCandidate) (model.CandidateAlbum, error) {
	canonical, err := a.fetchAlbumPage(ctx, candidate.URL)
	if err != nil {
		return model.CandidateAlbum{}, fmt.Errorf("hydrate bandcamp album %s: %w", candidate.URL, err)
	}
	return model.CandidateAlbum{
		CanonicalAlbum: *canonical,
		CandidateID:    canonical.SourceID,
		MatchURL:       canonical.SourceURL,
	}, nil
}

func (a *Adapter) hydrateBandcampSongSearchCandidate(ctx context.Context, candidate searchCandidate) (model.CandidateSong, error) {
	canonical, err := a.fetchSongPage(ctx, candidate.URL)
	if err != nil {
		return model.CandidateSong{}, fmt.Errorf("hydrate bandcamp song %s: %w", candidate.URL, err)
	}
	return model.CandidateSong{
		CanonicalSong: *canonical,
		CandidateID:   canonical.SourceID,
		MatchURL:      canonical.SourceURL,
	}, nil
}

func searchBandcampCandidates[Candidate any](ctx context.Context, adapter *Adapter, query string, extract func([]byte) []searchCandidate, hydrate func(context.Context, searchCandidate) (Candidate, error), collectErr string) ([]Candidate, error) {
	if query == "" {
		return nil, nil
	}

	searchURL := fmt.Sprintf("%s/search?q=%s", adapter.searchBaseURL, url.QueryEscape(query))
	body, err := adapter.fetchPage(ctx, searchURL)
	if err != nil {
		return nil, fmt.Errorf("fetch bandcamp search page: %w", err)
	}

	results, err := adapterutil.CollectCandidatesWithContext(
		ctx,
		extract(body),
		searchHydrationLimit,
		bandcampSearchCandidateURL,
		hydrate,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", collectErr, err)
	}
	if len(results) == 0 {
		return nil, nil
	}
	return results, nil
}

func metadataQuery(album model.CanonicalAlbum) string {
	parts := make([]string, 0, 2)
	if album.Title != "" {
		parts = append(parts, album.Title)
	}
	if len(album.Artists) > 0 {
		parts = append(parts, album.Artists[0])
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func songMetadataQuery(song model.CanonicalSong) string {
	parts := make([]string, 0, 2)
	if song.Title != "" {
		parts = append(parts, song.Title)
	}
	if len(song.Artists) > 0 {
		parts = append(parts, song.Artists[0])
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func topRankedCandidates[Ranked any, Candidate any](ranked []Ranked, candidate func(Ranked) Candidate) []Candidate {
	ordered := make([]Candidate, 0, minInt(len(ranked), searchLimit))
	for i, rankedCandidate := range ranked {
		if i >= searchLimit {
			break
		}
		ordered = append(ordered, candidate(rankedCandidate))
	}
	return ordered
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}
