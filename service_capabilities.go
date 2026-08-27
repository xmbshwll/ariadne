// Provider Catalog query surface: Describe reports built-in and config-enabled
// Music Service support, EvaluateTarget validates a Target Service Request, and
// TargetServices lists matching services. The entity shape (album, song, or any)
// is a parameter, not a function name. All of it delegates to internal/wiring.
package ariadne

import "github.com/xmbshwll/ariadne/internal/wiring"

// EntityShape selects the entity shape a Provider Catalog query applies to.
// The zero value queries any target Capability.
type EntityShape = wiring.EntityShape

// Re-export entity shape constants so callers don't need to import the wiring module.
const (
	// EntityShapeAny queries services with any target Capability.
	EntityShapeAny = wiring.EntityShapeAny
	// EntityShapeAlbum queries the album target Capability.
	EntityShapeAlbum = wiring.EntityShapeAlbum
	// EntityShapeSong queries the song target Capability.
	EntityShapeSong = wiring.EntityShapeSong
)

// LookupServiceName normalizes a service name or alias into the canonical public service name.
func LookupServiceName(raw string) (ServiceName, bool) {
	return wiring.Default.LookupServiceName(raw)
}

// Describe reports Ariadne's built-in service support, independent of config.
func Describe(service ServiceName) (ServiceCapabilities, bool) {
	return wiring.Default.DescribeService(service)
}

// DescribeEnabled reports the service support currently enabled under config.
func DescribeEnabled(config Config, service ServiceName) (ServiceCapabilities, bool) {
	return wiring.Default.DescribeEnabledService(internalConfig(config), service)
}

// EvaluateTarget reports whether the requested service can join Target Search
// under config for the given entity shape. The name may be an alias or a
// canonical service name; the decision explains unknown names, missing
// capabilities, parse-only constraints, and required credentials.
func EvaluateTarget(config Config, name string, entity EntityShape) TargetServiceRequestDecision {
	return wiring.Default.EvaluateTarget(internalConfig(config), name, entity)
}

// TargetServices returns the canonical service names with the requested target
// Capability in Provider Catalog order. A nil config lists built-in supported
// services; a non-nil config lists the services enabled under it.
func TargetServices(config *Config, entity EntityShape) []ServiceName {
	if config == nil {
		return wiring.Default.TargetServices(nil, entity)
	}
	normalized := internalConfig(*config)
	return wiring.Default.TargetServices(&normalized, entity)
}

// SupportsRuntimeSongInputURL reports whether Ariadne can resolve the input URL through the runtime song pipeline.
func SupportsRuntimeSongInputURL(raw string) bool {
	return wiring.Default.SupportsRuntimeSongInputURL(raw)
}
