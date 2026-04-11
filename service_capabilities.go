package ariadne

import (
	"slices"
	"strings"
)

var serviceLookupNormalizer = strings.NewReplacer("-", "", "_", "")

func serviceCapabilityByName(service ServiceName) (serviceCapability, bool) {
	binding, ok := serviceBindingByName(service)
	if !ok {
		return serviceCapability{}, false
	}
	return binding.capability, true
}

// LookupServiceName normalizes a service name or alias into the canonical public service name.
func LookupServiceName(raw string) (ServiceName, bool) {
	normalized := normalizeServiceLookupKey(raw)
	if normalized == "" {
		return "", false
	}

	for _, binding := range defaultServiceBindings {
		capability := binding.capability
		if normalized == normalizeServiceLookupKey(string(capability.name)) || slices.Contains(capability.aliases, normalized) {
			return capability.name, true
		}
	}
	return "", false
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
	return ok && (capability.supportsAlbumTarget || capability.supportsSongTarget)
}

// SupportedSongTargetServices returns the canonical service names that currently support runtime song target search.
func SupportedSongTargetServices() []ServiceName {
	services := []ServiceName{}
	for _, binding := range defaultServiceBindings {
		capability := binding.capability
		if !capability.supportsSongTarget {
			continue
		}
		services = append(services, capability.name)
	}
	return services
}

// SupportedTargetServices returns the canonical service names that currently support runtime target search.
func SupportedTargetServices() []ServiceName {
	services := []ServiceName{}
	for _, binding := range defaultServiceBindings {
		capability := binding.capability
		if !SupportsTarget(capability.name) {
			continue
		}
		services = append(services, capability.name)
	}
	return services
}

func registeredRuntimeSongURLParsers() []songURLParser {
	parsers := []songURLParser{}
	for _, binding := range defaultServiceBindings {
		if binding.capability.runtimeSongURLParser == nil {
			continue
		}
		parsers = append(parsers, binding.capability.runtimeSongURLParser)
	}
	return parsers
}

// SupportsRuntimeSongInputURL reports whether Ariadne can resolve the input URL through the runtime song pipeline.
func SupportsRuntimeSongInputURL(raw string) bool {
	for _, parseSongURL := range registeredRuntimeSongURLParsers() {
		parsed, err := parseSongURL(raw)
		if err == nil && parsed != nil {
			return true
		}
	}
	return false
}
