package ariadne

import (
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/resolve"
)

func orderedAdapters[T comparable](sets map[ServiceName]serviceAdapterSet, services []ServiceName, pick func(serviceAdapterSet) T) []T {
	adapters := make([]T, 0, len(services))
	var zero T
	for _, service := range services {
		adapter := pick(sets[service])
		if adapter == zero {
			continue
		}
		adapters = append(adapters, adapter)
	}
	return adapters
}

func filterAdaptersByServiceName[T interface{ Service() model.ServiceName }](adapters []T, services []ServiceName) []T {
	allowed := serviceNameSet(services)
	if len(allowed) == 0 {
		return adapters
	}

	filtered := make([]T, 0, len(adapters))
	for _, adapter := range adapters {
		if _, ok := allowed[adapter.Service()]; !ok {
			continue
		}
		filtered = append(filtered, adapter)
	}
	return filtered
}

func serviceNameSet(services []ServiceName) map[ServiceName]struct{} {
	if len(services) == 0 {
		return nil
	}

	allowed := make(map[ServiceName]struct{}, len(services))
	for _, service := range services {
		allowed[service] = struct{}{}
	}
	return allowed
}

func resolveSourceAdapters(sources []SourceAdapter) []resolve.SourceAdapter {
	adapters := make([]resolve.SourceAdapter, 0, len(sources))
	for _, source := range sources {
		adapters = append(adapters, source)
	}
	return adapters
}

func resolveSongSourceAdapters(sources []SongSourceAdapter) []resolve.SongSourceAdapter {
	adapters := make([]resolve.SongSourceAdapter, 0, len(sources))
	for _, source := range sources {
		adapters = append(adapters, source)
	}
	return adapters
}

func resolveTargetAdapters(targets []TargetAdapter) []resolve.TargetAdapter {
	adapters := make([]resolve.TargetAdapter, 0, len(targets))
	for _, target := range targets {
		adapters = append(adapters, target)
	}
	return adapters
}

func resolveSongTargetAdapters(targets []SongTargetAdapter) []resolve.SongTargetAdapter {
	adapters := make([]resolve.SongTargetAdapter, 0, len(targets))
	for _, target := range targets {
		adapters = append(adapters, target)
	}
	return adapters
}
