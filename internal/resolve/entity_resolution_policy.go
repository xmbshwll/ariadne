package resolve

import (
	"context"

	"github.com/xmbshwll/ariadne/internal/model"
)

func resolveEntity[Parsed any, Entity any, Target serviceAdapter, Match any, Result any](
	ctx context.Context,
	inputURL string,
	resolveSourceInput func(context.Context, string) (sourceInput[Parsed, Entity], error),
	sourceService func(Entity) model.ServiceName,
	targets []Target,
	resolveTargetMatches func(context.Context, []Target, Entity) (map[model.ServiceName]Match, error),
	afterTargetMatches func(context.Context, []Target, Entity, map[model.ServiceName]Match) error,
	resolution func(string, sourceInput[Parsed, Entity], map[model.ServiceName]Match) *Result,
) (*Result, error) {
	source, err := resolveSourceInput(ctx, inputURL)
	if err != nil {
		return nil, err
	}

	targets = excludeTargetService(targets, sourceService(source.Entity))
	matches, err := resolveTargetMatches(ctx, targets, source.Entity)
	if err != nil {
		return nil, err
	}

	if err := afterTargetMatches(ctx, targets, source.Entity, matches); err != nil {
		return nil, err
	}

	return resolution(inputURL, source, matches), nil
}
