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
	called := false
	resolver := New(
		[]SourceAdapter{newStubSourceAdapter()},
		[]TargetAdapter{
			newSourceServiceTargetAdapter(&called),
			newStubTargetAdapter(),
		},
		score.DefaultWeights(),
	)

	resolution, err := resolver.ResolveAlbum(context.Background(), "https://www.deezer.com/album/12047952")
	require.NoError(t, err)
	assert.False(t, called)
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

func TestCollectCandidateLayersPreservesFirstSeenOrdering(t *testing.T) {
	candidates, err := collectCandidateLayers(context.Background(), albumCandidateKey,
		candidateLayer[model.CandidateAlbum]{
			enabled: true,
			search: func(context.Context) ([]model.CandidateAlbum, error) {
				return []model.CandidateAlbum{
					{CandidateID: "album-1", CanonicalAlbum: model.CanonicalAlbum{Service: model.ServiceSpotify}},
					{CandidateID: "album-2", CanonicalAlbum: model.CanonicalAlbum{Service: model.ServiceSpotify}},
				}, nil
			},
		},
		candidateLayer[model.CandidateAlbum]{
			enabled: true,
			search: func(context.Context) ([]model.CandidateAlbum, error) {
				return []model.CandidateAlbum{
					{CandidateID: "album-2", CanonicalAlbum: model.CanonicalAlbum{Service: model.ServiceSpotify}},
					{CandidateID: "album-3", CanonicalAlbum: model.CanonicalAlbum{Service: model.ServiceSpotify}},
				}, nil
			},
		},
	)
	require.NoError(t, err)
	require.Len(t, candidates, 3)
	assert.Equal(t, "album-1", candidates[0].CandidateID)
	assert.Equal(t, "album-2", candidates[1].CandidateID)
	assert.Equal(t, "album-3", candidates[2].CandidateID)
}
