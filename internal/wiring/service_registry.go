package wiring

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/xmbshwll/ariadne/internal/config"
	"github.com/xmbshwll/ariadne/internal/httpx"
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/resolve"
)

type capabilitySpec struct {
	name                        model.ServiceName
	aliases                     []string
	supportsAlbumSource         bool
	supportsAlbumTarget         bool
	supportsSongSource          bool
	supportsSongTarget          bool
	supportsRuntimeSongInputURL bool
	targetSearchEnabled         func(config.Config) bool
}

func (c capabilitySpec) describe() Capabilities {
	return Capabilities{
		Aliases:                     append([]string(nil), c.aliases...),
		SupportsAlbumSource:         c.supportsAlbumSource,
		SupportsAlbumTarget:         c.supportsAlbumTarget,
		SupportsSongSource:          c.supportsSongSource,
		SupportsSongTarget:          c.supportsSongTarget,
		SupportsRuntimeSongInputURL: c.supportsRuntimeSongInputURL,
	}
}

func (c capabilitySpec) enabled(cfg config.Config) capabilitySpec {
	if c.targetSearchEnabled != nil && !c.targetSearchEnabled(cfg) {
		c.supportsAlbumTarget = false
		c.supportsSongTarget = false
	}
	return c
}

type adapterSet struct {
	AlbumSource resolve.SourceAdapter
	AlbumTarget resolve.TargetAdapter
	SongSource  resolve.SongSourceAdapter
	SongTarget  resolve.SongTargetAdapter
}

type adapterBuilder func(client *http.Client, cfg config.Config) adapterSet

// binding describes Ariadne's built-in service support. The capability
// metadata is config-independent and feeds the Supported* helpers, while build
// applies config.Config-specific credential gating to the adapter set used by the
// Enabled* helpers and default resolver wiring.
type binding struct {
	capability capabilitySpec
	build      adapterBuilder
}

type serviceOrder struct {
	AlbumSources []model.ServiceName
	AlbumTargets []model.ServiceName
	SongSources  []model.ServiceName
	SongTargets  []model.ServiceName
}

func (o serviceOrder) clone() serviceOrder {
	return serviceOrder{
		AlbumSources: append([]model.ServiceName(nil), o.AlbumSources...),
		AlbumTargets: append([]model.ServiceName(nil), o.AlbumTargets...),
		SongSources:  append([]model.ServiceName(nil), o.SongSources...),
		SongTargets:  append([]model.ServiceName(nil), o.SongTargets...),
	}
}

// catalog is Ariadne's Provider Catalog: one Module owns built-in Music
// Service capabilities, default ordering, credential gating, runtime URL parsing,
// and service name resolution.
type catalog struct {
	bindings              []binding
	order                 serviceOrder
	capabilitiesByService map[model.ServiceName]capabilitySpec
	servicesByLookupKey   map[string]model.ServiceName
	defaultAdapterSets    map[model.ServiceName]adapterSet
}

// ResolverAdapters is the built-in adapter set the public constructors wire into
// a Resolver, split by Entity Shape and role.
type ResolverAdapters struct {
	AlbumSources []resolve.SourceAdapter
	AlbumTargets []resolve.TargetAdapter
	SongSources  []resolve.SongSourceAdapter
	SongTargets  []resolve.SongTargetAdapter
}

