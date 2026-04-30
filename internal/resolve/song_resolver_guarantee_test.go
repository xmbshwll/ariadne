package resolve

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/score"
)

func TestSongResolverResolveSongExcludesSourceServiceFromTargets(t *testing.T) {
	sourceServiceTarget := newSourceServiceSongTargetAdapter()
	resolver := NewSongs(
		[]SongSourceAdapter{newStubSongSourceAdapter()},
		[]SongTargetAdapter{sourceServiceTarget, newStubSongTargetAdapter()},
		score.DefaultSongWeights(),
	)

	resolution, err := resolver.ResolveSong(context.Background(), "https://open.spotify.com/track/track-1")
	require.NoError(t, err)
	assert.Zero(t, sourceServiceTarget.CallCount())
	require.NotNil(t, resolution.Matches[model.ServiceAppleMusic].Best)
	_, ok := resolution.Matches[model.ServiceSpotify]
	assert.False(t, ok)
}

func TestSongResolverResolveSongReturnsTargetError(t *testing.T) {
	resolver := NewSongs(
		[]SongSourceAdapter{newStubSongSourceAdapter()},
		[]SongTargetAdapter{newFailingSongTargetAdapter()},
		score.DefaultSongWeights(),
	)

	resolution, err := resolver.ResolveSong(context.Background(), "https://open.spotify.com/track/track-1")
	require.Error(t, err)
	assert.Nil(t, resolution)
	assert.ErrorIs(t, err, errTargetSearchBoom)
}
