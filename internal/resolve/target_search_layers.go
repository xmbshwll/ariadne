package resolve

import (
	"context"
	"fmt"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/targetsearch"
)

// albumMetadataCandidateFilter narrows metadata candidates before they reach ranking.
type albumMetadataCandidateFilter func([]model.CandidateAlbum) []model.CandidateAlbum

// targetSearchLayer describes one optional Target Search layer as data: its
// name, how to probe the Capability off the target adapter and drive it from the
// Source Entity, and an optional candidate Filter. The order of the table is the
// execution order, so a layer's position is visible without reading control flow.
//
// A layer whose Capability or Source Entity identifier is missing stays in the
// Plan as Enabled:false, which Plan.Collect skips, so every adapter sees the
// same layer list and only the supported layers run. A probe therefore never
// returns a nil search together with true.
type targetSearchLayer[Target serviceAdapter, Entity, Candidate any] struct {
	name   string
	probe  func(Target, Entity) (func(context.Context) ([]Candidate, error), bool)
	filter func([]Candidate) []Candidate
}

// buildTargetSearchPlan turns a layer table into the Plan Target Search runs.
func buildTargetSearchPlan[Target serviceAdapter, Entity, Candidate any](
	target Target,
	entity Entity,
	candidateKey func(Candidate) string,
	layers []targetSearchLayer[Target, Entity, Candidate],
) targetsearch.Plan[Candidate] {
	planLayers := make([]targetsearch.Layer[Candidate], 0, len(layers))
	for _, layer := range layers {
		search, enabled := layer.probe(target, entity)
		planLayers = append(planLayers, targetsearch.Layer[Candidate]{
			Name:    layer.name,
			Enabled: enabled,
			Search:  search,
			Filter:  layer.filter,
		})
	}
	return targetsearch.Plan[Candidate]{
		Target:       target,
		Service:      string(target.Service()),
		CandidateKey: candidateKey,
		Layers:       planLayers,
	}
}

func collectAlbumTargetCandidates(ctx context.Context, target TargetAdapter, source model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
	return collectAlbumTargetCandidatesWithMetadataFilter(ctx, target, source, nil)
}

func collectAlbumTargetCandidatesWithMetadataFilter(
	ctx context.Context,
	target TargetAdapter,
	source model.CanonicalAlbum,
	metadataFilter albumMetadataCandidateFilter,
) ([]model.CandidateAlbum, error) {
	candidates, err := albumTargetSearchPlan(target, source, metadataFilter).Collect(ctx)
	if err != nil {
		return nil, fmt.Errorf("album target search: %w", err)
	}
	return candidates, nil
}

func albumTargetSearchPlan(target TargetAdapter, source model.CanonicalAlbum, metadataFilter albumMetadataCandidateFilter) targetsearch.Plan[model.CandidateAlbum] {
	return buildTargetSearchPlan(target, source, model.CandidateAlbum.SearchKey, albumTargetSearchLayers(metadataFilter))
}

// albumTargetSearchLayers is the album Target Search ladder: identifier lookups
// first because they are exact, metadata search last because it is fuzzy and
// needs the Apple Music candidate filter applied.
func albumTargetSearchLayers(metadataFilter albumMetadataCandidateFilter) []targetSearchLayer[TargetAdapter, model.CanonicalAlbum, model.CandidateAlbum] {
	type search = func(context.Context) ([]model.CandidateAlbum, error)
	return []targetSearchLayer[TargetAdapter, model.CanonicalAlbum, model.CandidateAlbum]{
		{
			name: "SearchByUPC",
			probe: func(target TargetAdapter, source model.CanonicalAlbum) (search, bool) {
				searcher, ok := target.(UPCSearcher)
				if !ok || source.UPC == "" {
					return nil, false
				}
				return func(ctx context.Context) ([]model.CandidateAlbum, error) {
					return searcher.SearchByUPC(ctx, source.UPC)
				}, true
			},
		},
		{
			name: "SearchByISRC",
			probe: func(target TargetAdapter, source model.CanonicalAlbum) (search, bool) {
				searcher, ok := target.(ISRCSearcher)
				if !ok {
					return nil, false
				}
				isrcs := collectISRCs(source)
				if len(isrcs) == 0 {
					return nil, false
				}
				return func(ctx context.Context) ([]model.CandidateAlbum, error) {
					return searcher.SearchByISRC(ctx, isrcs)
				}, true
			},
		},
		{
			name: "SearchByMetadata",
			probe: func(target TargetAdapter, source model.CanonicalAlbum) (search, bool) {
				searcher, ok := target.(MetadataSearcher)
				if !ok {
					return nil, false
				}
				return func(ctx context.Context) ([]model.CandidateAlbum, error) {
					return searcher.SearchByMetadata(ctx, source)
				}, true
			},
			filter: metadataFilter,
		},
	}
}

func collectSongTargetCandidates(ctx context.Context, target SongTargetAdapter, source model.CanonicalSong) ([]model.CandidateSong, error) {
	candidates, err := songTargetSearchPlan(target, source).Collect(ctx)
	if err != nil {
		return nil, fmt.Errorf("song target search: %w", err)
	}
	return candidates, nil
}

func songTargetSearchPlan(target SongTargetAdapter, source model.CanonicalSong) targetsearch.Plan[model.CandidateSong] {
	return buildTargetSearchPlan(target, source, model.CandidateSong.SearchKey, songTargetSearchLayers())
}

// songTargetSearchLayers is the song Target Search ladder: ISRC first, then metadata.
func songTargetSearchLayers() []targetSearchLayer[SongTargetAdapter, model.CanonicalSong, model.CandidateSong] {
	type search = func(context.Context) ([]model.CandidateSong, error)
	return []targetSearchLayer[SongTargetAdapter, model.CanonicalSong, model.CandidateSong]{
		{
			name: "SearchSongByISRC",
			probe: func(target SongTargetAdapter, source model.CanonicalSong) (search, bool) {
				searcher, ok := target.(SongISRCSearcher)
				if !ok || source.ISRC == "" {
					return nil, false
				}
				return func(ctx context.Context) ([]model.CandidateSong, error) {
					return searcher.SearchSongByISRC(ctx, source.ISRC)
				}, true
			},
		},
		{
			name: "SearchSongByMetadata",
			probe: func(target SongTargetAdapter, source model.CanonicalSong) (search, bool) {
				searcher, ok := target.(SongMetadataSearcher)
				if !ok {
					return nil, false
				}
				return func(ctx context.Context) ([]model.CandidateSong, error) {
					return searcher.SearchSongByMetadata(ctx, source)
				}, true
			},
		},
	}
}
