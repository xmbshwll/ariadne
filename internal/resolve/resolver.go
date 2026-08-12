package resolve

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/score"
)

var (
	// ErrUnsupportedURL indicates that no registered source adapter recognized the input URL.
	ErrUnsupportedURL = errors.New("unsupported url")
	// ErrNoSourceAdapters indicates that the resolver was created without source adapters.
	ErrNoSourceAdapters = errors.New("no source adapters configured")
)

// SourceAdapter fetches canonical album metadata from a parsed source URL.
type SourceAdapter interface {
	Service() model.ServiceName
	ParseAlbumURL(raw string) (*model.ParsedAlbumURL, error)
	FetchAlbum(ctx context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalAlbum, error)
}

// TargetAdapter identifies one album target Music Service.
type TargetAdapter interface {
	Service() model.ServiceName
}

// UPCSearcher searches album targets by UPC.
type UPCSearcher interface {
	SearchByUPC(ctx context.Context, upc string) ([]model.CandidateAlbum, error)
}

// ISRCSearcher searches album targets by track ISRCs.
type ISRCSearcher interface {
	SearchByISRC(ctx context.Context, isrcs []string) ([]model.CandidateAlbum, error)
}

// MetadataSearcher searches album targets by canonical metadata.
type MetadataSearcher interface {
	SearchByMetadata(ctx context.Context, album model.CanonicalAlbum) ([]model.CandidateAlbum, error)
}

// ScoredMatchOf is one scored candidate exposed by the resolver.
type ScoredMatchOf[C any] struct {
	URL       string
	Score     int
	Reasons   []string
	Candidate C
}

// MatchResultOf is the resolver output for one target service. When the Target
// Search for this service failed, Err carries the failure and Best/Alternates
// are empty; other services are unaffected.
type MatchResultOf[C any] struct {
	Service    model.ServiceName
	Best       *ScoredMatchOf[C]
	Alternates []ScoredMatchOf[C]
	Err        error
}

// ResolutionOf contains the source entity and ranked target matches collected by the resolver.
type ResolutionOf[P, E, C any] struct {
	InputURL string
	Parsed   P
	Source   E
	Matches  map[model.ServiceName]MatchResultOf[C]
}

type (
	// ScoredMatch is one scored album candidate.
	ScoredMatch = ScoredMatchOf[model.CandidateAlbum]
	// MatchResult is the album resolver output for one target service.
	MatchResult = MatchResultOf[model.CandidateAlbum]
	// Resolution is the album resolver output.
	Resolution = ResolutionOf[model.ParsedAlbumURL, model.CanonicalAlbum, model.CandidateAlbum]
)

const albumEntityLabel = "album"

type serviceAdapter interface {
	Service() model.ServiceName
}

// Resolver coordinates album Source Input, Runtime Hydration, Target Search, and Identifier Enrichment.
type Resolver struct {
	policy albumEntityResolutionPolicy
}

type albumEntityResolutionPolicy struct {
	sourceAdapters          []SourceAdapter
	targetAdapters          []TargetAdapter
	weights                 score.Weights
	collectTargetCandidates func(context.Context, TargetAdapter, model.CanonicalAlbum) ([]model.CandidateAlbum, error)
	afterTargetMatches      func(context.Context, []TargetAdapter, model.CanonicalAlbum, map[model.ServiceName]MatchResult)
}

func newAlbumEntityResolutionPolicy(sources []SourceAdapter, targets []TargetAdapter, weights score.Weights) albumEntityResolutionPolicy {
	enrichment := newAppleMusicEnrichmentPolicy(weights)
	return albumEntityResolutionPolicy{
		sourceAdapters:          append([]SourceAdapter(nil), sources...),
		targetAdapters:          append([]TargetAdapter(nil), targets...),
		weights:                 weights,
		collectTargetCandidates: enrichment.collectTargetCandidates,
		afterTargetMatches:      enrichment.apply,
	}
}

// New creates a resolver from registered source and target adapters.
func New(sources []SourceAdapter, targets []TargetAdapter, weights score.Weights) *Resolver {
	return &Resolver{policy: newAlbumEntityResolutionPolicy(sources, targets, weights)}
}

// ResolveAlbum parses an input album URL, fetches the canonical source album,
// then collects and ranks candidates from every target adapter except the source
// service. A failing target does not abort the resolution: its MatchResult
// carries the error in Err while other targets resolve normally.
func (r *Resolver) ResolveAlbum(ctx context.Context, inputURL string) (*Resolution, error) {
	return r.policy.resolve(ctx, inputURL)
}

