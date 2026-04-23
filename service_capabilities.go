// Package ariadne exposes two service-capability views: Supported* helpers report
// built-in, config-independent support baked into Ariadne, while Enabled* helpers
// report the services actually active for a specific Config after credential
// gating is applied.
package ariadne

import "github.com/xmbshwll/ariadne/internal/services"

var (
	supportedCapabilitiesByService = buildServiceCapabilitiesByName(defaultServiceBindings)
	supportedTargetServices        = collectSupportedServicesInOrder(defaultServiceOrder.albumTargets, supportedCapabilitiesByService, supportsAnyTarget)
	supportedSongTargetServices    = collectSupportedServicesInOrder(defaultServiceOrder.songTargets, supportedCapabilitiesByService, func(capability serviceCapability) bool {
		return capability.supportsSongTarget
	})
	supportedRuntimeSongURLParsers = collectRuntimeSongURLParsers(defaultServiceBindings)
)

// LookupServiceName normalizes a service name or alias into the canonical public service name.
func LookupServiceName(raw string) (ServiceName, bool) {
	service, ok := services.Lookup(raw)
	if !ok {
		return "", false
	}
	return fromInternalServiceName(service), true
}

// DescribeService reports Ariadne's built-in service support, independent of config.
func DescribeService(service ServiceName) (ServiceCapabilities, bool) {
	capability, ok := supportedServiceCapability(service)
	if !ok {
		return ServiceCapabilities{}, false
	}
	return capability.describe(), true
}

// DescribeEnabledService reports the service support currently enabled under config.
func DescribeEnabledService(config Config, service ServiceName) (ServiceCapabilities, bool) {
	capability, ok := enabledServiceCapability(config, service)
	if !ok {
		return ServiceCapabilities{}, false
	}
	return capability.describe(), true
}

// SupportsSongTarget reports whether the service has built-in song target support.
func SupportsSongTarget(service ServiceName) bool {
	capability, ok := supportedServiceCapability(service)
	return ok && capability.supportsSongTarget
}

// SupportsEnabledSongTarget reports whether the service is enabled for song target search under config.
func SupportsEnabledSongTarget(config Config, service ServiceName) bool {
	capability, ok := enabledServiceCapability(config, service)
	return ok && capability.supportsSongTarget
}

// SupportsTarget reports whether the service has any built-in target support.
func SupportsTarget(service ServiceName) bool {
	capability, ok := supportedServiceCapability(service)
	return ok && supportsAnyTarget(capability)
}

// SupportsEnabledTarget reports whether the service is enabled for any target search under config.
func SupportsEnabledTarget(config Config, service ServiceName) bool {
	capability, ok := enabledServiceCapability(config, service)
	return ok && supportsAnyTarget(capability)
}

// SupportedSongTargetServices returns the canonical service names with built-in song target support.
func SupportedSongTargetServices() []ServiceName {
	return append([]ServiceName(nil), supportedSongTargetServices...)
}

// EnabledSongTargetServices returns the canonical service names enabled for runtime song target search under config.
func EnabledSongTargetServices(config Config) []ServiceName {
	return collectEnabledServicesInOrder(config, defaultServiceOrder.songTargets, func(capability serviceCapability) bool {
		return capability.supportsSongTarget
	})
}

// SupportedTargetServices returns the canonical service names with any built-in target support.
func SupportedTargetServices() []ServiceName {
	return append([]ServiceName(nil), supportedTargetServices...)
}

// EnabledTargetServices returns the canonical service names enabled for runtime target search under config.
func EnabledTargetServices(config Config) []ServiceName {
	return collectEnabledServicesInOrder(config, defaultServiceOrder.albumTargets, supportsAnyTarget)
}

// SupportsRuntimeSongInputURL reports whether Ariadne can resolve the input URL through the runtime song pipeline.
func SupportsRuntimeSongInputURL(raw string) bool {
	for _, parseSongURL := range supportedRuntimeSongURLParsers {
		parsed, err := parseSongURL(raw)
		if err == nil && parsed != nil {
			return true
		}
	}
	return false
}

func supportedServiceCapability(service ServiceName) (serviceCapability, bool) {
	capability, ok := supportedCapabilitiesByService[service]
	return capability, ok
}

func enabledServiceCapability(config Config, service ServiceName) (serviceCapability, bool) {
	capability, ok := supportedServiceCapability(service)
	if !ok {
		return serviceCapability{}, false
	}

	config = normalizedConfig(config)
	switch service {
	case ServiceSpotify:
		if !config.SpotifyEnabled() {
			capability.supportsAlbumTarget = false
			capability.supportsSongTarget = false
		}
	case ServiceTIDAL:
		if !config.TIDALEnabled() {
			capability.supportsAlbumTarget = false
			capability.supportsSongTarget = false
		}
	}
	return capability, true
}

func supportsAnyTarget(capability serviceCapability) bool {
	return capability.supportsAlbumTarget || capability.supportsSongTarget
}

func buildServiceCapabilitiesByName(bindings []serviceBinding) map[ServiceName]serviceCapability {
	capabilities := make(map[ServiceName]serviceCapability, len(bindings))
	for _, binding := range bindings {
		capabilities[binding.capability.name] = binding.capability
	}
	return capabilities
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

func collectEnabledServicesInOrder(config Config, order []ServiceName, supported func(serviceCapability) bool) []ServiceName {
	services := make([]ServiceName, 0, len(order))
	for _, service := range order {
		capability, ok := enabledServiceCapability(config, service)
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
