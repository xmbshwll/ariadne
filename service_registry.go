package ariadne

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/xmbshwll/ariadne/internal/httpx"
	"github.com/xmbshwll/ariadne/internal/resolve"
)

type serviceCapability struct {
	name                        ServiceName
	aliases                     []string
	supportsAlbumSource         bool
	supportsAlbumTarget         bool
	supportsSongSource          bool
	supportsSongTarget          bool
	supportsRuntimeSongInputURL bool
	targetSearchEnabled         func(Config) bool
}

func (c serviceCapability) describe() ServiceCapabilities {
	return ServiceCapabilities{
		Aliases:                     append([]string(nil), c.aliases...),
		SupportsAlbumSource:         c.supportsAlbumSource,
		SupportsAlbumTarget:         c.supportsAlbumTarget,
		SupportsSongSource:          c.supportsSongSource,
		SupportsSongTarget:          c.supportsSongTarget,
		SupportsRuntimeSongInputURL: c.supportsRuntimeSongInputURL,
	}
}

func (c serviceCapability) enabled(config Config) serviceCapability {
	config = normalizedConfig(config)
	if c.targetSearchEnabled != nil && !c.targetSearchEnabled(config) {
		c.supportsAlbumTarget = false
		c.supportsSongTarget = false
	}
	return c
}

type serviceAdapterSet struct {
	albumSource resolve.SourceAdapter
	albumTarget resolve.TargetAdapter
	songSource  resolve.SongSourceAdapter
	songTarget  resolve.SongTargetAdapter
}

type serviceAdapterBuilder func(client *http.Client, config Config) serviceAdapterSet

// serviceBinding describes Ariadne's built-in service support. The capability
// metadata is config-independent and feeds the Supported* helpers, while build
// applies Config-specific credential gating to the adapter set used by the
// Enabled* helpers and default resolver wiring.
type serviceBinding struct {
	capability serviceCapability
	build      serviceAdapterBuilder
}

type serviceOrder struct {
	albumSources []ServiceName
	albumTargets []ServiceName
	songSources  []ServiceName
	songTargets  []ServiceName
}

func (o serviceOrder) clone() serviceOrder {
	return serviceOrder{
		albumSources: append([]ServiceName(nil), o.albumSources...),
		albumTargets: append([]ServiceName(nil), o.albumTargets...),
		songSources:  append([]ServiceName(nil), o.songSources...),
		songTargets:  append([]ServiceName(nil), o.songTargets...),
	}
}

// providerCatalog is Ariadne's Provider Catalog: one Module owns built-in Music
// Service capabilities, default ordering, credential gating, runtime URL parsing,
// and service name resolution.
type providerCatalog struct {
	bindings              []serviceBinding
	order                 serviceOrder
	capabilitiesByService map[ServiceName]serviceCapability
	servicesByLookupKey   map[string]ServiceName
	defaultAdapterSets    map[ServiceName]serviceAdapterSet
}

type providerResolverAdapters struct {
	albumSources []resolve.SourceAdapter
	albumTargets []resolve.TargetAdapter
	songSources  []resolve.SongSourceAdapter
	songTargets  []resolve.SongTargetAdapter
}

func newProviderCatalog(bindings []serviceBinding, order serviceOrder) providerCatalog {
	catalog := providerCatalog{
		bindings:              append([]serviceBinding(nil), bindings...),
		order:                 order.clone(),
		capabilitiesByService: make(map[ServiceName]serviceCapability, len(bindings)),
		servicesByLookupKey:   make(map[string]ServiceName, len(bindings)*3),
	}

	for _, binding := range catalog.bindings {
		service := binding.capability.name
		if _, exists := catalog.capabilitiesByService[service]; exists {
			panic("duplicate default service binding: " + string(service))
		}
		capability := binding.capability
		catalog.capabilitiesByService[service] = capability
		catalog.addServiceLookup(service, string(service))
		for _, alias := range capability.aliases {
			catalog.addServiceLookup(service, alias)
		}
	}

	// Derive song Source Input recognition from the real adapter sets instead
	// of a parallel parser list: the built song source adapter is the seam the
	// runtime song pipeline actually uses. Construction is cheap struct wiring
	// with zero config — no network, no credentials.
	catalog.defaultAdapterSets = buildServiceAdapterSets(httpx.NewClient(0), Config{}, catalog.bindings)
	for service, capability := range catalog.capabilitiesByService {
		capability.supportsRuntimeSongInputURL = catalog.defaultAdapterSets[service].songSource != nil
		catalog.capabilitiesByService[service] = capability
	}

	catalog.validateOrder(catalog.order.albumSources)
	catalog.validateOrder(catalog.order.albumTargets)
	catalog.validateOrder(catalog.order.songSources)
	catalog.validateOrder(catalog.order.songTargets)

	return catalog
}