func (p albumEntityResolutionPolicy) resolve(ctx context.Context, inputURL string) (*Resolution, error) {
	source, err := p.resolveSourceInput(ctx, inputURL)
	if err != nil {
		return nil, err
	}

	targets := excludeTargetService(p.targetAdapters, source.Entity.Service)
	matches := p.resolveTargetMatches(ctx, targets, source.Entity)

	p.afterTargetMatches(ctx, targets, source.Entity, matches)

	return &Resolution{
		InputURL: inputURL,
		Parsed:   source.Parsed,
		Source:   source.Entity,
		Matches:  matches,
	}, nil
}

// albumSourceInput pairs the recognized Source Input with its hydrated album.
type albumSourceInput struct {
	Parsed model.ParsedAlbumURL
	Entity model.CanonicalAlbum
}

func (p albumEntityResolutionPolicy) resolveSourceInput(ctx context.Context, inputURL string) (albumSourceInput, error) {
	return resolveAlbumSourceInput(ctx, p.sourceAdapters, inputURL)
}

func resolveAlbumSourceInput(ctx context.Context, sources []SourceAdapter, inputURL string) (albumSourceInput, error) {
	parsed, adapter, err := recognizeSourceInput(sources, inputURL, func(source SourceAdapter) (*model.ParsedAlbumURL, error) {
		return source.ParseAlbumURL(inputURL)
	})
	if err != nil {
		return albumSourceInput{}, err
	}

	album, err := hydrateSourceInput(ctx, adapter, albumEntityLabel, errNilSourceAlbum,
		func(ctx context.Context) (*model.CanonicalAlbum, error) {
			return adapter.FetchAlbum(ctx, *parsed)
		})
	if err != nil {
		return albumSourceInput{}, err
	}

	return albumSourceInput{Parsed: *parsed, Entity: *album}, nil
}

func (p albumEntityResolutionPolicy) resolveTargetMatches(ctx context.Context, targets []TargetAdapter, source model.CanonicalAlbum) map[model.ServiceName]MatchResult {
	matches := make(map[model.ServiceName]MatchResult, len(targets))
	var matchesMu sync.Mutex

	resolveTargetsConcurrently(ctx, targets, func(targetCtx context.Context, target TargetAdapter) {
		var result MatchResult
		candidates, err := p.collectTargetCandidates(targetCtx, target, source)
		if err != nil {
			result = MatchResult{Service: target.Service(), Err: fmt.Errorf("collect candidates: %w", err)}
		} else {
			result = albumMatchResultFromRanking(target.Service(), score.RankAlbums(source, candidates, p.weights))
		}

		matchesMu.Lock()
		matches[target.Service()] = result
		matchesMu.Unlock()
	})
	return matches
}

func excludeTargetService[T serviceAdapter](targets []T, sourceService model.ServiceName) []T {
	filtered := make([]T, 0, len(targets))
	for _, target := range targets {
		if target.Service() == sourceService {
			continue
		}
		filtered = append(filtered, target)
	}
	return filtered
}

// resolveTargetsConcurrently runs resolve for every target without canceling
// siblings: one failing Target Search must not affect the others.
func resolveTargetsConcurrently[T serviceAdapter](ctx context.Context, targets []T, resolve func(context.Context, T)) {
	var group sync.WaitGroup
	for _, target := range targets {
		group.Go(func() {
			resolve(ctx, target)
		})
	}
	group.Wait()
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

func albumCandidateKey(candidate model.CandidateAlbum) string {
	if candidate.CandidateID != "" {
		return string(candidate.Service) + ":id:" + candidate.CandidateID
	}
	return string(candidate.Service) + ":url:" + candidate.MatchURL
}

func albumMatchResultFromRanking(service model.ServiceName, ranking score.Ranking[model.CandidateAlbum]) MatchResult {
	return matchResultFromRanking(service, ranking, func(candidate model.CandidateAlbum) string { return candidate.MatchURL })
}

func matchResultFromRanking[C any](service model.ServiceName, ranking score.Ranking[C], urlOf func(C) string) MatchResultOf[C] {
	result := MatchResultOf[C]{
		Service:    service,
		Alternates: make([]ScoredMatchOf[C], 0),
	}
	if ranking.Best == nil {
		return result
	}

	best := toScoredMatch(*ranking.Best, urlOf)
	result.Best = &best
	for _, ranked := range ranking.Ranked[1:] {
		result.Alternates = append(result.Alternates, toScoredMatch(ranked, urlOf))
	}
	return result
}

func toScoredMatch[C any](ranked score.Ranked[C], urlOf func(C) string) ScoredMatchOf[C] {
	return ScoredMatchOf[C]{
		URL:       urlOf(ranked.Candidate),
		Score:     ranked.Score,
		Reasons:   append([]string(nil), ranked.Reasons...),
		Candidate: ranked.Candidate,
	}
}
