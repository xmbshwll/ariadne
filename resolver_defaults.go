package ariadne

import (
	"net/http"

	"github.com/xmbshwll/ariadne/internal/resolve"
)

func buildResolverAdapters(client *http.Client, config Config, bindings []serviceBinding, order serviceOrder, targetServices []ServiceName) providerResolverAdapters {
	sets := buildServiceAdapterSets(client, config, bindings)
	return providerResolverAdapters{
		albumSources: orderedAdapters(sets, order.albumSources, func(set serviceAdapterSet) resolve.SourceAdapter {
			return set.albumSource
		}),
		albumTargets: filterAdaptersByServiceName(
			orderedAdapters(sets, order.albumTargets, func(set serviceAdapterSet) resolve.TargetAdapter {
				return set.albumTarget
			}),
			targetServices,
		),
		songSources: orderedAdapters(sets, order.songSources, func(set serviceAdapterSet) resolve.SongSourceAdapter {
			return set.songSource
		}),
		songTargets: filterAdaptersByServiceName(
			orderedAdapters(sets, order.songTargets, func(set serviceAdapterSet) resolve.SongTargetAdapter {
				return set.songTarget
			}),
			targetServices,
		),
	}
}

func buildServiceAdapterSets(client *http.Client, config Config, bindings []serviceBinding) map[ServiceName]serviceAdapterSet {
	sets := make(map[ServiceName]serviceAdapterSet, len(bindings))
	for _, binding := range bindings {
		service := binding.capability.name
		sets[service] = binding.build(client, config)
	}
	return sets
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

func filterAdaptersByServiceName[T interface{ Service() ServiceName }](adapters []T, services []ServiceName) []T {
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
