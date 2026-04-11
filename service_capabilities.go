package ariadne

import "strings"

var serviceLookupNormalizer = strings.NewReplacer("-", "", "_", "")

var (
	serviceCapabilitiesByName     = buildServiceCapabilitiesByName(defaultServiceBindings)
	serviceNamesByLookupKey       = buildServiceNamesByLookupKey(defaultServiceBindings)
	supportedTargetServiceSet     = collectSupportedServices(defaultServiceBindings, supportsAnyTarget)
	supportedSongTargetServiceSet = collectSupportedServices(defaultServiceBindings, supportsSongTargetOnly)
	runtimeSongURLParsers         = collectRuntimeSongURLParsers(defaultServiceBindings)
)

func serviceCapabilityByName(service ServiceName) (serviceCapability, bool) {
	capability, ok := serviceCapabilitiesByName[service]
	return capability, ok
}

// LookupServiceName normalizes a service name or alias into the canonical public service name.
func LookupServiceName(raw string) (ServiceName, bool) {
	service, ok := serviceNamesByLookupKey[normalizeServiceLookupKey(raw)]
	return service, ok
}

func normalizeServiceLookupKey(raw string) string {
	return serviceLookupNormalizer.Replace(strings.ToLower(strings.TrimSpace(raw)))
}

// SupportsSongTarget reports whether the service currently participates in runtime song target search.
func SupportsSongTarget(service ServiceName) bool {
	capability, ok := serviceCapabilityByName(service)
	return ok && capability.supportsSongTarget
}

// SupportsTarget reports whether the service currently participates in any runtime target search.
func SupportsTarget(service ServiceName) bool {
	capability, ok := serviceCapabilityByName(service)
	return ok && supportsAnyTarget(capability)
}

// SupportedSongTargetServices returns the canonical service names that currently support runtime song target search.
func SupportedSongTargetServices() []ServiceName {
	return append([]ServiceName(nil), supportedSongTargetServiceSet...)
}

// SupportedTargetServices returns the canonical service names that currently support runtime target search.
func SupportedTargetServices() []ServiceName {
	return append([]ServiceName(nil), supportedTargetServiceSet...)
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

func collectSupportedServices(bindings []serviceBinding, supported func(serviceCapability) bool) []ServiceName {
	services := make([]ServiceName, 0, len(bindings))
	for _, binding := range bindings {
		if !supported(binding.capability) {
			continue
		}
		services = append(services, binding.capability.name)
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

func supportsAnyTarget(capability serviceCapability) bool {
	return capability.supportsAlbumTarget || capability.supportsSongTarget
}

func supportsSongTargetOnly(capability serviceCapability) bool {
	return capability.supportsSongTarget
}
