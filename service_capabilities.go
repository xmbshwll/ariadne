package ariadne

import "strings"

var serviceLookupNormalizer = strings.NewReplacer("-", "", "_", "")

var (
	serviceCapabilitiesByName = buildServiceCapabilitiesByName(defaultServiceBindings)
	serviceNamesByLookupKey   = buildServiceNamesByLookupKey(defaultServiceBindings)
	supportedTargetServices   = collectSupportedServicesInOrder(defaultServiceOrder.albumTargets, serviceCapabilitiesByName, func(capability serviceCapability) bool {
		return capability.supportsAlbumTarget || capability.supportsSongTarget
	})
	supportedSongTargetServices = collectSupportedServicesInOrder(defaultServiceOrder.songTargets, serviceCapabilitiesByName, func(capability serviceCapability) bool {
		return capability.supportsSongTarget
	})
	runtimeSongURLParsers = collectRuntimeSongURLParsers(defaultServiceBindings)
)

// LookupServiceName normalizes a service name or alias into the canonical public service name.
func LookupServiceName(raw string) (ServiceName, bool) {
	service, ok := serviceNamesByLookupKey[normalizeServiceLookupKey(raw)]
	return service, ok
}

func normalizeServiceLookupKey(raw string) string {
	return serviceLookupNormalizer.Replace(strings.ToLower(strings.TrimSpace(raw)))
}

// DescribeService reports Ariadne's built-in runtime capabilities for one service.
func DescribeService(service ServiceName) (ServiceCapabilities, bool) {
	capability, ok := serviceCapabilitiesByName[service]
	if !ok {
		return ServiceCapabilities{}, false
	}
	return capability.describe(), true
}

// SupportsSongTarget reports whether the service currently participates in built-in song target search.
func SupportsSongTarget(service ServiceName) bool {
	capability, ok := serviceCapabilitiesByName[service]
	return ok && capability.supportsSongTarget
}

// SupportsTarget reports whether the service currently participates in any built-in target search.
func SupportsTarget(service ServiceName) bool {
	capability, ok := serviceCapabilitiesByName[service]
	return ok && (capability.supportsAlbumTarget || capability.supportsSongTarget)
}

// SupportedSongTargetServices returns the canonical service names that currently support built-in song target search.
func SupportedSongTargetServices() []ServiceName {
	return append([]ServiceName(nil), supportedSongTargetServices...)
}

// SupportedTargetServices returns the canonical service names that currently support any built-in target search.
func SupportedTargetServices() []ServiceName {
	return append([]ServiceName(nil), supportedTargetServices...)
}

// SupportsRuntimeSongInputURL reports whether Ariadne can resolve the input URL through the runtime song pipeline.
func SupportsRuntimeSongInputURL(raw string) bool {
	for _, parseSongURL := range runtimeSongURLParsers {
		parsed, err := parseSongURL(raw)
		if err == nil && parsed != nil {
			return true
		}
	}
	return false
}

func buildServiceCapabilitiesByName(bindings []serviceBinding) map[ServiceName]serviceCapability {
	capabilities := make(map[ServiceName]serviceCapability, len(bindings))
	for _, binding := range bindings {
		capabilities[binding.capability.name] = binding.capability
	}
	return capabilities
}

func buildServiceNamesByLookupKey(bindings []serviceBinding) map[string]ServiceName {
	services := make(map[string]ServiceName, len(bindings)*2)
	for _, binding := range bindings {
		capability := binding.capability
		services[normalizeServiceLookupKey(string(capability.name))] = capability.name
		for _, alias := range capability.aliases {
			services[normalizeServiceLookupKey(alias)] = capability.name
		}
	}
	return services
}

func collectSupportedServicesInOrder(order []ServiceName, capabilities map[ServiceName]serviceCapability, supported func(serviceCapability) bool) []ServiceName {
	services := make([]ServiceName, 0, len(order))
	for _, service := range order {
		capability, ok := capabilities[service]
		if !ok || !supported(capability) {
			continue
		}
		services = append(services, service)
	}
	return services
}

func collectRuntimeSongURLParsers(bindings []serviceBinding) []songURLParser {
	parsers := make([]songURLParser, 0, len(bindings))
	for _, binding := range bindings {
		if binding.capability.runtimeSongURLParser == nil {
			continue
		}
		parsers = append(parsers, binding.capability.runtimeSongURLParser)
	}
	return parsers
}
