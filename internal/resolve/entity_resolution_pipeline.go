package resolve

import (
	"context"
	"fmt"

	"github.com/xmbshwll/ariadne/internal/model"
)

type entityResolution[P any, Entity any, Match any] struct {
	InputURL string
	Parsed   P
	Source   Entity
	Matches  map[model.ServiceName]Match
}

type entityResolutionPipeline[SourceAdapter interface{ Service() model.ServiceName }, TargetAdapter interface{ Service() model.ServiceName }, Parsed any, Entity any, Candidate any, Ranking any, Match any] struct {
	sources []SourceAdapter
	targets []TargetAdapter

	parse         func(SourceAdapter, string) (*Parsed, error)
	hydrate       func(context.Context, SourceAdapter, Parsed) (*Entity, error)
	sourceService func(Entity) model.ServiceName
	collect       func(context.Context, TargetAdapter, Entity) ([]Candidate, error)
	rank          func(Entity, []Candidate) Ranking
	result        func(model.ServiceName, Ranking) Match

	entityLabel    string
	nilEntityErr   error
	candidateLabel string
	targetErrLabel string

	afterTargets         func(context.Context, []TargetAdapter, Entity, map[model.ServiceName]Match) error
	afterTargetsErrLabel string
}

func resolveEntity[SourceAdapter interface{ Service() model.ServiceName }, TargetAdapter interface{ Service() model.ServiceName }, Parsed any, Entity any, Candidate any, Ranking any, Match any](
	ctx context.Context,
	inputURL string,
	pipeline entityResolutionPipeline[SourceAdapter, TargetAdapter, Parsed, Entity, Candidate, Ranking, Match],
) (entityResolution[Parsed, Entity, Match], error) {
	var zero entityResolution[Parsed, Entity, Match]
	source, err := resolveSourceInput(
		ctx,
		pipeline.sources,
		inputURL,
		pipeline.parse,
		pipeline.hydrate,
		pipeline.entityLabel,
		pipeline.nilEntityErr,
	)
	if err != nil {
		return zero, err
	}

	targets := excludeTargetService(pipeline.targets, pipeline.sourceService(source.Entity))
	matches, err := resolveTargetMatches(
		ctx,
		targets,
		source.Entity,
		pipeline.collect,
		pipeline.rank,
		pipeline.result,
		pipeline.candidateLabel,
	)
	if err != nil {
		return zero, fmt.Errorf("%s: %w", pipeline.targetErrLabel, err)
	}

	if pipeline.afterTargets != nil {
		if err := pipeline.afterTargets(ctx, targets, source.Entity, matches); err != nil {
			return zero, fmt.Errorf("%s: %w", pipeline.afterTargetsErrLabel, err)
		}
	}

	return entityResolution[Parsed, Entity, Match]{
		InputURL: inputURL,
		Parsed:   source.Parsed,
		Source:   source.Entity,
		Matches:  matches,
	}, nil
}
