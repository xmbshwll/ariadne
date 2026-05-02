package score

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScoreTitleSignalUsesExactAndCoreEvidence(t *testing.T) {
	weights := titleSignalWeights{exact: 25, core: 15}

	exact := scoreTitleSignal("Abbey Road", "", "Abbey Road", "", weights)
	assert.Equal(t, 25, exact.value)
	assert.Equal(t, "title exact match", exact.reason)
	assert.True(t, exact.evidence.Title)

	core := scoreTitleSignal("Abbey Road (Live)", "", "Abbey Road", "", weights)
	assert.Equal(t, 15, core.value)
	assert.Equal(t, "core title match", core.reason)
	assert.True(t, core.evidence.Title)
}

func TestScoreArtistSignalUsesNormalizedFallbacks(t *testing.T) {
	weights := artistSignalWeights{primaryExact: 20, overlap: 10}

	primary := scoreArtistSignal([]string{"The Beatles"}, nil, []string{"the beatles"}, nil, weights)
	assert.Equal(t, 20, primary.value)
	assert.Equal(t, "primary artist exact match", primary.reason)
	assert.True(t, primary.evidence.Artist)

	overlap := scoreArtistSignal([]string{"Paul McCartney", "The Beatles"}, nil, []string{"The Beatles"}, nil, weights)
	assert.Equal(t, 10, overlap.value)
	assert.Equal(t, "artist overlap", overlap.reason)
	assert.True(t, overlap.evidence.Artist)
}

func TestSharedScoreSignalsPreserveReasonText(t *testing.T) {
	assert.Equal(t, "release date exact match", scoreReleaseDateSignal("2024-01-01", "2024-01-01", releaseDateSignalWeights{exact: 10, year: 5}).reason)
	assert.Equal(t, "release year match", scoreReleaseDateSignal("2024-01-01", "2024-12-31", releaseDateSignalWeights{exact: 10, year: 5}).reason)
	assert.Equal(t, "duration near match", scoreDurationSignal(100_000, 101_000, 10).reason)
	assert.Equal(t, "explicit mismatch", scoreExplicitSignal(true, false, -10).reason)
	assert.Equal(t, "edition mismatch", scoreEditionHintSignal([]string{"deluxe"}, []string{"standard"}, -20).reason)
	assert.Equal(t, "album title exact match", scoreNormalizedExactSignal("Album", "", "Album", "", 5, "album title exact match").reason)
}