func newProviderCatalog(bindings []binding, order serviceOrder) catalog {
	catalog := catalog{
		bindings:              append([]binding(nil), bindings...),
		order:                 order.clone(),
		capabilitiesByService: make(map[model.ServiceName]capabilitySpec, len(bindings)),
		servicesByLookupKey:   make(map[string]model.ServiceName, len(bindings)*3),
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
	catalog.defaultAdapterSets = buildAdapterSets(httpx.NewClient(0), config.Config{}, catalog.bindings)
	for service, capability := range catalog.capabilitiesByService {
		capability.supportsRuntimeSongInputURL = catalog.defaultAdapterSets[service].SongSource != nil
		catalog.capabilitiesByService[service] = capability
	}

	catalog.validateOrder(catalog.order.AlbumSources)
	catalog.validateOrder(catalog.order.AlbumTargets)
	catalog.validateOrder(catalog.order.SongSources)
	catalog.validateOrder(catalog.order.SongTargets)

	return catalog
}

func (c catalog) validateOrder(order []model.ServiceName) {
	for _, service := range order {
		if _, ok := c.capabilitiesByService[service]; !ok {
			panic("service order references missing binding: " + string(service))
		}
	}
}

var serviceLookupKeyNormalizer = strings.NewReplacer("-", "", "_", "")

func (c catalog) addServiceLookup(service model.ServiceName, raw string) {
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

// defaultOrder preserves intentional priority differences between
// supported service lists and enabled runtime wiring. Amazon Music appears in
// AlbumSources because its album URLs parse, while runtime fetch remains
// deferred. YouTube Music and Amazon Music appear in SongSources so parse-only
// song Source Input reaches the deferred Runtime Hydration sentinel instead of
// falling through as unsupported. Spotify and TIDAL stay behind the public-web
// targets in target ordering because their official APIs are credential-gated in
// the Enabled* view.
var defaultOrder = serviceOrder{
	AlbumSources: []model.ServiceName{
		model.ServiceAppleMusic,
		model.ServiceDeezer,
		model.ServiceSpotify,
		model.ServiceTIDAL,
		model.ServiceSoundCloud,
		model.ServiceYouTubeMusic,
		model.ServiceAmazonMusic,
		model.ServiceBandcamp,
	},
	AlbumTargets: []model.ServiceName{
		model.ServiceAppleMusic,
		model.ServiceBandcamp,
		model.ServiceDeezer,
		model.ServiceSoundCloud,
		model.ServiceYouTubeMusic,
		model.ServiceSpotify,
		model.ServiceTIDAL,
	},
	SongSources: []model.ServiceName{
		model.ServiceAppleMusic,
		model.ServiceBandcamp,
		model.ServiceDeezer,
		model.ServiceSoundCloud,
		model.ServiceSpotify,
		model.ServiceTIDAL,
		model.ServiceYouTubeMusic,
		model.ServiceAmazonMusic,
	},
	SongTargets: []model.ServiceName{
		model.ServiceAppleMusic,
		model.ServiceBandcamp,
		model.ServiceDeezer,
		model.ServiceSoundCloud,
		model.ServiceSpotify,
		model.ServiceTIDAL,
	},
}

// Default is the built-in Provider Catalog: Ariadne's Music Services, their
// capabilities, and the order Target Search and the Supported* lists follow.
var Default = newProviderCatalog(defaultBindings, defaultOrder)

func (c catalog) LookupServiceName(raw string) (model.ServiceName, bool) {
	service, ok := c.servicesByLookupKey[normalizeServiceLookupKey(raw)]
	return service, ok
}

func (c catalog) LookupSupportedTargetService(raw string) (model.ServiceName, bool) {
	service, ok := c.LookupServiceName(raw)
	if !ok || !c.supportsTarget(service) {
		return "", false
	}
	return service, true
}

func (c catalog) capability(service model.ServiceName) (capabilitySpec, bool) {
	capability, ok := c.capabilitiesByService[service]
	return capability, ok
}

func (c catalog) enabledServiceCapability(cfg config.Config, service model.ServiceName) (capabilitySpec, bool) {
	capability, ok := c.capability(service)
	if !ok {
		return capabilitySpec{}, false
	}
	return capability.enabled(cfg), true
}

func (c catalog) DescribeService(service model.ServiceName) (Capabilities, bool) {
	capability, ok := c.capability(service)
	if !ok {
		return Capabilities{}, false
	}
	return capability.describe(), true
}

func (c catalog) DescribeEnabledService(cfg config.Config, service model.ServiceName) (Capabilities, bool) {
	capability, ok := c.enabledServiceCapability(cfg, service)
	if !ok {
		return Capabilities{}, false
	}
	return capability.describe(), true
}

func (c catalog) EvaluateTarget(cfg config.Config, name string, entity EntityShape) TargetServiceRequestDecision {
	message := c.unavailableTargetServiceMessage(name, cfg, entity)
	service, ok := c.LookupServiceName(name)
	if !ok {
		return TargetServiceRequestDecision{Status: TargetServiceRequestUnknown, Message: message}
	}
	return c.targetCapabilityRequest(cfg, service, targetCapabilityFor(entity), message)
}

func targetCapabilityFor(entity EntityShape) func(capabilitySpec) bool {
	switch entity {
	case EntityShapeAlbum:
		return func(capability capabilitySpec) bool { return capability.supportsAlbumTarget }
	case EntityShapeSong:
		return supportsSongTargetCapability
	default:
		return supportsAnyTarget
	}
}

func (c catalog) TargetServices(cfg *config.Config, entity EntityShape) []model.ServiceName {
	supports := targetCapabilityFor(entity)
	order := c.order.AlbumTargets
	if entity == EntityShapeSong {
		order = c.order.SongTargets
	}
	if cfg == nil {
		return c.supportedServices(order, supports)
	}
	return c.enabledServices(*cfg, order, supports)
}

func (c catalog) targetCapabilityRequest(cfg config.Config, service model.ServiceName, supports func(capabilitySpec) bool, message string) TargetServiceRequestDecision {
	decision := TargetServiceRequestDecision{Service: service, Message: message}
	capability, ok := c.capability(service)
	if !ok {
		decision.Status = TargetServiceRequestUnknown
		return decision
	}
	if !supports(capability) {
		decision.Status = targetServiceUnavailableStatus(service, capability)
		return decision
	}
	if !supports(capability.enabled(cfg)) {
		decision.Status = TargetServiceRequestCredentialsRequired
		decision.CredentialHint = credentialHints[service]
		return decision
	}
	decision.Status = TargetServiceRequestAvailable
	decision.Message = ""
	return decision
}

func (c catalog) unavailableTargetServiceMessage(raw string, cfg config.Config, entity EntityShape) string {
	if entity == EntityShapeSong {
		names := serviceNameStrings(c.TargetServices(&cfg, entity))
		if len(names) == 0 {
			return fmt.Sprintf("%q (enabled for songs: none)", raw)
		}
		return fmt.Sprintf("%q (enabled for songs: %s)", raw, strings.Join(names, ", "))
	}
	return fmt.Sprintf("%q (expected one of the supported target services: %s)", raw, strings.Join(serviceNameStrings(c.TargetServices(nil, entity)), ", "))
}

func serviceNameStrings(services []model.ServiceName) []string {
	names := make([]string, 0, len(services))
	for _, service := range services {
		names = append(names, string(service))
	}
	return names
}

func targetServiceUnavailableStatus(service model.ServiceName, capability capabilitySpec) TargetServiceRequestStatus {
	if service == model.ServiceAmazonMusic && !supportsAnyTarget(capability) {
		return TargetServiceRequestParseOnly
	}
	return TargetServiceRequestUnsupported
}

func (c catalog) supportsTarget(service model.ServiceName) bool {
	capability, ok := c.capability(service)
	return ok && supportsAnyTarget(capability)
}

func (c catalog) SupportsRuntimeSongInputURL(raw string) bool {
	for _, service := range c.order.SongSources {
		source := c.defaultAdapterSets[service].SongSource
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

func (c catalog) supportedServices(order []model.ServiceName, supported func(capabilitySpec) bool) []model.ServiceName {
	services := make([]model.ServiceName, 0, len(order))
	for _, service := range order {
		capability, ok := c.capability(service)
		if !ok || !supported(capability) {
			continue
		}
		services = append(services, service)
	}
	return services
}

func (c catalog) enabledServices(cfg config.Config, order []model.ServiceName, supported func(capabilitySpec) bool) []model.ServiceName {
	services := make([]model.ServiceName, 0, len(order))
	for _, service := range order {
		capability, ok := c.enabledServiceCapability(cfg, service)
		if !ok || !supported(capability) {
			continue
		}
		services = append(services, service)
	}
	return services
}

func supportsSongTargetCapability(capability capabilitySpec) bool {
	return capability.supportsSongTarget
}

func supportsAnyTarget(capability capabilitySpec) bool {
	return capability.supportsAlbumTarget || capability.supportsSongTarget
}
