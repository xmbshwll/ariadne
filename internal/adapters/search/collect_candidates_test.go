package search_test

// CollectCandidates is the seam seven providers lean on to hydrate wire items
// into candidates; this unit table pins its selection rules directly instead
// of only through provider HTTP tests.

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xmbshwll/ariadne/internal/adapters/search"
)

var errCollectFetchFailed = errors.New("fetch failed")

type collectItem struct {
	id string
	ok bool
}

// collectFixtureServer-ish fetch function: one item fails when ok is false.
func makeFetch(calls *[]string) func(context.Context, collectItem) (string, error) {
	return func(_ context.Context, item collectItem) (string, error) {
		*calls = append(*calls, item.id)
		if !item.ok {
			return "", errCollectFetchFailed
		}
		return "candidate-" + item.id, nil
	}
}

func TestCollectCandidates(t *testing.T) {
	tests := []struct {
		name           string
		limit          int
		items          []collectItem
		wantCandidates []string
		wantErr        error
		wantCalled     []string
	}{
		{
			name:           "collects one candidate per item in order",
			limit:          5,
			items:          []collectItem{{id: "a", ok: true}, {id: "b", ok: true}},
			wantCandidates: []string{"candidate-a", "candidate-b"},
			wantCalled:     []string{"a", "b"},
		},
		{
			name:           "skips items with blank ids",
			limit:          5,
			items:          []collectItem{{id: "  ", ok: true}, {id: "a", ok: true}},
			wantCandidates: []string{"candidate-a"},
			wantCalled:     []string{"a"},
		},
		{
			name:           "skips duplicate ids",
			limit:          5,
			items:          []collectItem{{id: "a", ok: true}, {id: "a", ok: true}, {id: "b", ok: true}},
			wantCandidates: []string{"candidate-a", "candidate-b"},
			wantCalled:     []string{"a", "b"},
		},
		{
			name:           "tolerates a mid-stream fetch failure",
			limit:          5,
			items:          []collectItem{{id: "a", ok: true}, {id: "bad", ok: false}, {id: "b", ok: true}},
			wantCandidates: []string{"candidate-a", "candidate-b"},
			wantCalled:     []string{"a", "bad", "b"},
		},
		{
			name:       "returns the first fetch error only when nothing was collected",
			limit:      5,
			items:      []collectItem{{id: "bad", ok: false}},
			wantErr:    errCollectFetchFailed,
			wantCalled: []string{"bad"},
		},
		{
			name:           "stops at the limit",
			limit:          2,
			items:          []collectItem{{id: "a", ok: true}, {id: "b", ok: true}, {id: "c", ok: true}},
			wantCandidates: []string{"candidate-a", "candidate-b"},
			wantCalled:     []string{"a", "b"},
		},
		{
			name:           "zero limit answers empty without fetching",
			limit:          0,
			items:          []collectItem{{id: "a", ok: true}},
			wantCandidates: []string{},
			wantCalled:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls []string
			candidates, err := search.CollectCandidates(
				context.Background(),
				tt.items,
				tt.limit,
				func(item collectItem) string { return item.id },
				makeFetch(&calls),
			)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr, tt.name)
				return
			}
			require.NoError(t, err, tt.name)
			assert.Equal(t, tt.wantCandidates, candidates, tt.name)
			assert.Equal(t, tt.wantCalled, calls, tt.name)
		})
	}
}

// TestCollectCandidatesCarriesTheContext pins that the caller's context
// reaches the fetch function: the loop itself does not check ctx, so honoring
// cancellation is the fetch function's job - the contract providers rely on
// when their fetches carry timeouts.
func TestCollectCandidatesCarriesTheContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var gotCtx context.Context
	candidates, err := search.CollectCandidates(ctx,
		[]collectItem{{id: "a", ok: true}}, 5,
		func(item collectItem) string { return item.id },
		func(ctx context.Context, item collectItem) (string, error) {
			gotCtx = ctx
			return "candidate-" + item.id, nil
		})

	require.NoError(t, err)
	assert.ErrorIs(t, gotCtx.Err(), context.Canceled)
	assert.Len(t, candidates, 1)
}
