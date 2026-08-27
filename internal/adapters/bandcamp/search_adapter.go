package bandcamp

import (
	"context"
	"fmt"
	"net/url"

	"github.com/xmbshwll/ariadne/internal/adapters/search"
	"github.com/xmbshwll/ariadne/internal/httpx"
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/normalize"
)

const searchHydrationLimit = 8

// SearchAlbumByMetadata searches Bandcamp metadata results and hydrates matching album pages.
// Candidates return untruncated and unranked: Entity Resolution owns ranking
// with the configured Score Signal weights.
func (a *Adapter) SearchAlbumByMetadata(ctx context.Context, album model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
	return a.albumTargetSearch(album).Run(ctx)
}

// SearchSongByMetadata searches Bandcamp metadata results and hydrates matching track pages.
// Candidates return untruncated and unranked: Entity Resolution owns ranking
// with the configured Score Signal weights.
func (a *Adapter) SearchSongByMetadata(ctx context.Context, song model.CanonicalSong) ([]model.CandidateSong, error) {
	return a.songTargetSearch(song).Run(ctx)
}

type BandcampTargetSearch[Candidate any] struct {
	Adapter                *Adapter
	Query                  string
	AutocompleteCandidates func(fuzzySearchResponse) []SearchCandidate
	HtmlCandidates         func([]byte) []SearchCandidate
	Hydrate                func(context.Context, SearchCandidate) (Candidate, error)
	CollectErr             string
}

func (a *Adapter) albumTargetSearch(album model.CanonicalAlbum) BandcampTargetSearch[model.CandidateAlbum] {
	return BandcampTargetSearch[model.CandidateAlbum]{
		Adapter: a,
		Query:   normalize.SearchPrimaryQuery(album.Title, album.Artists),
		AutocompleteCandidates: func(response fuzzySearchResponse) []SearchCandidate {
			return RankSearchCandidates(album, ExtractAutocompleteAlbumSearchCandidates(response))
		},
		HtmlCandidates: func(body []byte) []SearchCandidate {
			return RankSearchCandidates(album, ExtractSearchCandidates(body))
		},
		Hydrate:    a.hydrateBandcampAlbumSearchCandidate,
		CollectErr: "collect bandcamp album candidates",
	}
}

func (a *Adapter) songTargetSearch(song model.CanonicalSong) BandcampTargetSearch[model.CandidateSong] {
	return BandcampTargetSearch[model.CandidateSong]{
		Adapter: a,
		Query:   normalize.SearchPrimaryQuery(song.Title, song.Artists),
		AutocompleteCandidates: func(response fuzzySearchResponse) []SearchCandidate {
			return rankSongSearchCandidates(song, extractAutocompleteSongSearchCandidates(response))
		},
		HtmlCandidates: func(body []byte) []SearchCandidate {
			return rankSongSearchCandidates(song, ExtractSongSearchCandidates(body))
		},
		Hydrate:    a.hydrateBandcampSongSearchCandidate,
		CollectErr: "collect bandcamp song candidates",
	}
}

func bandcampSearchCandidateURL(candidate SearchCandidate) string {
	return candidate.URL
}

func (a *Adapter) hydrateBandcampAlbumSearchCandidate(ctx context.Context, candidate SearchCandidate) (model.CandidateAlbum, error) {
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

func (a *Adapter) hydrateBandcampSongSearchCandidate(ctx context.Context, candidate SearchCandidate) (model.CandidateSong, error) {
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

func (s BandcampTargetSearch[Candidate]) Run(ctx context.Context) ([]Candidate, error) {
	if s.Query == "" {
		return nil, nil
	}

	candidates, err := s.autocomplete(ctx)
	if err != nil || len(candidates) == 0 {
		return s.collectHTML(ctx)
	}

	results, err := s.collect(ctx, candidates)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return s.collectHTML(ctx)
	}
	return results, nil
}

func (s BandcampTargetSearch[Candidate]) collectHTML(ctx context.Context) ([]Candidate, error) {
	candidates, err := s.html(ctx)
	if err != nil {
		return nil, err
	}
	results, err := s.collect(ctx, candidates)
	if err != nil || len(results) > 0 {
		return results, err
	}
	return nil, nil
}

func (s BandcampTargetSearch[Candidate]) collect(ctx context.Context, candidates []SearchCandidate) ([]Candidate, error) {
	results, err := search.CollectCandidates(
		ctx,
		candidates,
		searchHydrationLimit,
		bandcampSearchCandidateURL,
		s.Hydrate,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", s.CollectErr, err)
	}
	return results, nil
}

func (s BandcampTargetSearch[Candidate]) html(ctx context.Context) ([]SearchCandidate, error) {
	searchURL := fmt.Sprintf("%s/search?q=%s", s.Adapter.searchBaseURL, url.QueryEscape(s.Query))
	body, err := s.Adapter.fetchPage(ctx, searchURL)
	if err != nil {
		return nil, fmt.Errorf("fetch bandcamp search page: %w", err)
	}
	return s.HtmlCandidates(body), nil
}

func (s BandcampTargetSearch[Candidate]) autocomplete(ctx context.Context) ([]SearchCandidate, error) {
	searchURL := fmt.Sprintf("%s/api/fuzzysearch/1/app_autocomplete?q=%s", s.Adapter.searchBaseURL, url.QueryEscape(s.Query))
	var response fuzzySearchResponse
	if err := httpx.GetJSON(ctx, httpx.JSONRequest{
		RequestSpec: httpx.RequestSpec{
			Client:       s.Adapter.client,
			URL:          searchURL,
			UserAgent:    httpx.DefaultUserAgent,
			BuildError:   "build bandcamp autocomplete request",
			ExecuteError: "execute bandcamp autocomplete request",
			StatusError:  httpx.StatusError(errUnexpectedBandcampStatus),
		},
		DecodeError:       "decode bandcamp autocomplete response",
		MalformedResponse: errMalformedBandcampSearchResponse,
	}, &response); err != nil {
		return nil, fmt.Errorf("bandcamp autocomplete: %w", err)
	}
	return s.AutocompleteCandidates(response), nil
}
