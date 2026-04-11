package adapterutil

import "strings"

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

func CollectCandidates[Input any, Candidate any](items []Input, limit int, itemID func(Input) string, fetch func(Input) (Candidate, error)) ([]Candidate, error) {
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
