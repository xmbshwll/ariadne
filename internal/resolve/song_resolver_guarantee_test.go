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
	called := false
	resolver := NewSongs(
		[]SongSourceAdapter{stubSongSourceAdapter{}},
		[]SongTargetAdapter{sourceServiceSongTargetAdapter{called: &called}, stubSongTargetAdapter{}},
		score.DefaultSongWeights(),
	)

	resolution, err := resolver.ResolveSong(context.Background(), "https://open.spotify.com/track/track-1")
	require.NoError(t, err)
	assert.False(t, called)
	require.NotNil(t, resolution.Matches[model.ServiceAppleMusic].Best)
	_, ok := resolution.Matches[model.ServiceSpotify]
	assert.False(t, ok)
}

func TestSongResolverResolveSongReturnsTargetError(t *testing.T) {
	resolver := NewSongs(
		[]SongSourceAdapter{stubSongSourceAdapter{}},
		[]SongTargetAdapter{failingSongTargetAdapter{}},
		score.DefaultSongWeights(),
	)

	resolution, err := resolver.ResolveSong(context.Background(), "https://open.spotify.com/track/track-1")
	require.Error(t, err)
	assert.Nil(t, resolution)
	assert.ErrorIs(t, err, errTargetSearchBoom)
}
