package resolve

import (
	"context"
	"fmt"

	"github.com/xmbshwll/ariadne/internal/adapters"
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/targetsearch"
)

// albumMetadataCandidateFilter narrows metadata candidates before they reach ranking.
type albumMetadataCandidateFilter func([]model.CandidateAlbum) []model.CandidateAlbum

// targetSearchLayer describes one Target Search layer as data: its name, the
// Capability it needs from the target adapter, how to drive it from the Source
// Entity, and an optional candidate Filter. The order of the table is the
// execution order, so a layer's position is visible without reading control flow.
//
// A layer whose Capability or Source Entity identifier is missing stays in the
// Plan as Enabled:false, which Plan.Collect skips, so every adapter sees the
// same layer list and only the layers its Capabilities and the Source Entity
// support ever run.
type targetSearchLayer[Entity, Candidate any] struct {
	// name identifies the layer in Target Search errors.
	name string
	// supported reports whether a service Capability enables this layer.
	supported func(adapters.Capabilities) bool
	// probe builds the search for this target and entity, or returns nil when the
	// Source Entity carries no identifier the layer needs.
	probe  func(adapters.Adapter, Entity) func(context.Context) ([]Candidate, error)
	filter func([]Candidate) []Candidate
}

// buildTargetSearchPlan turns a layer table into the Plan Target Search runs.
func buildTargetSearchPlan[Entity, Candidate any](
	target adapters.Adapter,
	entity Entity,
	candidateKey func(Candidate) string,
	layers []targetSearchLayer[Entity, Candidate],
) targetsearch.Plan[Candidate] {
	capabilities := target.Capabilities()
	planLayers := make([]targetsearch.Layer[Candidate], 0, len(layers))
	for _, layer := range layers {
		var search func(context.Context) ([]Candidate, error)
		if layer.supported(capabilities) {
			search = layer.probe(target, entity)
		}
		planLayers = append(planLayers, targetsearch.Layer[Candidate]{
			Name:    layer.name,
			Enabled: search != nil,
			Search:  search,
			Filter:  layer.filter,
		})
	}
	return targetsearch.Plan[Candidate]{
		Target:       string(target.Service()),
		CandidateKey: candidateKey,
		Layers:       planLayers,
	}
}

func collectAlbumTargetCandidates(ctx context.Context, target adapters.Adapter, source model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
	return collectAlbumTargetCandidatesWithMetadataFilter(ctx, target, source, nil)
}

func collectAlbumTargetCandidatesWithMetadataFilter(
	ctx context.Context,
	target adapters.Adapter,
	source model.CanonicalAlbum,
	metadataFilter albumMetadataCandidateFilter,
) ([]model.CandidateAlbum, error) {
	candidates, err := albumTargetSearchPlan(target, source, metadataFilter).Collect(ctx)
	if err != nil {
		return nil, fmt.Errorf("album target search: %w", err)
	}
	return candidates, nil
}

func albumTargetSearchPlan(target adapters.Adapter, source model.CanonicalAlbum, metadataFilter albumMetadataCandidateFilter) targetsearch.Plan[model.CandidateAlbum] {
	return buildTargetSearchPlan(target, source, model.CandidateAlbum.SearchKey, albumTargetSearchLayers(metadataFilter))
}

// albumTargetSearchLayers is the album Target Search ladder: identifier lookups
// first because they are exact, metadata search last because it is fuzzy and
// needs the Apple Music candidate filter applied.
func albumTargetSearchLayers(metadataFilter albumMetadataCandidateFilter) []targetSearchLayer[model.CanonicalAlbum, model.CandidateAlbum] {
	type search = func(context.Context) ([]model.CandidateAlbum, error)
	return []targetSearchLayer[model.CanonicalAlbum, model.CandidateAlbum]{
		{
			name:      "SearchAlbumByUPC",
			supported: func(capabilities adapters.Capabilities) bool { return capabilities.AlbumUPC },
			probe: func(target adapters.Adapter, source model.CanonicalAlbum) search {
				if source.UPC == "" {
					return nil
				}
				return func(ctx context.Context) ([]model.CandidateAlbum, error) {
					return target.SearchAlbumByUPC(ctx, source.UPC)
				}
			},
		},
		{
			name:      "SearchAlbumByISRC",
			supported: func(capabilities adapters.Capabilities) bool { return capabilities.AlbumISRC },
			probe: func(target adapters.Adapter, source model.CanonicalAlbum) search {
				isrcs := collectISRCs(source)
				if len(isrcs) == 0 {
					return nil
				}
				return func(ctx context.Context) ([]model.CandidateAlbum, error) {
					return target.SearchAlbumByISRC(ctx, isrcs)
				}
			},
		},
		{
			name:      "SearchAlbumByMetadata",
			supported: func(capabilities adapters.Capabilities) bool { return capabilities.AlbumMetadata },
			probe: func(target adapters.Adapter, source model.CanonicalAlbum) search {
				return func(ctx context.Context) ([]model.CandidateAlbum, error) {
					return target.SearchAlbumByMetadata(ctx, source)
				}
			},
			filter: metadataFilter,
		},
	}
}

func collectSongTargetCandidates(ctx context.Context, target adapters.Adapter, source model.CanonicalSong) ([]model.CandidateSong, error) {
	candidates, err := songTargetSearchPlan(target, source).Collect(ctx)
	if err != nil {
		return nil, fmt.Errorf("song target search: %w", err)
	}
	return candidates, nil
}

func songTargetSearchPlan(target adapters.Adapter, source model.CanonicalSong) targetsearch.Plan[model.CandidateSong] {
	return buildTargetSearchPlan(target, source, model.CandidateSong.SearchKey, songTargetSearchLayers())
}

// songTargetSearchLayers is the song Target Search ladder: ISRC first, then metadata.
func songTargetSearchLayers() []targetSearchLayer[model.CanonicalSong, model.CandidateSong] {
	type search = func(context.Context) ([]model.CandidateSong, error)
	return []targetSearchLayer[model.CanonicalSong, model.CandidateSong]{
		{
			name:      "SearchSongByISRC",
			supported: func(capabilities adapters.Capabilities) bool { return capabilities.SongISRC },
			probe: func(target adapters.Adapter, source model.CanonicalSong) search {
				if source.ISRC == "" {
					return nil
				}
				return func(ctx context.Context) ([]model.CandidateSong, error) {
					return target.SearchSongByISRC(ctx, source.ISRC)
				}
			},
		},
		{
			name:      "SearchSongByMetadata",
			supported: func(capabilities adapters.Capabilities) bool { return capabilities.SongMetadata },
			probe: func(target adapters.Adapter, source model.CanonicalSong) search {
				return func(ctx context.Context) ([]model.CandidateSong, error) {
					return target.SearchSongByMetadata(ctx, source)
				}
			},
		},
	}
}
