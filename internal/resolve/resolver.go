package resolve

import (
	"context"
	"errors"

	"github.com/xmbshwll/ariadne/internal/adapters"
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/score"
)

var (
	// ErrUnsupportedURL indicates that no registered source adapter recognized the input URL.
	ErrUnsupportedURL = errors.New("unsupported url")
	// ErrNoSourceAdapters indicates that the resolver was created without source adapters.
	ErrNoSourceAdapters = errors.New("no source adapters configured")
)

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

// Resolver coordinates album Source Input, Runtime Hydration, Target Search, and Identifier Enrichment.
type Resolver struct {
	policy albumEntityResolutionPolicy
}

// albumEntityResolutionPolicy is the Entity Resolution pipeline configured for albums.
type albumEntityResolutionPolicy = entityResolution[model.ParsedAlbumURL, model.CanonicalAlbum, model.CandidateAlbum]

func newAlbumEntityResolutionPolicy(sources []adapters.Adapter, targets []adapters.Adapter, weights score.Weights) albumEntityResolutionPolicy {
	enrichment := NewAppleMusicEnrichmentPolicy(weights)
	return albumEntityResolutionPolicy{
		targetAdapters:    append([]adapters.Adapter(nil), targets...),
		collectCandidates: enrichment.collectTargetCandidates,
		rank: func(album model.CanonicalAlbum, candidates []model.CandidateAlbum) score.Ranking[model.CandidateAlbum] {
			return score.RankAlbums(album, candidates, weights)
		},
		entityService:      func(album model.CanonicalAlbum) model.ServiceName { return album.Service },
		candidateURL:       func(candidate model.CandidateAlbum) string { return candidate.MatchURL },
		collectFailure:     "collect candidates",
		afterTargetMatches: enrichment.apply,
		resolveSourceInput: func(ctx context.Context, inputURL string) (model.ParsedAlbumURL, model.CanonicalAlbum, error) {
			return resolveEntitySourceInput(ctx, sources, inputURL, albumEntityLabel, ErrNilSourceAlbum,
				func(source adapters.Adapter, rawURL string) (*model.ParsedAlbumURL, error) {
					return source.ParseAlbumURL(rawURL)
				},
				func(ctx context.Context, source adapters.Adapter, parsed *model.ParsedAlbumURL) (*model.CanonicalAlbum, error) {
					return source.FetchAlbum(ctx, *parsed)
				})
		},
	}
}

// New creates a resolver from registered source and target adapters.
func New(sources []adapters.Adapter, targets []adapters.Adapter, weights score.Weights) *Resolver {
	return &Resolver{policy: newAlbumEntityResolutionPolicy(sources, targets, weights)}
}

// ResolveAlbum parses an input album URL, fetches the canonical source album,
// then collects and ranks candidates from every target adapter except the source
// service. A failing target does not abort the resolution: its MatchResult
// carries the error in Err while other targets resolve normally.
func (r *Resolver) ResolveAlbum(ctx context.Context, inputURL string) (*Resolution, error) {
	return r.policy.resolve(ctx, inputURL)
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
