package ariadne

import (
	"net/http"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/resolve"
)

func defaultSourceAdapters(client *http.Client, config Config) []resolve.SourceAdapter {
	return orderedAdapters(
		buildDefaultServiceAdapters(client, config),
		defaultServiceOrder.albumSources,
		func(set serviceAdapterSet) resolve.SourceAdapter { return set.albumSource },
	)
}

func defaultTargetAdapters(client *http.Client, config Config) []resolve.TargetAdapter {
	targets := orderedAdapters(
		buildDefaultServiceAdapters(client, config),
		defaultServiceOrder.albumTargets,
		func(set serviceAdapterSet) resolve.TargetAdapter { return set.albumTarget },
	)
	return filterAdaptersByServiceName(targets, config.TargetServices)
}

func defaultSongSourceAdapters(client *http.Client, config Config) []resolve.SongSourceAdapter {
	return orderedAdapters(
		buildDefaultServiceAdapters(client, config),
		defaultServiceOrder.songSources,
		func(set serviceAdapterSet) resolve.SongSourceAdapter { return set.songSource },
	)
}

func defaultSongTargetAdapters(client *http.Client, config Config) []resolve.SongTargetAdapter {
	targets := orderedAdapters(
		buildDefaultServiceAdapters(client, config),
		defaultServiceOrder.songTargets,
		func(set serviceAdapterSet) resolve.SongTargetAdapter { return set.songTarget },
	)
	return filterAdaptersByServiceName(targets, config.TargetServices)
}

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
		if _, ok := allowed[fromInternalServiceName(adapter.Service())]; !ok {
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

func wrapSourceAdapters(sources []SourceAdapter) []resolve.SourceAdapter {
	wrapped := make([]resolve.SourceAdapter, 0, len(sources))
	for _, source := range sources {
		wrapped = append(wrapped, sourceAdapterBridge{source: source})
	}
	return wrapped
}

func wrapSongSourceAdapters(sources []SongSourceAdapter) []resolve.SongSourceAdapter {
	wrapped := make([]resolve.SongSourceAdapter, 0, len(sources))
	for _, source := range sources {
		wrapped = append(wrapped, songSourceAdapterBridge{source: source})
	}
	return wrapped
}

func wrapTargetAdapters(targets []TargetAdapter) []resolve.TargetAdapter {
	wrapped := make([]resolve.TargetAdapter, 0, len(targets))
	for _, target := range targets {
		wrapped = append(wrapped, targetAdapterBridge{target: target})
	}
	return wrapped
}

func wrapSongTargetAdapters(targets []SongTargetAdapter) []resolve.SongTargetAdapter {
	wrapped := make([]resolve.SongTargetAdapter, 0, len(targets))
	for _, target := range targets {
		wrapped = append(wrapped, songTargetAdapterBridge{target: target})
	}
	return wrapped
}
