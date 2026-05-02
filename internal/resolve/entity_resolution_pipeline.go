package resolve

import (
	"context"
	"fmt"

	"github.com/xmbshwll/ariadne/internal/model"
)

type serviceAdapter interface {
	Service() model.ServiceName
}

type entityResolution[P any, Entity any, Match any] struct {
	InputURL string
	Parsed   P
	Source   Entity
	Matches  map[model.ServiceName]Match
}

type entitySourcePolicy[SourceAdapter serviceAdapter, Parsed any, Entity any] struct {
	parse         func(SourceAdapter, string) (*Parsed, error)
	hydrate       func(context.Context, SourceAdapter, Parsed) (*Entity, error)
	sourceService func(Entity) model.ServiceName
	entityLabel   string
	nilEntityErr  error
}

type entityTargetPolicy[TargetAdapter serviceAdapter, Entity any, Candidate any, Ranking any, Match any] struct {
	collect        func(context.Context, TargetAdapter, Entity) ([]Candidate, error)
	rank           func(Entity, []Candidate) Ranking
	result         func(model.ServiceName, Ranking) Match
	candidateLabel string
	errLabel       string
}

type entityAfterTargetsPolicy[TargetAdapter serviceAdapter, Entity any, Match any] struct {
	run      func(context.Context, []TargetAdapter, Entity, map[model.ServiceName]Match) error
	errLabel string
}

type entityResolutionPipeline[SourceAdapter serviceAdapter, TargetAdapter serviceAdapter, Parsed any, Entity any, Candidate any, Ranking any, Match any] struct {
	sources []SourceAdapter
	targets []TargetAdapter
	source  entitySourcePolicy[SourceAdapter, Parsed, Entity]
	target  entityTargetPolicy[TargetAdapter, Entity, Candidate, Ranking, Match]
	after   entityAfterTargetsPolicy[TargetAdapter, Entity, Match]
}

type entityResolver[SourceAdapter serviceAdapter, TargetAdapter serviceAdapter, Parsed any, Entity any, Candidate any, Ranking any, Match any] struct {
	pipeline entityResolutionPipeline[SourceAdapter, TargetAdapter, Parsed, Entity, Candidate, Ranking, Match]
}

func newEntityResolver[SourceAdapter serviceAdapter, TargetAdapter serviceAdapter, Parsed any, Entity any, Candidate any, Ranking any, Match any](
	sources []SourceAdapter,
	targets []TargetAdapter,
	pipeline entityResolutionPipeline[SourceAdapter, TargetAdapter, Parsed, Entity, Candidate, Ranking, Match],
) entityResolver[SourceAdapter, TargetAdapter, Parsed, Entity, Candidate, Ranking, Match] {
	pipeline.sources = append([]SourceAdapter(nil), sources...)
	pipeline.targets = append([]TargetAdapter(nil), targets...)
	return entityResolver[SourceAdapter, TargetAdapter, Parsed, Entity, Candidate, Ranking, Match]{pipeline: pipeline}
}

func (r entityResolver[SourceAdapter, TargetAdapter, Parsed, Entity, Candidate, Ranking, Match]) resolve(
	ctx context.Context,
	inputURL string,
) (entityResolution[Parsed, Entity, Match], error) {
	return resolveEntity(ctx, inputURL, r.pipeline)
}

func resolveEntity[SourceAdapter serviceAdapter, TargetAdapter serviceAdapter, Parsed any, Entity any, Candidate any, Ranking any, Match any](
	ctx context.Context,
	inputURL string,
	pipeline entityResolutionPipeline[SourceAdapter, TargetAdapter, Parsed, Entity, Candidate, Ranking, Match],
) (entityResolution[Parsed, Entity, Match], error) {
	var zero entityResolution[Parsed, Entity, Match]
	source, err := resolveSourceInput(
		ctx,
		pipeline.sources,
		inputURL,
		pipeline.source.parse,
		pipeline.source.hydrate,
		pipeline.source.entityLabel,
		pipeline.source.nilEntityErr,
	)
	if err != nil {
		return zero, err
	}

	targets := excludeTargetService(pipeline.targets, pipeline.source.sourceService(source.Entity))
	matches, err := resolveTargetMatches(
		ctx,
		targets,
		source.Entity,
		pipeline.target.collect,
		pipeline.target.rank,
		pipeline.target.result,
		pipeline.target.candidateLabel,
	)
	if err != nil {
		return zero, fmt.Errorf("%s: %w", pipeline.target.errLabel, err)
	}

	if pipeline.after.run != nil {
		if err := pipeline.after.run(ctx, targets, source.Entity, matches); err != nil {
			return zero, fmt.Errorf("%s: %w", pipeline.after.errLabel, err)
		}
	}

	return entityResolution[Parsed, Entity, Match]{
		InputURL: inputURL,
		Parsed:   source.Parsed,
		Source:   source.Entity,
		Matches:  matches,
	}, nil
}
