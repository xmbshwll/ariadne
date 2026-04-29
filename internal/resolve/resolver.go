package resolve

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

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

const appleMusicCascadeMinimumScore = 100

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

	targets := excludeTargetService(r.targets, sourceAlbum.Service)
	matches, err := resolveTargetMatches(
		ctx,
		targets,
		*sourceAlbum,
		func(ctx context.Context, target TargetAdapter, source model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
			return collectAlbumTargetCandidates(ctx, target, source, r.weights)
		},
		func(source model.CanonicalAlbum, candidates []model.CandidateAlbum) score.Ranking {
			return score.RankAlbums(source, candidates, r.weights)
		},
		albumMatchResultFromRanking,
		"candidates",
	)
	if err != nil {
		return nil, fmt.Errorf("resolve target searches: %w", err)
	}

	resolution := &Resolution{
		InputURL: inputURL,
		Parsed:   *parsed,
		Source:   *sourceAlbum,
		Matches:  matches,
	}

	if err := r.resolveAppleMusicWithCascadedIdentifiers(ctx, targets, *sourceAlbum, resolution.Matches); err != nil {
		return nil, fmt.Errorf("resolve apple music cascaded search: %w", err)
	}
	return resolution, nil
}

func excludeTargetService[T interface{ Service() model.ServiceName }](targets []T, sourceService model.ServiceName) []T {
	filtered := make([]T, 0, len(targets))
	for _, target := range targets {
		if target.Service() == sourceService {
			continue
		}
		filtered = append(filtered, target)
	}
	return filtered
}

func (r *Resolver) parseSource(inputURL string) (SourceAdapter, *model.ParsedAlbumURL, error) {
	return parseSourceAdapter(
		r.sources,
		inputURL,
		func(source SourceAdapter, raw string) (*model.ParsedAlbumURL, error) {
			return source.ParseAlbumURL(raw)
		},
	)
}

func (r *Resolver) resolveAppleMusicWithCascadedIdentifiers(
	ctx context.Context,
	targets []TargetAdapter,
	source model.CanonicalAlbum,
	matches map[model.ServiceName]MatchResult,
) error {
	appleMusicTargets := appleMusicTargets(targets)
	if len(appleMusicTargets) == 0 {
		return nil
	}

	enriched := enrichAlbumIdentifiersFromStrongMatches(source, matches)
	if !albumIdentifiersChanged(source, enriched) {
		return nil
	}

	var matchesMu sync.Mutex
	return resolveTargetsConcurrently(ctx, appleMusicTargets, func(groupCtx context.Context, target TargetAdapter) error {
		candidates, err := collectAlbumTargetCandidates(groupCtx, target, enriched, r.weights)
		if err != nil {
			return fmt.Errorf("collect candidates from %s: %w", target.Service(), err)
		}
		ranking := score.RankAlbums(enriched, candidates, r.weights)

		newResult := albumMatchResultFromRanking(target.Service(), ranking)
		matchesMu.Lock()
		existing := matches[target.Service()]
		if newResult.Best != nil && (existing.Best == nil || newResult.Best.Score > existing.Best.Score) {
			matches[target.Service()] = newResult
		}
		matchesMu.Unlock()
		return nil
	})
}

func appleMusicTargets(targets []TargetAdapter) []TargetAdapter {
	filtered := make([]TargetAdapter, 0, len(targets))
	for _, target := range targets {
		if target.Service() != model.ServiceAppleMusic {
			continue
		}
		filtered = append(filtered, target)
	}
	return filtered
}

func enrichAlbumIdentifiersFromStrongMatches(source model.CanonicalAlbum, matches map[model.ServiceName]MatchResult) model.CanonicalAlbum {
	enriched := cloneAlbum(source)
	strongMatches := strongIntermediateAlbumMatches(matches)
	for _, match := range strongMatches {
		mergeAlbumIdentifiers(&enriched, match.Candidate)
	}
	return enriched
}

func cloneAlbum(album model.CanonicalAlbum) model.CanonicalAlbum {
	clone := album
	clone.Artists = append([]string(nil), album.Artists...)
	clone.NormalizedArtists = append([]string(nil), album.NormalizedArtists...)
	clone.EditionHints = append([]string(nil), album.EditionHints...)
	clone.Tracks = append([]model.CanonicalTrack(nil), album.Tracks...)
	return clone
}

func strongIntermediateAlbumMatches(matches map[model.ServiceName]MatchResult) []ScoredMatch {
	strongMatches := make([]ScoredMatch, 0, len(matches))
	for service, match := range matches {
		if service == model.ServiceAppleMusic || match.Best == nil || match.Best.Score < appleMusicCascadeMinimumScore {
			continue
		}
		strongMatches = append(strongMatches, *match.Best)
	}
	sort.SliceStable(strongMatches, func(i, j int) bool {
		if strongMatches[i].Score == strongMatches[j].Score {
			leftService := string(strongMatches[i].Candidate.Service)
			rightService := string(strongMatches[j].Candidate.Service)
			if leftService == rightService {
				return strongMatches[i].Candidate.CandidateID < strongMatches[j].Candidate.CandidateID
			}
			return leftService < rightService
		}
		return strongMatches[i].Score > strongMatches[j].Score
	})
	return strongMatches
}

func mergeAlbumIdentifiers(album *model.CanonicalAlbum, candidate model.CandidateAlbum) {
	if album.UPC == "" && candidate.UPC != "" {
		album.UPC = candidate.UPC
	}
	mergeTrackISRCs(album, candidate.Tracks)
}

func mergeTrackISRCs(album *model.CanonicalAlbum, tracks []model.CanonicalTrack) {
	if len(tracks) == 0 {
		return
	}
	if len(album.Tracks) == 0 {
		album.Tracks = append([]model.CanonicalTrack(nil), tracks...)
		return
	}
	if len(album.Tracks) != len(tracks) {
		return
	}
	for i := range album.Tracks {
		if album.Tracks[i].ISRC != "" || tracks[i].ISRC == "" {
			continue
		}
		album.Tracks[i].ISRC = tracks[i].ISRC
	}
}

func albumIdentifiersChanged(source model.CanonicalAlbum, enriched model.CanonicalAlbum) bool {
	if source.UPC != enriched.UPC {
		return true
	}
	sourceISRCs := collectISRCs(source)
	enrichedISRCs := collectISRCs(enriched)
	if len(sourceISRCs) != len(enrichedISRCs) {
		return true
	}
	for i := range sourceISRCs {
		if !strings.EqualFold(sourceISRCs[i], enrichedISRCs[i]) {
			return true
		}
	}
	return false
}

type fatalParseFailure interface {
	FatalParseFailure() bool
}

func parseSourceAdapter[S any, P any](sources []S, inputURL string, parse func(S, string) (*P, error)) (S, *P, error) {
	var zero S
	for _, source := range sources {
		parsed, err := parse(source, inputURL)
		if err != nil {
			var fatal fatalParseFailure
			if errors.As(err, &fatal) && fatal.FatalParseFailure() {
				return zero, nil, err
			}
			continue
		}
		if parsed == nil {
			continue
		}
		return source, parsed, nil
	}
	return zero, nil, fmt.Errorf("%w: %s", ErrUnsupportedURL, inputURL)
}

func resolveTargetsConcurrently[T interface{ Service() model.ServiceName }](ctx context.Context, targets []T, resolve func(context.Context, T) error) error {
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
