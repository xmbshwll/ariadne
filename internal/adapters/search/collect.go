// Package search runs the Target Search of one provider: the Metadata Queries
// it issues, the per-item fetches each result needs, and the deduplicated
// candidate list that comes out.
package search

import (
	"context"
	"strings"
)

// CollectCandidates fetches one candidate per item, skipping items with empty
// or duplicate IDs and tolerating per-item fetch errors. It returns the first
// fetch error only when no candidate was collected.
func CollectCandidates[Input any, Candidate any](
	ctx context.Context,
	items []Input,
	limit int,
	itemID func(Input) string,
	fetch func(context.Context, Input) (Candidate, error),
) ([]Candidate, error) {
	if limit <= 0 {
		return []Candidate{}, nil
	}

	results := make([]Candidate, 0, min(len(items), limit))
	seen := make(map[string]struct{}, len(items))
	var firstErr error
	for _, item := range items {
		id := strings.TrimSpace(itemID(item))
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}

		candidate, err := fetch(ctx, item)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		results = append(results, candidate)
		if len(results) >= limit {
			return results, nil
		}
	}
	if len(results) == 0 && firstErr != nil {
		return nil, firstErr
	}
	return results, nil
}

// MetadataQuerySearch runs one Metadata Query Target Search: for each query it
// searches, then builds deduplicated candidates up to Limit. Per-item build
// errors are tolerated; a search error aborts unless ContinueAfterSearchError
// approves. The first error surfaces only when no candidate was collected.
type MetadataQuerySearch[Item any, Candidate any] struct {
	Queries                  []string
	Limit                    int
	Search                   func(context.Context, string) ([]Item, error)
	ItemID                   func(Item) string
	BuildCandidate           func(context.Context, Item) (Candidate, error)
	ContinueAfterSearchError func(collected int) bool
}

func (search MetadataQuerySearch[Item, Candidate]) Collect(ctx context.Context) ([]Candidate, error) {
	if search.Limit <= 0 || len(search.Queries) == 0 {
		return []Candidate{}, nil
	}

	candidates := make([]Candidate, 0, search.Limit)
	seen := make(map[string]struct{}, search.Limit)
	var firstSearchErr error
	var firstCandidateErr error

	for _, query := range search.Queries {
		items, err := search.Search(ctx, query)
		if err != nil {
			if search.ContinueAfterSearchError != nil && !search.ContinueAfterSearchError(len(candidates)) {
				return nil, err
			}
			if firstSearchErr == nil {
				firstSearchErr = err
			}
			continue
		}

		for _, item := range items {
			id := strings.TrimSpace(search.ItemID(item))
			if id == "" {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}

			candidate, err := search.BuildCandidate(ctx, item)
			if err != nil {
				if firstCandidateErr == nil {
					firstCandidateErr = err
				}
				continue
			}
			candidates = append(candidates, candidate)
			if len(candidates) >= search.Limit {
				return candidates, nil
			}
		}
	}

	if len(candidates) == 0 {
		if firstSearchErr != nil {
			return nil, firstSearchErr
		}
		if firstCandidateErr != nil {
			return nil, firstCandidateErr
		}
	}
	return candidates, nil
}
