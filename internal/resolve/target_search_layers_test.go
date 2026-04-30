package resolve

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
)

var errTargetSearchLayerBoom = errors.New("target search layer boom")

func TestCollectTargetSearchLayersPreservesOrderAndDeduplicates(t *testing.T) {
	candidates, err := collectTargetSearchLayers(
		context.Background(),
		newStubTargetAdapter(),
		model.ServiceSpotify,
		albumCandidateKey,
		targetSearchLayer[model.CandidateAlbum]{
			name:    "disabled",
			enabled: false,
			search: func(context.Context) ([]model.CandidateAlbum, error) {
				return []model.CandidateAlbum{{CandidateID: "disabled", CanonicalAlbum: model.CanonicalAlbum{Service: model.ServiceSpotify}}}, nil
			},
		},
		targetSearchLayer[model.CandidateAlbum]{
			name:    "first",
			enabled: true,
			search: func(context.Context) ([]model.CandidateAlbum, error) {
				return []model.CandidateAlbum{
					{CandidateID: "album-1", CanonicalAlbum: model.CanonicalAlbum{Service: model.ServiceSpotify}},
					{CandidateID: "album-2", CanonicalAlbum: model.CanonicalAlbum{Service: model.ServiceSpotify}},
				}, nil
			},
		},
		targetSearchLayer[model.CandidateAlbum]{
			name:    "second",
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

func TestCollectTargetSearchLayersWrapsLayerErrors(t *testing.T) {
	_, err := collectTargetSearchLayers(
		context.Background(),
		newStubTargetAdapter(),
		model.ServiceSpotify,
		albumCandidateKey,
		targetSearchLayer[model.CandidateAlbum]{
			name:    "SearchByUPC",
			enabled: true,
			search: func(context.Context) ([]model.CandidateAlbum, error) {
				return nil, errTargetSearchLayerBoom
			},
		},
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, errTargetSearchLayerBoom)
	assert.Contains(t, err.Error(), "SearchByUPC spotify")
}
