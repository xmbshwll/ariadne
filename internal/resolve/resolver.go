package resolve

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/sync/errgroup"

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

const albumEntityLabel = "album"

type serviceAdapter interface {
	Service() model.ServiceName
}

// Resolver coordinates album Source Input, Runtime Hydration, Target Search, and Identifier Enrichment.
type Resolver struct {
	policy albumEntityResolutionPolicy
}

type albumEntityResolutionPolicy struct {
	sourceAdapters   []SourceAdapter
	targetAdapters   []TargetAdapter
	weights          score.Weights
	appleMusicPolicy appleMusicEnrichmentPolicy
}

func newAlbumEntityResolutionPolicy(sources []SourceAdapter, targets []TargetAdapter, weights score.Weights) albumEntityResolutionPolicy {
	return albumEntityResolutionPolicy{
		sourceAdapters:   append([]SourceAdapter(nil), sources...),
		targetAdapters:   append([]TargetAdapter(nil), targets...),
		weights:          weights,
		appleMusicPolicy: newAppleMusicEnrichmentPolicy(weights),
	}
}

// New creates a resolver from registered source and target adapters.
func New(sources []SourceAdapter, targets []TargetAdapter, weights score.Weights) *Resolver {
	return &Resolver{policy: newAlbumEntityResolutionPolicy(sources, targets, weights)}
}

// ResolveAlbum parses an input album URL, fetches the canonical source album,
// then collects and ranks candidates from every target adapter except the source service.
func (r *Resolver) ResolveAlbum(ctx context.Context, inputURL string) (*Resolution, error) {
	return r.policy.resolve(ctx, inputURL)
}

func (p albumEntityResolutionPolicy) resolve(ctx context.Context, inputURL string) (*Resolution, error) {
	return resolveEntity[model.ParsedAlbumURL, model.CanonicalAlbum, TargetAdapter, MatchResult, Resolution](
		ctx,
		inputURL,
		p.resolveSourceInput,
		p.sourceService,
		p.targetAdapters,
		p.resolveTargetMatches,
		p.afterTargetMatches,
		p.resolution,
	)
}

func (p albumEntityResolutionPolicy) resolveSourceInput(ctx context.Context, inputURL string) (sourceInput[model.ParsedAlbumURL, model.CanonicalAlbum], error) {
	return resolveAlbumSourceInput(ctx, p.sourceAdapters, inputURL)
}

func resolveAlbumSourceInput(ctx context.Context, sources []SourceAdapter, inputURL string) (sourceInput[model.ParsedAlbumURL, model.CanonicalAlbum], error) {
	return resolveSourceInput(
		ctx,
		sources,
		inputURL,
		func(source SourceAdapter, raw string) (*model.ParsedAlbumURL, error) {
			return source.ParseAlbumURL(raw)
		},
		func(ctx context.Context, source SourceAdapter, parsed model.ParsedAlbumURL) (*model.CanonicalAlbum, error) {
			return source.FetchAlbum(ctx, parsed)
		},
		albumEntityLabel,
		errNilSourceAlbum,
	)
}

func (p albumEntityResolutionPolicy) sourceService(source model.CanonicalAlbum) model.ServiceName {
	return source.Service
}

func (p albumEntityResolutionPolicy) resolveTargetMatches(ctx context.Context, targets []TargetAdapter, source model.CanonicalAlbum) (map[model.ServiceName]MatchResult, error) {
	matches, err := resolveTargetMatches(
		ctx,
		targets,
		source,
		p.appleMusicPolicy.collectTargetCandidates,
		func(source model.CanonicalAlbum, candidates []model.CandidateAlbum) score.Ranking {
			return score.RankAlbums(source, candidates, p.weights)
		},
		albumMatchResultFromRanking,
		"candidates",
	)
	if err != nil {
		return nil, fmt.Errorf("resolve target searches: %w", err)
	}
	return matches, nil
}

func (p albumEntityResolutionPolicy) afterTargetMatches(ctx context.Context, targets []TargetAdapter, source model.CanonicalAlbum, matches map[model.ServiceName]MatchResult) error {
	if err := p.appleMusicPolicy.apply(ctx, targets, source, matches); err != nil {
		return fmt.Errorf("resolve apple music cascaded search: %w", err)
	}
	return nil
}

func (p albumEntityResolutionPolicy) resolution(inputURL string, source sourceInput[model.ParsedAlbumURL, model.CanonicalAlbum], matches map[model.ServiceName]MatchResult) *Resolution {
	return &Resolution{
		InputURL: inputURL,
		Parsed:   source.Parsed,
		Source:   source.Entity,
		Matches:  matches,
	}
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

func resolveTargetsConcurrently[T serviceAdapter](ctx context.Context, targets []T, resolve func(context.Context, T) error) error {
	group, groupCtx := errgroup.WithContext(ctx)
	for _, target := range targets {
		group.Go(func() error {
			return resolve(groupCtx, target)
		})
	}
	//nolint:wrapcheck // Preserve worker errors without adding another wrapper layer.
	return group.Wait()
}

func appendUniqueByKey[T any](dst []T, seen map[string]struct{}, items []T, keyFunc func(T) string) []T {
	for _, item := range items {
		key := keyFunc(item)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		dst = append(dst, item)
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

func albumCandidateKey(candidate model.CandidateAlbum) string {
	if candidate.CandidateID != "" {
		return string(candidate.Service) + ":id:" + candidate.CandidateID
	}
	return string(candidate.Service) + ":url:" + candidate.MatchURL
}

func albumMatchResultFromRanking(service model.ServiceName, ranking score.Ranking) MatchResult {
	result := MatchResult{
		Service:    service,
		Alternates: make([]ScoredMatch, 0),
	}
	if ranking.Best == nil {
		return result
	}

	best := toAlbumScoredMatch(*ranking.Best)
	result.Best = &best
	for _, ranked := range ranking.Ranked[1:] {
		result.Alternates = append(result.Alternates, toAlbumScoredMatch(ranked))
	}
	return result
}

func toAlbumScoredMatch(ranked score.RankedCandidate) ScoredMatch {
	return ScoredMatch{
		URL:       ranked.Candidate.MatchURL,
		Score:     ranked.Score,
		Reasons:   append([]string(nil), ranked.Reasons...),
		Candidate: ranked.Candidate,
	}
}
