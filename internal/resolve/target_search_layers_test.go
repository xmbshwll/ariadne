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

func TestTargetSearchPlanPreservesOrderAndDeduplicates(t *testing.T) {
	plan := targetSearchPlan[model.CandidateAlbum]{
		target:  newStubTargetAdapter(),
		service: model.ServiceSpotify,
		keyFunc: albumCandidateKey,
		layers: []targetSearchLayer[model.CandidateAlbum]{
			{
				name:    "disabled",
				enabled: false,
				search: func(context.Context) ([]model.CandidateAlbum, error) {
					return []model.CandidateAlbum{{CandidateID: "disabled", CanonicalAlbum: model.CanonicalAlbum{Service: model.ServiceSpotify}}}, nil
				},
			},
			{
				name:    "first",
				enabled: true,
				search: func(context.Context) ([]model.CandidateAlbum, error) {
					return []model.CandidateAlbum{
						{CandidateID: "album-1", CanonicalAlbum: model.CanonicalAlbum{Service: model.ServiceSpotify}},
						{CandidateID: "album-2", CanonicalAlbum: model.CanonicalAlbum{Service: model.ServiceSpotify}},
					}, nil
				},
			},
			{
				name:    "second",
				enabled: true,
				search: func(context.Context) ([]model.CandidateAlbum, error) {
					return []model.CandidateAlbum{
						{CandidateID: "album-2", CanonicalAlbum: model.CanonicalAlbum{Service: model.ServiceSpotify}},
						{CandidateID: "album-3", CanonicalAlbum: model.CanonicalAlbum{Service: model.ServiceSpotify}},
					}, nil
				},
			},
		},
	}

	candidates, err := plan.collect(context.Background())

	require.NoError(t, err)
	require.Len(t, candidates, 3)
	assert.Equal(t, "album-1", candidates[0].CandidateID)
	assert.Equal(t, "album-2", candidates[1].CandidateID)
	assert.Equal(t, "album-3", candidates[2].CandidateID)
}

func TestTargetSearchPlanWrapsLayerErrors(t *testing.T) {
	plan := targetSearchPlan[model.CandidateAlbum]{
		target:  newStubTargetAdapter(),
		service: model.ServiceSpotify,
		keyFunc: albumCandidateKey,
		layers: []targetSearchLayer[model.CandidateAlbum]{
			{
				name:    "SearchByUPC",
				enabled: true,
				search: func(context.Context) ([]model.CandidateAlbum, error) {
					return nil, errTargetSearchLayerBoom
				},
			},
		},
	}

	_, err := plan.collect(context.Background())

	require.Error(t, err)
	assert.ErrorIs(t, err, errTargetSearchLayerBoom)
	assert.Contains(t, err.Error(), "SearchByUPC spotify")
}
