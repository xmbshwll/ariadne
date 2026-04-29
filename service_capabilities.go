// Package ariadne exposes two service-capability views: Supported* helpers report
// built-in, config-independent support baked into Ariadne, while Enabled* helpers
// report the services actually active for a specific Config after credential
// gating is applied.
package ariadne

import "github.com/xmbshwll/ariadne/internal/services"

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
	return defaultProviderCatalog.describeService(service)
}

// DescribeEnabledService reports the service support currently enabled under config.
func DescribeEnabledService(config Config, service ServiceName) (ServiceCapabilities, bool) {
	return defaultProviderCatalog.describeEnabledService(config, service)
}

// SupportsSongTarget reports whether the service has built-in song target support.
func SupportsSongTarget(service ServiceName) bool {
	return defaultProviderCatalog.supportsSongTarget(service)
}

// SupportsEnabledSongTarget reports whether the service is enabled for song target search under config.
func SupportsEnabledSongTarget(config Config, service ServiceName) bool {
	return defaultProviderCatalog.supportsEnabledSongTarget(config, service)
}

// SupportsTarget reports whether the service has any built-in target support.
func SupportsTarget(service ServiceName) bool {
	return defaultProviderCatalog.supportsTarget(service)
}

// SupportsEnabledTarget reports whether the service is enabled for any target search under config.
func SupportsEnabledTarget(config Config, service ServiceName) bool {
	return defaultProviderCatalog.supportsEnabledTarget(config, service)
}

// SupportedSongTargetServices returns the canonical service names with built-in song target support.
func SupportedSongTargetServices() []ServiceName {
	return defaultProviderCatalog.supportedSongTargetServices()
}

// EnabledSongTargetServices returns the canonical service names enabled for runtime song target search under config.
func EnabledSongTargetServices(config Config) []ServiceName {
	return defaultProviderCatalog.enabledSongTargetServices(config)
}

// SupportedTargetServices returns the canonical service names with any built-in target support.
func SupportedTargetServices() []ServiceName {
	return defaultProviderCatalog.supportedTargetServices()
}

// EnabledTargetServices returns the canonical service names enabled for runtime target search under config.
func EnabledTargetServices(config Config) []ServiceName {
	return defaultProviderCatalog.enabledTargetServices(config)
}

// SupportsRuntimeSongInputURL reports whether Ariadne can resolve the input URL through the runtime song pipeline.
func SupportsRuntimeSongInputURL(raw string) bool {
	return defaultProviderCatalog.supportsRuntimeSongInputURL(raw)
}
