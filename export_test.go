package ariadne

import (
	"github.com/xmbshwll/ariadne/internal/adapters"
	"github.com/xmbshwll/ariadne/internal/score"
)

// Test-only seams for the external ariadne_test package: configuration
// normalization is internal behavior with no public surface.
var NormalizedConfig = normalizedConfig

// TestAdapters is the test-only counterpart of the built-in adapter set.
//
// The public API deliberately has no adapter-authoring seam: ariadne.Resolver
// is built from the Provider Catalog via New, and internal/adapters.Adapter is
// internal. Tests that need to pin Resolver behavior around adapter contract
// violations (nil parsed URL, nil entity, a failing Target Search) build mocks
// of that internal interface and hand them here.
type TestAdapters struct {
	AlbumSources []adapters.Adapter
	AlbumTargets []adapters.Adapter
	SongSources  []adapters.Adapter
	SongTargets  []adapters.Adapter
	Weights      score.Weights
	SongWeights  score.SongWeights
}

// NewWithAdapters builds a Resolver from test-supplied adapters. Zero weights
// use the defaults, matching the documented behavior of the default set.
func NewWithAdapters(set TestAdapters) *Resolver {
	weights := set.Weights
	if weights == (score.Weights{}) {
		weights = score.DefaultWeights()
	}
	songWeights := set.SongWeights
	if songWeights == (score.SongWeights{}) {
		songWeights = score.DefaultSongWeights()
	}
	return newResolver(set.AlbumSources, set.AlbumTargets, set.SongSources, set.SongTargets, weights, songWeights)
}
