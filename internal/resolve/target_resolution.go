package resolve

import (
	"context"
	"fmt"
	"sync"

	"github.com/xmbshwll/ariadne/internal/model"
)

func resolveTargetMatches[Target interface{ Service() model.ServiceName }, Source any, Candidate any, Ranking any, Match any](
	ctx context.Context,
	targets []Target,
	source Source,
	collect func(context.Context, Target, Source) ([]Candidate, error),
	rank func(Source, []Candidate) Ranking,
	result func(model.ServiceName, Ranking) Match,
	candidateLabel string,
) (map[model.ServiceName]Match, error) {
	matches := make(map[model.ServiceName]Match, len(targets))
	var matchesMu sync.Mutex

	if err := resolveTargetsConcurrently(ctx, targets, func(groupCtx context.Context, target Target) error {
		candidates, err := collect(groupCtx, target, source)
		if err != nil {
			return fmt.Errorf("collect %s from %s: %w", candidateLabel, target.Service(), err)
		}
		ranking := rank(source, candidates)
		match := result(target.Service(), ranking)

		matchesMu.Lock()
		matches[target.Service()] = match
		matchesMu.Unlock()
		return nil
	}); err != nil {
		return nil, err
	}

	return matches, nil
}
