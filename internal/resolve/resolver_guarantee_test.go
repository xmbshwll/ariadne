package resolve_test

import (
	"context"
	"testing"

	"github.com/xmbshwll/ariadne/internal/adapters"
	resolve "github.com/xmbshwll/ariadne/internal/resolve"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/score"
)

func TestResolverResolveAlbumSkipsSourceServiceAsTarget(t *testing.T) {
	resolver := resolve.New(
		[]adapters.Adapter{newStubSourceAdapter()},
		[]adapters.Adapter{
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

func TestResolverResolveAlbumSurfacesTargetErrorPerMatch(t *testing.T) {
	resolver := resolve.New(
		[]adapters.Adapter{newStubSourceAdapter()},
		[]adapters.Adapter{newStubTargetAdapter(), newFailingTargetAdapter()},
		score.DefaultWeights(),
	)

	resolution, err := resolver.ResolveAlbum(context.Background(), "https://www.deezer.com/album/12047952")
	require.NoError(t, err)
	require.NotNil(t, resolution)

	// One failing Target Search surfaces on its own match without affecting others.
	failing := resolution.Matches[model.ServiceBandcamp]
	assert.ErrorIs(t, failing.Err, errTargetSearchBoom)
	assert.Nil(t, failing.Best)
	require.NotNil(t, resolution.Matches[model.ServiceSpotify].Best)
}
