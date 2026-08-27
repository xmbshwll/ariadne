package wiring

import (
	"net/http"

	"github.com/xmbshwll/ariadne/internal/config"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/resolve"
)

func buildResolverAdapters(client *http.Client, cfg config.Config, bindings []binding, order serviceOrder, targetServices []model.ServiceName) ResolverAdapters {
	sets := buildAdapterSets(client, cfg, bindings)
	return ResolverAdapters{
		AlbumSources: orderedAdapters(sets, order.AlbumSources, func(set adapterSet) resolve.SourceAdapter {
			return set.AlbumSource
		}),
		AlbumTargets: filterAdaptersByServiceName(
			orderedAdapters(sets, order.AlbumTargets, func(set adapterSet) resolve.TargetAdapter {
				return set.AlbumTarget
			}),
			targetServices,
		),
		SongSources: orderedAdapters(sets, order.SongSources, func(set adapterSet) resolve.SongSourceAdapter {
			return set.SongSource
		}),
		SongTargets: filterAdaptersByServiceName(
			orderedAdapters(sets, order.SongTargets, func(set adapterSet) resolve.SongTargetAdapter {
				return set.SongTarget
			}),
			targetServices,
		),
	}
}

func buildAdapterSets(client *http.Client, cfg config.Config, bindings []binding) map[model.ServiceName]adapterSet {
	sets := make(map[model.ServiceName]adapterSet, len(bindings))
	for _, binding := range bindings {
		service := binding.capability.name
		sets[service] = binding.build(client, cfg)
	}
	return sets
}

func orderedAdapters[T comparable](sets map[model.ServiceName]adapterSet, services []model.ServiceName, pick func(adapterSet) T) []T {
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

func filterAdaptersByServiceName[T interface{ Service() model.ServiceName }](adapters []T, services []model.ServiceName) []T {
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

func serviceNameSet(services []model.ServiceName) map[model.ServiceName]struct{} {
	if len(services) == 0 {
		return nil
	}

	allowed := make(map[model.ServiceName]struct{}, len(services))
	for _, service := range services {
		allowed[service] = struct{}{}
	}
	return allowed
}
