package targetsearch_test

import (
	"context"
	"errors"
	"testing"

	"github.com/xmbshwll/ariadne/internal/targetsearch"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	errMetadataQueryFirstSearch  = errors.New("first search failed")
	errMetadataQuerySecondSearch = errors.New("second search failed")
	errMetadataQuerySearch       = errors.New("search failed")
	errMetadataQueryCandidate    = errors.New("candidate failed")
)

type metadataQueryTestItem struct {
	ID    string
	Value string
}

type metadataQueryContextKey struct{}

func TestMetadataQuerySearchReturnsEmptySliceForNoQueries(t *testing.T) {
	candidates, err := (targetsearch.MetadataQuerySearch[metadataQueryTestItem, string]{}).Collect(context.Background())

	require.NoError(t, err)
	assert.NotNil(t, candidates)
	assert.Empty(t, candidates)
}

func TestMetadataQuerySearchPassesContextToSearchAndBuildCandidate(t *testing.T) {
	ctx := context.WithValue(context.Background(), metadataQueryContextKey{}, "metadata-context")
	targetSearch := targetsearch.MetadataQuerySearch[metadataQueryTestItem, string]{
		Queries: []string{"first"},
		Limit:   1,
		Search: func(ctx context.Context, query string) ([]metadataQueryTestItem, error) {
			assert.Equal(t, "metadata-context", ctx.Value(metadataQueryContextKey{}))
			assert.Equal(t, "first", query)
			return []metadataQueryTestItem{{ID: "a", Value: "A"}}, nil
		},
		ItemID: func(item metadataQueryTestItem) string {
			return item.ID
		},
		BuildCandidate: func(ctx context.Context, item metadataQueryTestItem) (string, error) {
			assert.Equal(t, "metadata-context", ctx.Value(metadataQueryContextKey{}))
			return item.Value, nil
		},
	}

	candidates, err := targetSearch.Collect(ctx)

	require.NoError(t, err)
	assert.Equal(t, []string{"A"}, candidates)
}

func TestMetadataQuerySearchDeduplicatesAndStopsAtLimit(t *testing.T) {
	searches := []string{}

	candidates, err := targetsearch.MetadataQuerySearch[metadataQueryTestItem, string]{
		Queries: []string{"first", "second"},
		Limit:   3,
		Search: func(_ context.Context, query string) ([]metadataQueryTestItem, error) {
			searches = append(searches, query)
			return map[string][]metadataQueryTestItem{
				"first": {
					{ID: " a ", Value: "A"},
					{ID: "", Value: "blank"},
					{ID: "b", Value: "B"},
				},
				"second": {
					{ID: "a", Value: "duplicate"},
					{ID: "c", Value: "C"},
					{ID: "d", Value: "D"},
				},
			}[query], nil
		},
		ItemID: func(item metadataQueryTestItem) string {
			return item.ID
		},
		BuildCandidate: func(_ context.Context, item metadataQueryTestItem) (string, error) {
			return item.Value, nil
		},
	}.Collect(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []string{"A", "B", "C"}, candidates)
	assert.Equal(t, []string{"first", "second"}, searches)
}

func TestMetadataQuerySearchReturnsFirstSearchErrorWhenNothingCollected(t *testing.T) {
	_, err := targetsearch.MetadataQuerySearch[metadataQueryTestItem, string]{
		Queries: []string{"first", "second"},
		Limit:   2,
		Search: func(_ context.Context, query string) ([]metadataQueryTestItem, error) {
			if query == "first" {
				return nil, errMetadataQueryFirstSearch
			}
			return nil, errMetadataQuerySecondSearch
		},
		ItemID: func(item metadataQueryTestItem) string {
			return item.ID
		},
		BuildCandidate: func(_ context.Context, item metadataQueryTestItem) (string, error) {
			return item.Value, nil
		},
	}.Collect(context.Background())

	assert.ErrorIs(t, err, errMetadataQueryFirstSearch)
}

func TestMetadataQuerySearchCanStopAfterSearchError(t *testing.T) {
	_, err := targetsearch.MetadataQuerySearch[metadataQueryTestItem, string]{
		Queries: []string{"first", "second"},
		Limit:   2,
		Search: func(context.Context, string) ([]metadataQueryTestItem, error) {
			return nil, errMetadataQuerySearch
		},
		ItemID: func(item metadataQueryTestItem) string {
			return item.ID
		},
		BuildCandidate: func(_ context.Context, item metadataQueryTestItem) (string, error) {
			return item.Value, nil
		},
		ContinueAfterSearchError: func(collected int) bool {
			return collected > 0
		},
	}.Collect(context.Background())

	assert.ErrorIs(t, err, errMetadataQuerySearch)
}

func TestMetadataQuerySearchReturnsFirstCandidateErrorWhenNothingBuilds(t *testing.T) {
	_, err := targetsearch.MetadataQuerySearch[metadataQueryTestItem, string]{
		Queries: []string{"first"},
		Limit:   2,
		Search: func(context.Context, string) ([]metadataQueryTestItem, error) {
			return []metadataQueryTestItem{{ID: "a", Value: "A"}}, nil
		},
		ItemID: func(item metadataQueryTestItem) string {
			return item.ID
		},
		BuildCandidate: func(context.Context, metadataQueryTestItem) (string, error) {
			return "", errMetadataQueryCandidate
		},
	}.Collect(context.Background())

	assert.ErrorIs(t, err, errMetadataQueryCandidate)
}
