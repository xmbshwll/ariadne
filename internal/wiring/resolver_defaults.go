package wiring

import (
	"net/http"

	"github.com/xmbshwll/ariadne/internal/adapters"
	"github.com/xmbshwll/ariadne/internal/config"
	"github.com/xmbshwll/ariadne/internal/model"
)

// buildResolverAdapters builds every built-in adapter once, then distributes the
// adapters into the four Resolver roles by their declared Capabilities under
// cfg. Credential gating happens here rather than inside a builder: a gated
// service keeps its Source Input and loses its Target Search roles.
func buildResolverAdapters(client *http.Client, cfg config.Config, bindings []binding, order serviceOrder, targetServices []model.ServiceName) ResolverAdapters {
	built := buildAdapters(client, cfg, bindings)
	enabled := enabledCapabilities(cfg, bindings)
	return ResolverAdapters{
		AlbumSources: orderedAdapters(built, enabled, order.AlbumSources, roleAlbumSource),
		AlbumTargets: filterAdaptersByServiceName(orderedAdapters(built, enabled, order.AlbumTargets, roleAlbumTarget), targetServices),
		SongSources:  orderedAdapters(built, enabled, order.SongSources, roleSongSource),
		SongTargets:  filterAdaptersByServiceName(orderedAdapters(built, enabled, order.SongTargets, roleSongTarget), targetServices),
	}
}

func buildAdapters(client *http.Client, cfg config.Config, bindings []binding) map[model.ServiceName]adapters.Adapter {
	built := make(map[model.ServiceName]adapters.Adapter, len(bindings))
	for _, binding := range bindings {
		built[binding.capability.name] = binding.build(client, cfg)
	}
	return built
}

func enabledCapabilities(cfg config.Config, bindings []binding) map[model.ServiceName]capabilitySpec {
	capabilities := make(map[model.ServiceName]capabilitySpec, len(bindings))
	for _, binding := range bindings {
		capabilities[binding.capability.name] = binding.capability.enabled(cfg)
	}
	return capabilities
}

// orderedAdapters returns the built adapters for one role in Catalog order.
func orderedAdapters(
	built map[model.ServiceName]adapters.Adapter,
	capabilities map[model.ServiceName]capabilitySpec,
	services []model.ServiceName,
	role adapterRole,
) []adapters.Adapter {
	ordered := make([]adapters.Adapter, 0, len(services))
	for _, service := range services {
		capability, ok := capabilities[service]
		if !ok || !capability.supports(role) {
			continue
		}
		if adapter := built[service]; adapter != nil {
			ordered = append(ordered, adapter)
		}
	}
	return ordered
}

// filterAdaptersByServiceName limits a Target list to an explicit --services
// selection, keeping Catalog order. An empty selection means every Target.
func filterAdaptersByServiceName(candidates []adapters.Adapter, services []model.ServiceName) []adapters.Adapter {
	allowed := serviceNameSet(services)
	if len(allowed) == 0 {
		return candidates
	}

	filtered := make([]adapters.Adapter, 0, len(candidates))
	for _, adapter := range candidates {
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
