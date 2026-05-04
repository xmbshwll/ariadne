package bandcamp

import (
	"context"
	"fmt"
	"net/url"

	"github.com/xmbshwll/ariadne/internal/adapters/adapterutil"
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/score"
)

const (
	searchLimit          = 5
	searchHydrationLimit = 8
)

// SearchByMetadata searches Bandcamp metadata results and hydrates matching album pages.
func (a *Adapter) SearchByMetadata(ctx context.Context, album model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
	results, err := searchBandcampCandidates(
		ctx,
		a,
		adapterutil.TitleAndFirstArtistQuery(album.Title, album.Artists),
		func(response fuzzySearchResponse) []searchCandidate {
			return rankSearchCandidates(album, extractAutocompleteAlbumSearchCandidates(response))
		},
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

// SearchSongByMetadata searches Bandcamp metadata results and hydrates matching track pages.
func (a *Adapter) SearchSongByMetadata(ctx context.Context, song model.CanonicalSong) ([]model.CandidateSong, error) {
	results, err := searchBandcampCandidates(
		ctx,
		a,
		adapterutil.TitleAndFirstArtistQuery(song.Title, song.Artists),
		func(response fuzzySearchResponse) []searchCandidate {
			return rankSongSearchCandidates(song, extractAutocompleteSongSearchCandidates(response))
		},
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

func searchBandcampCandidates[Candidate any](
	ctx context.Context,
	adapter *Adapter,
	query string,
	extractAutocomplete func(fuzzySearchResponse) []searchCandidate,
	extractHTML func([]byte) []searchCandidate,
	hydrate func(context.Context, searchCandidate) (Candidate, error),
	collectErr string,
) ([]Candidate, error) {
	if query == "" {
		return nil, nil
	}

	candidates, err := fetchBandcampAutocompleteCandidates(ctx, adapter, query, extractAutocomplete)
	if err != nil || len(candidates) == 0 {
		candidates, err = fetchBandcampHTMLCandidates(ctx, adapter, query, extractHTML)
		if err != nil {
			return nil, err
		}
	}

	results, err := adapterutil.CollectCandidatesWithContext(
		ctx,
		candidates,
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

func fetchBandcampHTMLCandidates(ctx context.Context, adapter *Adapter, query string, extract func([]byte) []searchCandidate) ([]searchCandidate, error) {
	searchURL := fmt.Sprintf("%s/search?q=%s", adapter.searchBaseURL, url.QueryEscape(query))
	body, err := adapter.fetchPage(ctx, searchURL)
	if err != nil {
		return nil, fmt.Errorf("fetch bandcamp search page: %w", err)
	}
	return extract(body), nil
}

func fetchBandcampAutocompleteCandidates(ctx context.Context, adapter *Adapter, query string, extract func(fuzzySearchResponse) []searchCandidate) ([]searchCandidate, error) {
	searchURL := fmt.Sprintf("%s/api/fuzzysearch/1/app_autocomplete?q=%s", adapter.searchBaseURL, url.QueryEscape(query))
	var response fuzzySearchResponse
	if err := adapterutil.GetJSON(ctx, adapterutil.JSONRequest{
		RequestSpec: adapterutil.RequestSpec{
			Client:       adapter.client,
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
	return extract(response), nil
}

func topRankedCandidates[Ranked any, Candidate any](ranked []Ranked, candidate func(Ranked) Candidate) []Candidate {
	if len(ranked) == 0 {
		return nil
	}
	if len(ranked) > searchLimit {
		ranked = ranked[:searchLimit]
	}
	ordered := make([]Candidate, 0, len(ranked))
	for _, rankedCandidate := range ranked {
		ordered = append(ordered, candidate(rankedCandidate))
	}
	return ordered
}
