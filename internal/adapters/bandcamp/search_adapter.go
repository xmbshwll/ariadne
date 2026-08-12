package bandcamp

import (
	"context"
	"fmt"
	"net/url"

	"github.com/xmbshwll/ariadne/internal/adapters/adapterutil"
	"github.com/xmbshwll/ariadne/internal/model"
)

const searchHydrationLimit = 8

// SearchByMetadata searches Bandcamp metadata results and hydrates matching album pages.
// Candidates return untruncated and unranked: Entity Resolution owns ranking
// with the configured Score Signal weights.
func (a *Adapter) SearchByMetadata(ctx context.Context, album model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
	return a.albumTargetSearch(album).run(ctx)
}

// SearchSongByMetadata searches Bandcamp metadata results and hydrates matching track pages.
// Candidates return untruncated and unranked: Entity Resolution owns ranking
// with the configured Score Signal weights.
func (a *Adapter) SearchSongByMetadata(ctx context.Context, song model.CanonicalSong) ([]model.CandidateSong, error) {
	return a.songTargetSearch(song).run(ctx)
}

type bandcampTargetSearch[Candidate any] struct {
	adapter                *Adapter
	query                  string
	autocompleteCandidates func(fuzzySearchResponse) []searchCandidate
	htmlCandidates         func([]byte) []searchCandidate
	hydrate                func(context.Context, searchCandidate) (Candidate, error)
	collectErr             string
}

func (a *Adapter) albumTargetSearch(album model.CanonicalAlbum) bandcampTargetSearch[model.CandidateAlbum] {
	return bandcampTargetSearch[model.CandidateAlbum]{
		adapter: a,
		query:   adapterutil.PrimaryMetadataQuery(album.Title, album.Artists),
		autocompleteCandidates: func(response fuzzySearchResponse) []searchCandidate {
			return rankSearchCandidates(album, extractAutocompleteAlbumSearchCandidates(response))
		},
		htmlCandidates: func(body []byte) []searchCandidate {
			return rankSearchCandidates(album, extractSearchCandidates(body))
		},
		hydrate:    a.hydrateBandcampAlbumSearchCandidate,
		collectErr: "collect bandcamp album candidates",
	}
}

func (a *Adapter) songTargetSearch(song model.CanonicalSong) bandcampTargetSearch[model.CandidateSong] {
	return bandcampTargetSearch[model.CandidateSong]{
		adapter: a,
		query:   adapterutil.PrimaryMetadataQuery(song.Title, song.Artists),
		autocompleteCandidates: func(response fuzzySearchResponse) []searchCandidate {
			return rankSongSearchCandidates(song, extractAutocompleteSongSearchCandidates(response))
		},
		htmlCandidates: func(body []byte) []searchCandidate {
			return rankSongSearchCandidates(song, extractSongSearchCandidates(body))
		},
		hydrate:    a.hydrateBandcampSongSearchCandidate,
		collectErr: "collect bandcamp song candidates",
	}
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

func (s bandcampTargetSearch[Candidate]) run(ctx context.Context) ([]Candidate, error) {
	if s.query == "" {
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

func (s bandcampTargetSearch[Candidate]) collectHTML(ctx context.Context) ([]Candidate, error) {
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

func (s bandcampTargetSearch[Candidate]) collect(ctx context.Context, candidates []searchCandidate) ([]Candidate, error) {
	results, err := adapterutil.CollectCandidates(
		ctx,
		candidates,
		searchHydrationLimit,
		bandcampSearchCandidateURL,
		s.hydrate,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", s.collectErr, err)
	}
	return results, nil
}

func (s bandcampTargetSearch[Candidate]) html(ctx context.Context) ([]searchCandidate, error) {
	searchURL := fmt.Sprintf("%s/search?q=%s", s.adapter.searchBaseURL, url.QueryEscape(s.query))
	body, err := s.adapter.fetchPage(ctx, searchURL)
	if err != nil {
		return nil, fmt.Errorf("fetch bandcamp search page: %w", err)
	}
	return s.htmlCandidates(body), nil
}

func (s bandcampTargetSearch[Candidate]) autocomplete(ctx context.Context) ([]searchCandidate, error) {
	searchURL := fmt.Sprintf("%s/api/fuzzysearch/1/app_autocomplete?q=%s", s.adapter.searchBaseURL, url.QueryEscape(s.query))
	var response fuzzySearchResponse
	if err := adapterutil.GetJSON(ctx, adapterutil.JSONRequest{
		RequestSpec: adapterutil.RequestSpec{
			Client:       s.adapter.client,
			URL:          searchURL,
			UserAgent:    adapterutil.DefaultUserAgent,
			BuildError:   "build bandcamp autocomplete request",
			ExecuteError: "execute bandcamp autocomplete request",
			StatusError:  adapterutil.StatusError(errUnexpectedBandcampStatus),
		},
		DecodeError:       "decode bandcamp autocomplete response",
		MalformedResponse: errMalformedBandcampSearchResponse,
	}, &response); err != nil {
		return nil, fmt.Errorf("bandcamp autocomplete: %w", err)
	}
	return s.autocompleteCandidates(response), nil
}
