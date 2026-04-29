package resolve

import (
	"context"
	"fmt"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/score"
)

type targetSearchLayer[T any] struct {
	name    string
	enabled bool
	search  func(context.Context) ([]T, error)
	filter  func([]T) []T
}

func collectTargetSearchLayers[T any](ctx context.Context, target any, service model.ServiceName, keyFunc func(T) string, layers ...targetSearchLayer[T]) ([]T, error) {
	combined := []T{}
	seen := map[string]struct{}{}
	for _, layer := range layers {
		if !layer.enabled {
			continue
		}
		candidates, err := layer.search(ctx)
		if err != nil {
			return nil, fmt.Errorf("%s %s (%T) failed: %w", layer.name, service, target, err)
		}
		if layer.filter != nil {
			candidates = layer.filter(candidates)
		}
		combined = appendUniqueByKey(combined, seen, candidates, keyFunc)
	}
	return combined, nil
}

func collectAlbumTargetCandidates(ctx context.Context, target TargetAdapter, source model.CanonicalAlbum, weights score.Weights) ([]model.CandidateAlbum, error) {
	return collectTargetSearchLayers(ctx, target, target.Service(), albumCandidateKey, albumTargetSearchLayers(target, source, weights)...)
}

func albumTargetSearchLayers(target TargetAdapter, source model.CanonicalAlbum, weights score.Weights) []targetSearchLayer[model.CandidateAlbum] {
	isrcs := collectISRCs(source)
	return []targetSearchLayer[model.CandidateAlbum]{
		{
			name:    "SearchByUPC",
			enabled: source.UPC != "",
			search: func(ctx context.Context) ([]model.CandidateAlbum, error) {
				return target.SearchByUPC(ctx, source.UPC)
			},
		},
		{
			name:    "SearchByISRC",
			enabled: len(isrcs) > 0,
			search: func(ctx context.Context) ([]model.CandidateAlbum, error) {
				return target.SearchByISRC(ctx, isrcs)
			},
		},
		{
			name:    "SearchByMetadata",
			enabled: true,
			search: func(ctx context.Context) ([]model.CandidateAlbum, error) {
				return target.SearchByMetadata(ctx, source)
			},
			filter: func(candidates []model.CandidateAlbum) []model.CandidateAlbum {
				return filterAlbumMetadataFallbackCandidates(target.Service(), source, candidates, weights)
			},
		},
	}
}

func collectSongTargetCandidates(ctx context.Context, target SongTargetAdapter, source model.CanonicalSong) ([]model.CandidateSong, error) {
	return collectTargetSearchLayers(ctx, target, target.Service(), songCandidateKey, songTargetSearchLayers(target, source)...)
}

func songTargetSearchLayers(target SongTargetAdapter, source model.CanonicalSong) []targetSearchLayer[model.CandidateSong] {
	return []targetSearchLayer[model.CandidateSong]{
		{
			name:    "SearchSongByISRC",
			enabled: source.ISRC != "",
			search: func(ctx context.Context) ([]model.CandidateSong, error) {
				return target.SearchSongByISRC(ctx, source.ISRC)
			},
		},
		{
			name:    "SearchSongByMetadata",
			enabled: true,
			search: func(ctx context.Context) ([]model.CandidateSong, error) {
				return target.SearchSongByMetadata(ctx, source)
			},
		},
	}
}

func filterAlbumMetadataFallbackCandidates(
	targetService model.ServiceName,
	source model.CanonicalAlbum,
	candidates []model.CandidateAlbum,
	weights score.Weights,
) []model.CandidateAlbum {
	if targetService != model.ServiceAppleMusic || len(candidates) == 0 {
		return candidates
	}

	filtered := make([]model.CandidateAlbum, 0, len(candidates))
	for _, candidate := range candidates {
		ranking := score.RankAlbums(source, []model.CandidateAlbum{candidate}, weights)
		if len(ranking.Ranked) == 0 {
			continue
		}
		ranked := ranking.Ranked[0]
		if ranked.Score <= 0 || !ranked.Evidence.HasTitleOrArtist() {
			continue
		}
		filtered = append(filtered, candidate)
	}
	return filtered
}
