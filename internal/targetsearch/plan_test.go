package targetsearch_test

import (
	"context"
	"errors"
	"net"
	"net/url"
	"testing"

	targetsearch "github.com/xmbshwll/ariadne/internal/targetsearch"

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

func TestPlanPreservesOrderAndDeduplicates(t *testing.T) {
	plan := targetsearch.Plan[model.CandidateAlbum]{
		Target:       string(model.ServiceSpotify),
		CandidateKey: model.CandidateAlbum.SearchKey,
		Layers: []targetsearch.Layer[model.CandidateAlbum]{
			{
				Name:    "disabled",
				Enabled: false,
				Search: func(context.Context) ([]model.CandidateAlbum, error) {
					return []model.CandidateAlbum{{CandidateID: "disabled", CanonicalAlbum: model.CanonicalAlbum{Service: model.ServiceSpotify}}}, nil
				},
			},
			{
				Name:    "first",
				Enabled: true,
				Search: func(context.Context) ([]model.CandidateAlbum, error) {
					return []model.CandidateAlbum{
						{CandidateID: targetSearchAlbum1, CanonicalAlbum: model.CanonicalAlbum{Service: model.ServiceSpotify}},
						{CandidateID: targetSearchAlbum2, CanonicalAlbum: model.CanonicalAlbum{Service: model.ServiceSpotify}},
					}, nil
				},
			},
			{
				Name:    "second",
				Enabled: true,
				Search: func(context.Context) ([]model.CandidateAlbum, error) {
					return []model.CandidateAlbum{
						{CandidateID: targetSearchAlbum2, CanonicalAlbum: model.CanonicalAlbum{Service: model.ServiceSpotify}},
						{CandidateID: targetSearchAlbum3, CanonicalAlbum: model.CanonicalAlbum{Service: model.ServiceSpotify}},
					}, nil
				},
			},
		},
	}

	candidates, err := plan.Collect(context.Background())

	require.NoError(t, err)
	require.Len(t, candidates, 3)
	assert.Equal(t, targetSearchAlbum1, candidates[0].CandidateID)
	assert.Equal(t, targetSearchAlbum2, candidates[1].CandidateID)
	assert.Equal(t, targetSearchAlbum3, candidates[2].CandidateID)
}

func TestPlanSkipsLayerTimeoutsWhenParentContextIsActive(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "deadline exceeded", err: context.DeadlineExceeded},
		{name: "net timeout", err: &url.Error{
			Op:  "Get",
			URL: "https://api.example.test/albums",
			Err: &net.OpError{Op: "dial", Net: "tcp", Err: targetSearchTimeoutError{}},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := planWithRecoverableTimeout(tt.err)

			candidates, err := plan.Collect(context.Background())

			require.NoError(t, err)
			require.Len(t, candidates, 1)
			assert.Equal(t, targetSearchAlbum1, candidates[0].CandidateID)
		})
	}
}

func planWithRecoverableTimeout(err error) targetsearch.Plan[model.CandidateAlbum] {
	return targetsearch.Plan[model.CandidateAlbum]{
		Target:       string(model.ServiceSpotify),
		CandidateKey: model.CandidateAlbum.SearchKey,
		Layers: []targetsearch.Layer[model.CandidateAlbum]{
			{
				Name:    "SearchAlbumByUPC",
				Enabled: true,
				Search: func(context.Context) ([]model.CandidateAlbum, error) {
					return nil, err
				},
			},
			{
				Name:    "SearchAlbumByMetadata",
				Enabled: true,
				Search: func(context.Context) ([]model.CandidateAlbum, error) {
					return []model.CandidateAlbum{{CandidateID: targetSearchAlbum1, CanonicalAlbum: model.CanonicalAlbum{Service: model.ServiceSpotify}}}, nil
				},
			},
		},
	}
}

func TestPlanKeepsParentContextDeadlineFatal(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()
	plan := targetsearch.Plan[model.CandidateAlbum]{
		Target:       string(model.ServiceSpotify),
		CandidateKey: model.CandidateAlbum.SearchKey,
		Layers: []targetsearch.Layer[model.CandidateAlbum]{
			{
				Name:    "SearchAlbumByMetadata",
				Enabled: true,
				Search: func(context.Context) ([]model.CandidateAlbum, error) {
					return nil, context.DeadlineExceeded
				},
			},
		},
	}

	_, err := plan.Collect(ctx)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestPlanWrapsLayerErrors(t *testing.T) {
	plan := targetsearch.Plan[model.CandidateAlbum]{
		Target:       string(model.ServiceSpotify),
		CandidateKey: model.CandidateAlbum.SearchKey,
		Layers: []targetsearch.Layer[model.CandidateAlbum]{
			{
				Name:    "SearchAlbumByUPC",
				Enabled: true,
				Search: func(context.Context) ([]model.CandidateAlbum, error) {
					return nil, errTargetSearchLayerBoom
				},
			},
		},
	}

	_, err := plan.Collect(context.Background())

	require.Error(t, err)
	assert.ErrorIs(t, err, errTargetSearchLayerBoom)
	assert.Contains(t, err.Error(), "SearchAlbumByUPC spotify")
}
