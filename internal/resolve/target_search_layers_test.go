package resolve

import (
	"context"
	"errors"
	"net"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
)

var errTargetSearchLayerBoom = errors.New("target search layer boom")

type targetSearchTimeoutError struct{}

func (targetSearchTimeoutError) Error() string { return "target search timeout" }
func (targetSearchTimeoutError) Timeout() bool { return true }

const (
	targetSearchAlbum1 = "album-1"
	targetSearchAlbum2 = "album-2"
	targetSearchAlbum3 = "album-3"
)

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
						{CandidateID: targetSearchAlbum1, CanonicalAlbum: model.CanonicalAlbum{Service: model.ServiceSpotify}},
						{CandidateID: targetSearchAlbum2, CanonicalAlbum: model.CanonicalAlbum{Service: model.ServiceSpotify}},
					}, nil
				},
			},
			{
				name:    "second",
				enabled: true,
				search: func(context.Context) ([]model.CandidateAlbum, error) {
					return []model.CandidateAlbum{
						{CandidateID: targetSearchAlbum2, CanonicalAlbum: model.CanonicalAlbum{Service: model.ServiceSpotify}},
						{CandidateID: targetSearchAlbum3, CanonicalAlbum: model.CanonicalAlbum{Service: model.ServiceSpotify}},
					}, nil
				},
			},
		},
	}

	candidates, err := plan.collect(context.Background())

	require.NoError(t, err)
	require.Len(t, candidates, 3)
	assert.Equal(t, targetSearchAlbum1, candidates[0].CandidateID)
	assert.Equal(t, targetSearchAlbum2, candidates[1].CandidateID)
	assert.Equal(t, targetSearchAlbum3, candidates[2].CandidateID)
}

func TestTargetSearchPlanSkipsLayerTimeoutsWhenParentContextIsActive(t *testing.T) {
	plan := targetSearchPlanWithRecoverableTimeout(context.DeadlineExceeded)

	candidates, err := plan.collect(context.Background())

	require.NoError(t, err)
	require.Len(t, candidates, 1)
	assert.Equal(t, targetSearchAlbum1, candidates[0].CandidateID)
}

func TestTargetSearchPlanSkipsNetErrorTimeoutsWhenParentContextIsActive(t *testing.T) {
	plan := targetSearchPlanWithRecoverableTimeout(&url.Error{
		Op:  "Get",
		URL: "https://api.example.test/albums",
		Err: &net.OpError{Op: "dial", Net: "tcp", Err: targetSearchTimeoutError{}},
	})

	candidates, err := plan.collect(context.Background())

	require.NoError(t, err)
	require.Len(t, candidates, 1)
	assert.Equal(t, targetSearchAlbum1, candidates[0].CandidateID)
}

func targetSearchPlanWithRecoverableTimeout(err error) targetSearchPlan[model.CandidateAlbum] {
	return targetSearchPlan[model.CandidateAlbum]{
		target:  newStubTargetAdapter(),
		service: model.ServiceSpotify,
		keyFunc: albumCandidateKey,
		layers: []targetSearchLayer[model.CandidateAlbum]{
			{
				name:    "SearchByUPC",
				enabled: true,
				search: func(context.Context) ([]model.CandidateAlbum, error) {
					return nil, err
				},
			},
			{
				name:    "SearchByMetadata",
				enabled: true,
				search: func(context.Context) ([]model.CandidateAlbum, error) {
					return []model.CandidateAlbum{{CandidateID: targetSearchAlbum1, CanonicalAlbum: model.CanonicalAlbum{Service: model.ServiceSpotify}}}, nil
				},
			},
		},
	}
}

func TestTargetSearchPlanKeepsParentContextDeadlineFatal(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()
	plan := targetSearchPlan[model.CandidateAlbum]{
		target:  newStubTargetAdapter(),
		service: model.ServiceSpotify,
		keyFunc: albumCandidateKey,
		layers: []targetSearchLayer[model.CandidateAlbum]{
			{
				name:    "SearchByMetadata",
				enabled: true,
				search: func(context.Context) ([]model.CandidateAlbum, error) {
					return nil, context.DeadlineExceeded
				},
			},
		},
	}

	_, err := plan.collect(ctx)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
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
