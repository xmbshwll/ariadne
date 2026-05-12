package adapterutil

import (
	"context"
	"strings"
)

func TrimmedNonEmptyStrings(values []string) []string {
	trimmed := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		trimmed = append(trimmed, value)
	}
	return trimmed
}

func CollectCandidates[Input any, Candidate any](
	items []Input,
	limit int,
	itemID func(Input) string,
	fetch func(Input) (Candidate, error),
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

		candidate, err := fetch(item)
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

func CollectCandidatesWithContext[Input any, Candidate any](
	ctx context.Context,
	items []Input,
	limit int,
	itemID func(Input) string,
	fetch func(context.Context, Input) (Candidate, error),
) ([]Candidate, error) {
	return CollectCandidates(items, limit, itemID, func(item Input) (Candidate, error) {
		return fetch(ctx, item)
	})
}

type MetadataQueryTargetSearch[Item any, Candidate any] struct {
	Queries                  []string
	Limit                    int
	Search                   func(context.Context, string) ([]Item, error)
	ItemID                   func(Item) string
	BuildCandidate           func(context.Context, Item) (Candidate, error)
	ContinueAfterSearchError func(collected int) bool
}

func (search MetadataQueryTargetSearch[Item, Candidate]) Collect(ctx context.Context) ([]Candidate, error) {
	if len(search.Queries) == 0 {
		return []Candidate{}, nil
	}

	collector := MetadataQueryCandidateCollector[Item, Candidate]{
		Queries: search.Queries,
		Limit:   search.Limit,
		Search: func(query string) ([]Item, error) {
			return search.Search(ctx, query)
		},
		ItemID: search.ItemID,
		BuildCandidate: func(item Item) (Candidate, error) {
			return search.BuildCandidate(ctx, item)
		},
		ContinueAfterSearchError: search.ContinueAfterSearchError,
	}
	return CollectMetadataQueryCandidates(collector)
}

type MetadataQueryCandidateCollector[Item any, Candidate any] struct {
	Queries                  []string
	Limit                    int
	Search                   func(string) ([]Item, error)
	ItemID                   func(Item) string
	BuildCandidate           func(Item) (Candidate, error)
	ContinueAfterSearchError func(collected int) bool
}

func CollectMetadataQueryCandidates[Item any, Candidate any](
	collector MetadataQueryCandidateCollector[Item, Candidate],
) ([]Candidate, error) {
	if collector.Limit <= 0 {
		return []Candidate{}, nil
	}

	candidates := make([]Candidate, 0, collector.Limit)
	seen := make(map[string]struct{}, collector.Limit)
	var firstSearchErr error
	var firstCandidateErr error

	for _, query := range collector.Queries {
		items, err := collector.Search(query)
		if err != nil {
			if !continueAfterMetadataSearchError(collector.ContinueAfterSearchError, len(candidates)) {
				return nil, err
			}
			if firstSearchErr == nil {
				firstSearchErr = err
			}
			continue
		}

		for _, item := range items {
			id := strings.TrimSpace(collector.ItemID(item))
			if id == "" {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}

			candidate, err := collector.BuildCandidate(item)
			if err != nil {
				if firstCandidateErr == nil {
					firstCandidateErr = err
				}
				continue
			}
			candidates = append(candidates, candidate)
			if len(candidates) >= collector.Limit {
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

func continueAfterMetadataSearchError(shouldContinue func(int) bool, collected int) bool {
	if shouldContinue == nil {
		return true
	}
	return shouldContinue(collected)
}
