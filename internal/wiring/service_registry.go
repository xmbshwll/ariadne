package wiring

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/xmbshwll/ariadne/internal/adapters"
	"github.com/xmbshwll/ariadne/internal/config"
	"github.com/xmbshwll/ariadne/internal/httpx"
	"github.com/xmbshwll/ariadne/internal/model"
)

func (c capabilitySpec) describe() Capabilities {
	return Capabilities{
		Aliases:                     append([]string(nil), c.aliases...),
		SupportsAlbumSource:         c.capabilities.AlbumSource,
		SupportsAlbumTarget:         c.capabilities.SupportsAlbumTarget(),
		SupportsSongSource:          c.capabilities.SongSource,
		SupportsSongTarget:          c.capabilities.SupportsSongTarget(),
		SupportsRuntimeSongInputURL: c.capabilities.SongSource,
	}
}

// enabled is the config-dependent view: a credential-gated service loses its
// Target Search Capabilities while its Source Input stays available.
func (c capabilitySpec) enabled(cfg config.Config) capabilitySpec {
	if c.targetSearchEnabled != nil && !c.targetSearchEnabled(cfg) {
		c.capabilities.AlbumUPC = false
		c.capabilities.AlbumISRC = false
		c.capabilities.AlbumMetadata = false
		c.capabilities.SongISRC = false
		c.capabilities.SongMetadata = false
	}
	return c
}

// adapterRole is one of the four roles a Music Service adapter plays in the
// default resolver wiring.
type adapterRole int

const (
	roleAlbumSource adapterRole = iota
	roleAlbumTarget
	roleSongSource
	roleSongTarget
)

// supports reports whether this Catalog entry takes part in one role.
func (c capabilitySpec) supports(role adapterRole) bool {
	switch role {
	case roleAlbumSource:
		return c.capabilities.AlbumSource
	case roleAlbumTarget:
		return c.capabilities.SupportsAlbumTarget()
	case roleSongSource:
		return c.capabilities.SongSource
	case roleSongTarget:
		return c.capabilities.SupportsSongTarget()
	default:
		return false
	}
}

// adapterBuilder builds one service's adapter under one config.
type adapterBuilder func(client *http.Client, cfg config.Config) adapters.Adapter

// capabilitySpec is one Provider Catalog entry. Capability support comes from
// the built service adapter's adapters.Capabilities, so a service states its own
// support once; the Catalog adds only aliases and credential gating.
type capabilitySpec struct {
	name                model.ServiceName
	aliases             []string
	capabilities        adapters.Capabilities
	targetSearchEnabled func(config.Config) bool
}

// capabilitySpecFor builds one Catalog entry from a binding and the adapter it
// built, so capabilities are never restated in wiring.
func capabilitySpecFor(binding bindingSpec, adapter adapters.Adapter) capabilitySpec {
	spec := capabilitySpec{
		name:                binding.service,
		aliases:             append([]string(nil), binding.aliases...),
		targetSearchEnabled: binding.targetSearchEnabled,
	}
	if adapter != nil {
		spec.capabilities = adapter.Capabilities()
	}
	return spec
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
	bindings              []bindingSpec
	order                 serviceOrder
	capabilitiesByService map[model.ServiceName]capabilitySpec
	servicesByLookupKey   map[string]model.ServiceName
	defaultAdapters       map[model.ServiceName]adapters.Adapter
}

// ResolverAdapters is the built-in adapter set the public constructors wire into
// a Resolver, split by Entity Shape and role. Every entry is the same
// adapters.Adapter interface; the role lists differ only by Capabilities.
type ResolverAdapters struct {
	AlbumSources []adapters.Adapter
	AlbumTargets []adapters.Adapter
	SongSources  []adapters.Adapter
	SongTargets  []adapters.Adapter
}

func newProviderCatalog(bindings []bindingSpec, order serviceOrder) catalog {
	catalog := catalog{
		bindings:              append([]bindingSpec(nil), bindings...),
		order:                 order.clone(),
		capabilitiesByService: make(map[model.ServiceName]capabilitySpec, len(bindings)),
		servicesByLookupKey:   make(map[string]model.ServiceName, len(bindings)*3),
	}

	// Construction is cheap struct wiring with zero config — no network, no
	// credentials — so the Catalog builds the built-in adapters once and reads
	// each service's Capabilities from them, for URL recognition and the
	// Supported* helpers without waiting for a Resolver to be built.
	catalog.defaultAdapters = buildAdapters(httpx.NewClient(0), config.Config{}, catalog.bindings)

	for _, binding := range catalog.bindings {
		service := binding.service
		if _, exists := catalog.capabilitiesByService[service]; exists {
			panic("duplicate default service binding: " + string(service))
		}
		capability := capabilitySpecFor(binding, catalog.defaultAdapters[service])
		catalog.capabilitiesByService[service] = capability
		catalog.addServiceLookup(service, string(service))
		for _, alias := range capability.aliases {
			catalog.addServiceLookup(service, alias)
		}
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
	if !ok || !c.supportsAnyTargetRole(service) {
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
	return c.targetCapabilityRequest(cfg, service, entity, message)
}

// targetRoleFor maps an Entity Shape to the Catalog role its query answers about.
func targetRoleFor(entity EntityShape) adapterRole {
	switch entity {
	case EntityShapeAlbum:
		return roleAlbumTarget
	case EntityShapeSong:
		return roleSongTarget
	default:
		return roleAlbumTarget
	}
}

// supportsTarget reports whether a Catalog entry can act as a Target for the
// requested Entity Shape; the zero Entity Shape accepts either.
func supportsTarget(capability capabilitySpec, entity EntityShape) bool {
	if entity == EntityShapeAny {
		return capability.capabilities.SupportsAnyTarget()
	}
	return capability.supports(targetRoleFor(entity))
}

func (c catalog) TargetServices(cfg *config.Config, entity EntityShape) []model.ServiceName {
	supports := func(capability capabilitySpec) bool { return supportsTarget(capability, entity) }
	order := c.order.AlbumTargets
	if entity == EntityShapeSong {
		order = c.order.SongTargets
	}
	if cfg == nil {
		return c.supportedServices(order, supports)
	}
	return c.enabledServices(*cfg, order, supports)
}

func (c catalog) targetCapabilityRequest(cfg config.Config, service model.ServiceName, entity EntityShape, message string) TargetServiceRequestDecision {
	decision := TargetServiceRequestDecision{Service: service, Message: message}
	capability, ok := c.capability(service)
	if !ok {
		decision.Status = TargetServiceRequestUnknown
		return decision
	}
	supports := func(spec capabilitySpec) bool { return supportsTarget(spec, entity) }
	if !supports(capability) {
		decision.Status = targetServiceUnavailableStatus(capability)
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

// targetServiceUnavailableStatus distinguishes a service that can only supply
// Source Input (Parse Only) from one that supports a different Target role than
// the one requested.
func targetServiceUnavailableStatus(capability capabilitySpec) TargetServiceRequestStatus {
	if !capability.capabilities.SupportsAnyTarget() {
		return TargetServiceRequestParseOnly
	}
	return TargetServiceRequestUnsupported
}

func (c catalog) supportsAnyTargetRole(service model.ServiceName) bool {
	capability, ok := c.capability(service)
	return ok && capability.capabilities.SupportsAnyTarget()
}

func (c catalog) SupportsRuntimeSongInputURL(raw string) bool {
	for _, service := range c.order.SongSources {
		source := c.defaultAdapters[service]
		if source == nil || !c.capabilitiesByService[service].capabilities.SongSource {
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
