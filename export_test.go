package ariadne

import (
	"github.com/xmbshwll/ariadne/internal/adapters"
	"github.com/xmbshwll/ariadne/internal/score"
)

// Test-only seams for the external ariadne_test package.

// NormalizedConfig exposes configuration normalization, which is internal
// behavior with no public surface.
var NormalizedConfig = normalizedConfig

// NewResolverForTest builds a Resolver from test-supplied adapters. It exists
// so ariadne_test can pin Resolver behavior around adapter contract violations
// (nil parsed URL, nil entity, a failing Target Search): the public API has no
// adapter-authoring seam, so tests reach the constructor directly. Zero
// weights use the built-in Scoring defaults.
func NewResolverForTest(
	albumSources []adapters.Adapter,
	albumTargets []adapters.Adapter,
	songSources []adapters.Adapter,
	songTargets []adapters.Adapter,
) *Resolver {
	return newResolver(
		albumSources,
		albumTargets,
		songSources,
		songTargets,
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
