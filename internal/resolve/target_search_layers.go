package resolve

import (
	"context"
	"fmt"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/targetsearch"
)

type targetSearchPlan[T any] struct {
	target  any
	service model.ServiceName
	keyFunc func(T) string
	layers  []targetSearchLayer[T]
}

type targetSearchLayer[T any] struct {
	name    string
	enabled bool
	search  func(context.Context) ([]T, error)
	filter  func([]T) []T
}

type targetSearchLayerOutcome[T any] struct {
	candidates []T
	err        error
}

func (p targetSearchPlan[T]) collect(ctx context.Context) ([]T, error) {
	combined := []T{}
	seen := map[string]struct{}{}
	for _, layer := range p.layers {
		outcome := p.runLayer(ctx, layer)
		if outcome.err != nil {
			return nil, outcome.err
		}
		combined = appendUniqueByKey(combined, seen, outcome.candidates, p.keyFunc)
	}
	return combined, nil
}

func (p targetSearchPlan[T]) runLayer(ctx context.Context, layer targetSearchLayer[T]) targetSearchLayerOutcome[T] {
	if !layer.enabled {
		return targetSearchLayerOutcome[T]{}
	}

	candidates, err := layer.search(ctx)
	if err != nil {
		if targetsearch.IsRecoverableTimeout(ctx, err) {
			return targetSearchLayerOutcome[T]{}
		}
		return targetSearchLayerOutcome[T]{err: fmt.Errorf("%s %s (%T) failed: %w", layer.name, p.service, p.target, err)}
	}
	if layer.filter != nil {
		candidates = layer.filter(candidates)
	}
	return targetSearchLayerOutcome[T]{candidates: candidates}
}

type albumMetadataCandidateFilter func([]model.CandidateAlbum) []model.CandidateAlbum

func collectAlbumTargetCandidates(ctx context.Context, target TargetAdapter, source model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
	return collectAlbumTargetCandidatesWithMetadataFilter(ctx, target, source, nil)
}

func collectAlbumTargetCandidatesWithMetadataFilter(
	ctx context.Context,
	target TargetAdapter,
	source model.CanonicalAlbum,
	metadataFilter albumMetadataCandidateFilter,
) ([]model.CandidateAlbum, error) {
	return albumTargetSearchPlan(target, source, metadataFilter).collect(ctx)
}

func albumTargetSearchPlan(target TargetAdapter, source model.CanonicalAlbum, metadataFilter albumMetadataCandidateFilter) targetSearchPlan[model.CandidateAlbum] {
	return targetSearchPlan[model.CandidateAlbum]{
		target:  target,
		service: target.Service(),
		keyFunc: albumCandidateKey,
		layers:  albumTargetSearchLayers(target, source, metadataFilter),
	}
}

func albumTargetSearchLayers(target TargetAdapter, source model.CanonicalAlbum, metadataFilter albumMetadataCandidateFilter) []targetSearchLayer[model.CandidateAlbum] {
	layers := make([]targetSearchLayer[model.CandidateAlbum], 0, 3)
	if searcher, ok := target.(UPCSearcher); ok {
		layers = append(layers, targetSearchLayer[model.CandidateAlbum]{
			name:    "SearchByUPC",
			enabled: source.UPC != "",
			search: func(ctx context.Context) ([]model.CandidateAlbum, error) {
				return searcher.SearchByUPC(ctx, source.UPC)
			},
		})
	}

	if searcher, ok := target.(ISRCSearcher); ok {
		isrcs := collectISRCs(source)
		layers = append(layers, targetSearchLayer[model.CandidateAlbum]{
			name:    "SearchByISRC",
			enabled: len(isrcs) > 0,
			search: func(ctx context.Context) ([]model.CandidateAlbum, error) {
				return searcher.SearchByISRC(ctx, isrcs)
			},
		})
	}

	if searcher, ok := target.(MetadataSearcher); ok {
		layers = append(layers, targetSearchLayer[model.CandidateAlbum]{
			name:    "SearchByMetadata",
			enabled: true,
			search: func(ctx context.Context) ([]model.CandidateAlbum, error) {
				return searcher.SearchByMetadata(ctx, source)
			},
			filter: metadataFilter,
		})
	}
	return layers
}

func collectSongTargetCandidates(ctx context.Context, target SongTargetAdapter, source model.CanonicalSong) ([]model.CandidateSong, error) {
	return songTargetSearchPlan(target, source).collect(ctx)
}

func songTargetSearchPlan(target SongTargetAdapter, source model.CanonicalSong) targetSearchPlan[model.CandidateSong] {
	return targetSearchPlan[model.CandidateSong]{
		target:  target,
		service: target.Service(),
		keyFunc: songCandidateKey,
		layers:  songTargetSearchLayers(target, source),
	}
}

func songTargetSearchLayers(target SongTargetAdapter, source model.CanonicalSong) []targetSearchLayer[model.CandidateSong] {
	layers := make([]targetSearchLayer[model.CandidateSong], 0, 2)
	if searcher, ok := target.(SongISRCSearcher); ok {
		layers = append(layers, targetSearchLayer[model.CandidateSong]{
			name:    "SearchSongByISRC",
			enabled: source.ISRC != "",
			search: func(ctx context.Context) ([]model.CandidateSong, error) {
				return searcher.SearchSongByISRC(ctx, source.ISRC)
			},
		})
	}
	if searcher, ok := target.(SongMetadataSearcher); ok {
		layers = append(layers, targetSearchLayer[model.CandidateSong]{
			name:    "SearchSongByMetadata",
			enabled: true,
			search: func(ctx context.Context) ([]model.CandidateSong, error) {
				return searcher.SearchSongByMetadata(ctx, source)
			},
		})
	}
	return layers
}
