package resolve

import (
	"context"
	"fmt"
	"sync"

	"github.com/xmbshwll/ariadne/internal/adapters"
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/score"
)

// entityResolution is the Entity Resolution pipeline shared by every entity
// shape: Source Input recognition, Runtime Hydration, per-target Target Search,
// and ranking. An entity shape supplies only what genuinely differs — how to
// parse and fetch its source, how to collect and rank its candidates, and the
// optional Identifier Enrichment pass that runs after Target Search.
type entityResolution[P, E, C any] struct {
	// resolveSourceInput recognizes the input URL and hydrates the source entity.
	resolveSourceInput func(context.Context, string) (P, E, error)
	// targetAdapters are all configured targets; the source service is excluded per run.
	targetAdapters []adapters.Adapter
	// collectCandidates runs the Target Search layers of one target.
	collectCandidates func(context.Context, adapters.Adapter, E) ([]C, error)
	// rank orders collected candidates for one source entity.
	rank func(E, []C) score.Ranking[C]
	// entityService is the Music Service a source entity came from.
	entityService func(E) model.ServiceName
	// candidateURL is the match URL of one candidate.
	candidateURL func(C) string
	// collectFailure prefixes a Target Search failure inside a MatchResult.
	collectFailure string
	// afterTargetMatches is the optional Identifier Enrichment pass.
	afterTargetMatches func(context.Context, []adapters.Adapter, E, map[model.ServiceName]MatchResultOf[C])
}

// resolve runs Entity Resolution for one input URL. A failing target does not
// abort the run: its MatchResult carries the error while other targets resolve.
func (p entityResolution[P, E, C]) resolve(ctx context.Context, inputURL string) (*ResolutionOf[P, E, C], error) {
	parsed, source, err := p.resolveSourceInput(ctx, inputURL)
	if err != nil {
		return nil, err
	}

	targets := excludeTargetService(p.targetAdapters, p.entityService(source))
	matches := p.resolveTargetMatches(ctx, targets, source)
	if p.afterTargetMatches != nil {
		p.afterTargetMatches(ctx, targets, source, matches)
	}

	return &ResolutionOf[P, E, C]{
		InputURL: inputURL,
		Parsed:   parsed,
		Source:   source,
		Matches:  matches,
	}, nil
}

// resolveTargetMatches resolves every target concurrently.
func (p entityResolution[P, E, C]) resolveTargetMatches(
	ctx context.Context,
	targets []adapters.Adapter,
	source E,
) map[model.ServiceName]MatchResultOf[C] {
	matches := make(map[model.ServiceName]MatchResultOf[C], len(targets))
	var matchesMu sync.Mutex

	resolveTargetsConcurrently(ctx, targets, func(targetCtx context.Context, target adapters.Adapter) {
		result := p.resolveTarget(targetCtx, target, source)

		matchesMu.Lock()
		matches[target.Service()] = result
		matchesMu.Unlock()
	})
	return matches
}

// resolveTarget collects and ranks one target's candidates, carrying a Target
// Search failure in the MatchResult instead of returning it.
func (p entityResolution[P, E, C]) resolveTarget(ctx context.Context, target adapters.Adapter, source E) MatchResultOf[C] {
	result, err := resolveTargetFor(ctx, p.collectCandidates, p.rank, p.candidateURL, target, source)
	if err != nil {
		return MatchResultOf[C]{Service: target.Service(), Err: fmt.Errorf("%s: %w", p.collectFailure, err)}
	}
	return result
}

// resolveTargetFor is the single-target collect-and-rank path shared by Entity
// Resolution and Identifier Enrichment. A Target Search failure returns to the
// caller: Entity Resolution embeds it in the MatchResult, while a failed
// enrichment search must leave the base matches untouched.
func resolveTargetFor[E, C any](
	ctx context.Context,
	collect func(context.Context, adapters.Adapter, E) ([]C, error),
	rank func(E, []C) score.Ranking[C],
	candidateURL func(C) string,
	target adapters.Adapter,
	source E,
) (MatchResultOf[C], error) {
	candidates, err := collect(ctx, target, source)
	if err != nil {
		return MatchResultOf[C]{}, err
	}
	return matchResultFromRanking(target.Service(), rank(source, candidates), candidateURL), nil
}

// resolveEntitySourceInput runs Source Input recognition and Runtime Hydration
// for one entity shape.
func resolveEntitySourceInput[P, E any](
	ctx context.Context,
	sources []adapters.Adapter,
	inputURL string,
	entityLabel string,
	nilEntityErr error,
	parse func(adapters.Adapter, string) (*P, error),
	hydrate func(context.Context, adapters.Adapter, *P) (*E, error),
) (P, E, error) {
	var (
		zeroParsed P
		zeroEntity E
	)

	parsedURL, source, err := RecognizeSourceInput(sources, inputURL, func(source adapters.Adapter) (*P, error) {
		return parse(source, inputURL)
	})
	if err != nil {
		return zeroParsed, zeroEntity, err
	}

	entity, err := HydrateSourceInput(ctx, source, entityLabel, nilEntityErr, func(ctx context.Context) (*E, error) {
		return hydrate(ctx, source, parsedURL)
	})
	if err != nil {
		return zeroParsed, zeroEntity, err
	}

	return *parsedURL, *entity, nil
}

// excludeTargetService removes the source service from a target set.
func excludeTargetService(targets []adapters.Adapter, sourceService model.ServiceName) []adapters.Adapter {
	filtered := make([]adapters.Adapter, 0, len(targets))
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
func resolveTargetsConcurrently(ctx context.Context, targets []adapters.Adapter, resolve func(context.Context, adapters.Adapter)) {
	var group sync.WaitGroup
	for _, target := range targets {
		group.Go(func() {
			resolve(ctx, target)
		})
	}
	group.Wait()
}
