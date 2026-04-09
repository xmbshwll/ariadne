package resolve

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/score"
)

var (
	// ErrUnsupportedURL indicates that no registered source adapter recognized the input URL.
	ErrUnsupportedURL = errors.New("unsupported album url")
	// ErrNoSourceAdapters indicates that the resolver was created without source adapters.
	ErrNoSourceAdapters = errors.New("no source adapters configured")
)

// SourceAdapter fetches canonical album metadata from a parsed source URL.
type SourceAdapter interface {
	Service() model.ServiceName
	ParseAlbumURL(raw string) (*model.ParsedAlbumURL, error)
	FetchAlbum(ctx context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalAlbum, error)
}

// TargetAdapter searches one target service for matching albums.
type TargetAdapter interface {
	Service() model.ServiceName
	SearchByUPC(ctx context.Context, upc string) ([]model.CandidateAlbum, error)
	SearchByISRC(ctx context.Context, isrcs []string) ([]model.CandidateAlbum, error)
	SearchByMetadata(ctx context.Context, album model.CanonicalAlbum) ([]model.CandidateAlbum, error)
}

// ScoredMatch is one scored candidate exposed by the resolver.
type ScoredMatch struct {
	URL       string
	Score     int
	Reasons   []string
	Candidate model.CandidateAlbum
}

// MatchResult is the resolver output for one target service.
type MatchResult struct {
	Service    model.ServiceName
	Best       *ScoredMatch
	Alternates []ScoredMatch
}

// Resolution contains the source album and ranked target matches collected by the resolver.
type Resolution struct {
	InputURL string
	Parsed   model.ParsedAlbumURL
	Source   model.CanonicalAlbum
	Matches  map[model.ServiceName]MatchResult
}

// Resolver coordinates source parsing, source fetching, and layered target search.
type Resolver struct {
	sources []SourceAdapter
	targets []TargetAdapter
	weights score.Weights
}

// New creates a resolver from registered source and target adapters.
func New(sources []SourceAdapter, targets []TargetAdapter, weights score.Weights) *Resolver {
	return &Resolver{
		sources: append([]SourceAdapter(nil), sources...),
		targets: append([]TargetAdapter(nil), targets...),
		weights: weights,
	}
}

// ResolveAlbum parses an input album URL, fetches the canonical source album,
// then collects and ranks candidates from every target adapter except the source service.
func (r *Resolver) ResolveAlbum(ctx context.Context, inputURL string) (*Resolution, error) {
	if len(r.sources) == 0 {
		return nil, ErrNoSourceAdapters
	}

	sourceAdapter, parsed, err := r.parseSource(inputURL)
	if err != nil {
		return nil, err
	}

	sourceAlbum, err := sourceAdapter.FetchAlbum(ctx, *parsed)
	if err != nil {
		return nil, fmt.Errorf("fetch source album with %s: %w", sourceAdapter.Service(), err)
	}

	targets := r.targetSearchesFor(sourceAlbum.Service)
	resolution := &Resolution{
		InputURL: inputURL,
		Parsed:   *parsed,
		Source:   *sourceAlbum,
		Matches:  make(map[model.ServiceName]MatchResult, len(targets)),
	}

	group, groupCtx := errgroup.WithContext(ctx)
	var matchesMu sync.Mutex

	for _, target := range targets {
		group.Go(func() error {
			candidates, err := r.collectCandidates(groupCtx, target, *sourceAlbum)
			if err != nil {
				return fmt.Errorf("collect candidates from %s: %w", target.Service(), err)
			}
			ranking := score.RankAlbums(*sourceAlbum, candidates, r.weights)

			matchesMu.Lock()
			resolution.Matches[target.Service()] = matchResultFromRanking(target.Service(), ranking)
			matchesMu.Unlock()
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return nil, fmt.Errorf("resolve target searches: %w", err)
	}
	return resolution, nil
}

func (r *Resolver) targetSearchesFor(sourceService model.ServiceName) []TargetAdapter {
	targets := make([]TargetAdapter, 0, len(r.targets))
	for _, target := range r.targets {
		if target.Service() == sourceService {
			continue
		}
		targets = append(targets, target)
	}
	return targets
}

func (r *Resolver) parseSource(inputURL string) (SourceAdapter, *model.ParsedAlbumURL, error) {
	for _, source := range r.sources {
		parsed, err := source.ParseAlbumURL(inputURL)
		if err != nil || parsed == nil {
			continue
		}
		return source, parsed, nil
	}
	return nil, nil, fmt.Errorf("%w: %s", ErrUnsupportedURL, inputURL)
}

func (r *Resolver) collectCandidates(ctx context.Context, target TargetAdapter, source model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
	combined := []model.CandidateAlbum{}
	seen := map[string]struct{}{}

	if source.UPC != "" {
		candidates, err := target.SearchByUPC(ctx, source.UPC)
		if err != nil {
			//nolint:wrapcheck // Preserve target adapter errors without adding another wrapper layer.
			return nil, err
		}
		combined = appendUniqueAlbumCandidates(combined, seen, candidates)
	}

	isrcs := collectISRCs(source)
	if len(isrcs) > 0 {
		candidates, err := target.SearchByISRC(ctx, isrcs)
		if err != nil {
			//nolint:wrapcheck // Preserve target adapter errors without adding another wrapper layer.
			return nil, err
		}
		combined = appendUniqueAlbumCandidates(combined, seen, candidates)
	}

	metadataCandidates, err := target.SearchByMetadata(ctx, source)
	if err != nil {
		//nolint:wrapcheck // Preserve target adapter errors without adding another wrapper layer.
		return nil, err
	}
	combined = appendUniqueAlbumCandidates(combined, seen, metadataCandidates)

	return combined, nil
}

func appendUniqueAlbumCandidates(dst []model.CandidateAlbum, seen map[string]struct{}, candidates []model.CandidateAlbum) []model.CandidateAlbum {
	for _, candidate := range candidates {
		key := candidateKey(candidate)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		dst = append(dst, candidate)
	}
	return dst
}

func collectISRCs(album model.CanonicalAlbum) []string {
	isrcs := make([]string, 0, len(album.Tracks))
	seen := make(map[string]struct{}, len(album.Tracks))
	for _, track := range album.Tracks {
		if track.ISRC == "" {
			continue
		}
		if _, ok := seen[track.ISRC]; ok {
			continue
		}
		seen[track.ISRC] = struct{}{}
		isrcs = append(isrcs, track.ISRC)
	}
	return isrcs
}

func candidateKey(candidate model.CandidateAlbum) string {
	if candidate.CandidateID != "" {
		return string(candidate.Service) + ":id:" + candidate.CandidateID
	}
	return string(candidate.Service) + ":url:" + candidate.MatchURL
}

func matchResultFromRanking(service model.ServiceName, ranking score.Ranking) MatchResult {
	result := MatchResult{
		Service:    service,
		Alternates: make([]ScoredMatch, 0),
	}
	if ranking.Best == nil {
		return result
	}

	best := toScoredMatch(*ranking.Best)
	result.Best = &best
	for _, ranked := range ranking.Ranked[1:] {
		result.Alternates = append(result.Alternates, toScoredMatch(ranked))
	}
	return result
}

func toScoredMatch(ranked score.RankedCandidate) ScoredMatch {
	return ScoredMatch{
		URL:       ranked.Candidate.MatchURL,
		Score:     ranked.Score,
		Reasons:   append([]string(nil), ranked.Reasons...),
		Candidate: ranked.Candidate,
	}
}
