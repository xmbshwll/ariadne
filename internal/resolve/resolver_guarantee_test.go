package resolve

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/score"
)

func TestResolverResolveAlbumSkipsSourceServiceAsTarget(t *testing.T) {
	resolver := New(
		[]SourceAdapter{newStubSourceAdapter()},
		[]TargetAdapter{
			newSourceServiceTargetAdapter(),
			newStubTargetAdapter(),
		},
		score.DefaultWeights(),
	)

	resolution, err := resolver.ResolveAlbum(context.Background(), "https://www.deezer.com/album/12047952")
	require.NoError(t, err)
	require.NotNil(t, resolution.Matches[model.ServiceSpotify].Best)
	_, ok := resolution.Matches[model.ServiceDeezer]
	assert.False(t, ok)
}

func TestResolverResolveAlbumReturnsTargetError(t *testing.T) {
	resolver := New(
		[]SourceAdapter{newStubSourceAdapter()},
		[]TargetAdapter{newFailingTargetAdapter()},
		score.DefaultWeights(),
	)

	resolution, err := resolver.ResolveAlbum(context.Background(), "https://www.deezer.com/album/12047952")
	require.Error(t, err)
	assert.Nil(t, resolution)
	assert.ErrorIs(t, err, errTargetSearchBoom)
}