func (c providerCatalog) validateOrder(order []ServiceName) {
	for _, service := range order {
		if _, ok := c.capabilitiesByService[service]; !ok {
			panic("service order references missing binding: " + string(service))
		}
	}
}

var serviceLookupKeyNormalizer = strings.NewReplacer("-", "", "_", "")

func (c providerCatalog) addServiceLookup(service ServiceName, raw string) {
	key := normalizeServiceLookupKey(raw)
	if key == "" {
		return
	}
	if existing, exists := c.servicesByLookupKey[key]; exists && existing != service {
		panic(fmt.Sprintf("duplicate service lookup key %q for %s and %s", key, existing, service))
	}
	c.servicesByLookupKey[key] = service
}

func normalizeServiceLookupKey(raw string) string {
	return serviceLookupKeyNormalizer.Replace(strings.ToLower(strings.TrimSpace(raw)))
}

// defaultServiceOrder preserves intentional priority differences between
// supported service lists and enabled runtime wiring. Amazon Music appears in
// albumSources because its album URLs parse, while runtime fetch remains
// deferred. YouTube Music and Amazon Music appear in songSources so parse-only
// song Source Input reaches the deferred Runtime Hydration sentinel instead of
// falling through as unsupported. Spotify and TIDAL stay behind the public-web
// targets in target ordering because their official APIs are credential-gated in
// the Enabled* view.
var defaultServiceOrder = serviceOrder{
	albumSources: []ServiceName{
		ServiceAppleMusic,
		ServiceDeezer,
		ServiceSpotify,
		ServiceTIDAL,
		ServiceSoundCloud,
		ServiceYouTubeMusic,
		ServiceAmazonMusic,
		ServiceBandcamp,
	},
	albumTargets: []ServiceName{
		ServiceAppleMusic,
		ServiceBandcamp,
		ServiceDeezer,
		ServiceSoundCloud,
		ServiceYouTubeMusic,
		ServiceSpotify,
		ServiceTIDAL,
	},
	songSources: []ServiceName{
		ServiceAppleMusic,
		ServiceBandcamp,
		ServiceDeezer,
		ServiceSoundCloud,
		ServiceSpotify,
		ServiceTIDAL,
		ServiceYouTubeMusic,
		ServiceAmazonMusic,
	},
	songTargets: []ServiceName{
		ServiceAppleMusic,
		ServiceBandcamp,
		ServiceDeezer,
		ServiceSoundCloud,
		ServiceSpotify,
		ServiceTIDAL,
	},
}

var defaultProviderCatalog = newProviderCatalog(defaultServiceBindings, defaultServiceOrder)

func (c providerCatalog) lookupServiceName(raw string) (ServiceName, bool) {
	service, ok := c.servicesByLookupKey[normalizeServiceLookupKey(raw)]
	return service, ok
}

func (c providerCatalog) lookupSupportedTargetService(raw string) (ServiceName, bool) {
	service, ok := c.lookupServiceName(raw)
	if !ok || !c.supportsTarget(service) {
		return "", false
	}
	return service, true
}

func (c providerCatalog) serviceCapability(service ServiceName) (serviceCapability, bool) {
	capability, ok := c.capabilitiesByService[service]
	return capability, ok
}

func (c providerCatalog) enabledServiceCapability(config Config, service ServiceName) (serviceCapability, bool) {
	capability, ok := c.serviceCapability(service)
	if !ok {
		return serviceCapability{}, false
	}
	return capability.enabled(config), true
}

func (c providerCatalog) describeService(service ServiceName) (ServiceCapabilities, bool) {
	capability, ok := c.serviceCapability(service)
	if !ok {
		return ServiceCapabilities{}, false
	}
	return capability.describe(), true
}

func (c providerCatalog) describeEnabledService(config Config, service ServiceName) (ServiceCapabilities, bool) {
	capability, ok := c.enabledServiceCapability(config, service)
	if !ok {
		return ServiceCapabilities{}, false
	}
	return capability.describe(), true
}

