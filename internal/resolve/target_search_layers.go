package resolve

import (
	"context"
	"fmt"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/targetsearch"
)

type albumMetadataCandidateFilter func([]model.CandidateAlbum) []model.CandidateAlbum

func newTargetSearchPlan[Candidate any](
	target serviceAdapter,
	keyFunc func(Candidate) string,
	layers []targetsearch.Layer[Candidate],
) targetsearch.Plan[Candidate] {
	return targetsearch.Plan[Candidate]{
		Target:       target,
		Service:      string(target.Service()),
		CandidateKey: keyFunc,
		Layers:       layers,
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
		return nil, fmt.Errorf("collect album target candidates: %w", err)
	}
	return candidates, nil
}

func albumTargetSearchPlan(target TargetAdapter, source model.CanonicalAlbum, metadataFilter albumMetadataCandidateFilter) targetsearch.Plan[model.CandidateAlbum] {
	return newTargetSearchPlan(target, albumCandidateKey, albumTargetSearchLayers(target, source, metadataFilter))
}

func albumTargetSearchLayers(target TargetAdapter, source model.CanonicalAlbum, metadataFilter albumMetadataCandidateFilter) []targetsearch.Layer[model.CandidateAlbum] {
	layers := make([]targetsearch.Layer[model.CandidateAlbum], 0, 3)
	if searcher, ok := target.(UPCSearcher); ok {
		layers = append(layers, targetsearch.Layer[model.CandidateAlbum]{
			Name:    "SearchByUPC",
			Enabled: source.UPC != "",
			Search: func(ctx context.Context) ([]model.CandidateAlbum, error) {
				return searcher.SearchByUPC(ctx, source.UPC)
			},
		})
	}

	if searcher, ok := target.(ISRCSearcher); ok {
		isrcs := collectISRCs(source)
		layers = append(layers, targetsearch.Layer[model.CandidateAlbum]{
			Name:    "SearchByISRC",
			Enabled: len(isrcs) > 0,
			Search: func(ctx context.Context) ([]model.CandidateAlbum, error) {
				return searcher.SearchByISRC(ctx, isrcs)
			},
		})
	}

	if searcher, ok := target.(MetadataSearcher); ok {
		layers = append(layers, targetsearch.Layer[model.CandidateAlbum]{
			Name:    "SearchByMetadata",
			Enabled: true,
			Search: func(ctx context.Context) ([]model.CandidateAlbum, error) {
				return searcher.SearchByMetadata(ctx, source)
			},
			Filter: metadataFilter,
		})
	}
	return layers
}

func collectSongTargetCandidates(ctx context.Context, target SongTargetAdapter, source model.CanonicalSong) ([]model.CandidateSong, error) {
	candidates, err := songTargetSearchPlan(target, source).Collect(ctx)
	if err != nil {
		return nil, fmt.Errorf("collect song target candidates: %w", err)
	}
	return candidates, nil
}

func songTargetSearchPlan(target SongTargetAdapter, source model.CanonicalSong) targetsearch.Plan[model.CandidateSong] {
	return newTargetSearchPlan(target, songCandidateKey, songTargetSearchLayers(target, source))
}

func songTargetSearchLayers(target SongTargetAdapter, source model.CanonicalSong) []targetsearch.Layer[model.CandidateSong] {
	layers := make([]targetsearch.Layer[model.CandidateSong], 0, 2)
	if searcher, ok := target.(SongISRCSearcher); ok {
		layers = append(layers, targetsearch.Layer[model.CandidateSong]{
			Name:    "SearchSongByISRC",
			Enabled: source.ISRC != "",
			Search: func(ctx context.Context) ([]model.CandidateSong, error) {
				return searcher.SearchSongByISRC(ctx, source.ISRC)
			},
		})
	}
	if searcher, ok := target.(SongMetadataSearcher); ok {
		layers = append(layers, targetsearch.Layer[model.CandidateSong]{
			Name:    "SearchSongByMetadata",
			Enabled: true,
			Search: func(ctx context.Context) ([]model.CandidateSong, error) {
				return searcher.SearchSongByMetadata(ctx, source)
			},
		})
	}
	return layers
}
