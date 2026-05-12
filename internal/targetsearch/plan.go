package targetsearch

import (
	"context"
	"fmt"
)

// Plan runs ordered Target Search layers for one Music Service target.
type Plan[T any] struct {
	Target       any
	Service      string
	CandidateKey func(T) string
	Layers       []Layer[T]
}

// Layer is one ordered Target Search attempt inside a Plan.
type Layer[T any] struct {
	Name    string
	Enabled bool
	Search  func(context.Context) ([]T, error)
	Filter  func([]T) []T
}

type layerOutcome[T any] struct {
	candidates []T
	err        error
}

// Collect runs enabled layers, skips recoverable per-layer timeouts, and deduplicates candidates.
func (p Plan[T]) Collect(ctx context.Context) ([]T, error) {
	combined := []T{}
	seen := map[string]struct{}{}
	for _, layer := range p.Layers {
		outcome := p.runLayer(ctx, layer)
		if outcome.err != nil {
			return nil, outcome.err
		}
		combined = appendUniqueByKey(combined, seen, outcome.candidates, p.CandidateKey)
	}
	return combined, nil
}

func (p Plan[T]) runLayer(ctx context.Context, layer Layer[T]) layerOutcome[T] {
	if !layer.Enabled {
		return layerOutcome[T]{}
	}

	candidates, err := layer.Search(ctx)
	if err != nil {
		if IsRecoverableTimeout(ctx, err) {
			return layerOutcome[T]{}
		}
		return layerOutcome[T]{err: fmt.Errorf("%s %s (%T) failed: %w", layer.Name, p.Service, p.Target, err)}
	}
	if layer.Filter != nil {
		candidates = layer.Filter(candidates)
	}
	return layerOutcome[T]{candidates: candidates}
}

func appendUniqueByKey[T any](dst []T, seen map[string]struct{}, items []T, keyFunc func(T) string) []T {
	for _, item := range items {
		key := keyFunc(item)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		dst = append(dst, item)
	}
	return dst
}