func (c providerCatalog) evaluateTarget(config Config, name string, entity EntityShape) TargetServiceRequestDecision {
	message := c.unavailableTargetServiceMessage(name, config, entity)
	service, ok := c.lookupServiceName(name)
	if !ok {
		return TargetServiceRequestDecision{Status: TargetServiceRequestUnknown, Message: message}
	}
	return c.targetCapabilityRequest(config, service, targetCapabilityFor(entity), message)
}

func targetCapabilityFor(entity EntityShape) func(serviceCapability) bool {
	switch entity {
	case EntityShapeAlbum:
		return func(capability serviceCapability) bool { return capability.supportsAlbumTarget }
	case EntityShapeSong:
		return supportsSongTargetCapability
	default:
		return supportsAnyTarget
	}
}

func (c providerCatalog) targetServices(config *Config, entity EntityShape) []ServiceName {
	supports := targetCapabilityFor(entity)
	order := c.order.albumTargets
	if entity == EntityShapeSong {
		order = c.order.songTargets
	}
	if config == nil {
		return c.supportedServices(order, supports)
	}
	return c.enabledServices(*config, order, supports)
}

func (c providerCatalog) targetCapabilityRequest(config Config, service ServiceName, supports func(serviceCapability) bool, message string) TargetServiceRequestDecision {
	decision := TargetServiceRequestDecision{Service: service, Message: message}
	capability, ok := c.serviceCapability(service)
	if !ok {
		decision.Status = TargetServiceRequestUnknown
		return decision
	}
	if !supports(capability) {
		decision.Status = targetServiceUnavailableStatus(service, capability)
		return decision
	}
	if !supports(capability.enabled(config)) {
		decision.Status = TargetServiceRequestCredentialsRequired
		return decision
	}
	decision.Status = TargetServiceRequestAvailable
	decision.Message = ""
	return decision
}

func (c providerCatalog) unavailableTargetServiceMessage(raw string, config Config, entity EntityShape) string {
	if entity == EntityShapeSong {
		names := serviceNameStrings(c.targetServices(&config, entity))
		if len(names) == 0 {
			return fmt.Sprintf("%q (enabled for songs: none)", raw)
		}
		return fmt.Sprintf("%q (enabled for songs: %s)", raw, strings.Join(names, ", "))
	}
	return fmt.Sprintf("%q (expected one of the supported target services: %s)", raw, strings.Join(serviceNameStrings(c.targetServices(nil, entity)), ", "))
}

func serviceNameStrings(services []ServiceName) []string {
	names := make([]string, 0, len(services))
	for _, service := range services {
		names = append(names, string(service))
	}
	return names
}

func targetServiceUnavailableStatus(service ServiceName, capability serviceCapability) TargetServiceRequestStatus {
	if service == ServiceAmazonMusic && !supportsAnyTarget(capability) {
		return TargetServiceRequestParseOnly
	}
	return TargetServiceRequestUnsupported
}

func (c providerCatalog) supportsTarget(service ServiceName) bool {
	capability, ok := c.serviceCapability(service)
	return ok && supportsAnyTarget(capability)
}

func (c providerCatalog) supportsRuntimeSongInputURL(raw string) bool {
	for _, service := range c.order.songSources {
		source := c.defaultAdapterSets[service].songSource
		if source == nil {
			continue
		}
		parsed, err := source.ParseSongURL(raw)
		if err == nil && parsed != nil {
			return true
		}
	}
	return false
}

func (c providerCatalog) supportedServices(order []ServiceName, supported func(serviceCapability) bool) []ServiceName {
	services := make([]ServiceName, 0, len(order))
	for _, service := range order {
		capability, ok := c.serviceCapability(service)
		if !ok || !supported(capability) {
			continue
		}
		services = append(services, service)
	}
	return services
}

func (c providerCatalog) enabledServices(config Config, order []ServiceName, supported func(serviceCapability) bool) []ServiceName {
	services := make([]ServiceName, 0, len(order))
	for _, service := range order {
		capability, ok := c.enabledServiceCapability(config, service)
		if !ok || !supported(capability) {
			continue
		}
		services = append(services, service)
	}
	return services
}

func supportsSongTargetCapability(capability serviceCapability) bool {
	return capability.supportsSongTarget
}

func supportsAnyTarget(capability serviceCapability) bool {
	return capability.supportsAlbumTarget || capability.supportsSongTarget
}
