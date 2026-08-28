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
}

// NewWithAdapters builds a Resolver from test-supplied adapters with the
// built-in Scoring weights, which no caller overrides.
func NewWithAdapters(set TestAdapters) *Resolver {
	return newResolver(
		set.AlbumSources,
		set.AlbumTargets,
		set.SongSources,
		set.SongTargets,
		score.DefaultWeights(),
		score.DefaultSongWeights(),
	)
}

// ConfigWeights reports the Scoring weights a Config resolves to. Ranking is
// Ariadne's decision rather than a caller knob, so the weights are unexported
// fields and only this seam can read them.
func ConfigWeights(config Config) (score.Weights, score.SongWeights) {
	return config.scoreWeights, config.songScoreWeights
}
