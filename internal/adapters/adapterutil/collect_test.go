package adapterutil

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type metadataQueryTestItem struct {
	ID    string
	Value string
}

func TestCollectMetadataQueryCandidatesDeduplicatesAndStopsAtLimit(t *testing.T) {
	searches := []string{}

	candidates, err := CollectMetadataQueryCandidates(MetadataQueryCandidateCollector[metadataQueryTestItem, string]{
		Queries: []string{"first", "second"},
		Limit:   3,
		Search: func(query string) ([]metadataQueryTestItem, error) {
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
		BuildCandidate: func(item metadataQueryTestItem) (string, error) {
			return item.Value, nil
		},
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"A", "B", "C"}, candidates)
	assert.Equal(t, []string{"first", "second"}, searches)
}

func TestCollectMetadataQueryCandidatesReturnsFirstSearchErrorWhenNothingCollected(t *testing.T) {
	errFirst := errors.New("first search failed")
	errSecond := errors.New("second search failed")

	_, err := CollectMetadataQueryCandidates(MetadataQueryCandidateCollector[metadataQueryTestItem, string]{
		Queries: []string{"first", "second"},
		Limit:   2,
		Search: func(query string) ([]metadataQueryTestItem, error) {
			if query == "first" {
				return nil, errFirst
			}
			return nil, errSecond
		},
		ItemID: func(item metadataQueryTestItem) string {
			return item.ID
		},
		BuildCandidate: func(item metadataQueryTestItem) (string, error) {
			return item.Value, nil
		},
	})

	assert.ErrorIs(t, err, errFirst)
}

func TestCollectMetadataQueryCandidatesCanStopAfterSearchError(t *testing.T) {
	errSearch := errors.New("search failed")

	_, err := CollectMetadataQueryCandidates(MetadataQueryCandidateCollector[metadataQueryTestItem, string]{
		Queries: []string{"first", "second"},
		Limit:   2,
		Search: func(string) ([]metadataQueryTestItem, error) {
			return nil, errSearch
		},
		ItemID: func(item metadataQueryTestItem) string {
			return item.ID
		},
		BuildCandidate: func(item metadataQueryTestItem) (string, error) {
			return item.Value, nil
		},
		ContinueAfterSearchError: func(collected int) bool {
			return collected > 0
		},
	})

	assert.ErrorIs(t, err, errSearch)
}

func TestCollectMetadataQueryCandidatesReturnsFirstCandidateErrorWhenNothingBuilds(t *testing.T) {
	errCandidate := errors.New("candidate failed")

	_, err := CollectMetadataQueryCandidates(MetadataQueryCandidateCollector[metadataQueryTestItem, string]{
		Queries: []string{"first"},
		Limit:   2,
		Search: func(string) ([]metadataQueryTestItem, error) {
			return []metadataQueryTestItem{{ID: "a", Value: "A"}}, nil
		},
		ItemID: func(item metadataQueryTestItem) string {
			return item.ID
		},
		BuildCandidate: func(metadataQueryTestItem) (string, error) {
			return "", errCandidate
		},
	})

	assert.ErrorIs(t, err, errCandidate)
}
