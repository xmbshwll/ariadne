package ariadne

import (
	"net/http"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/resolve"
	"github.com/xmbshwll/ariadne/internal/services"
)

type songURLParser func(string) (*model.ParsedURL, error)

type serviceCapability struct {
	name                 ServiceName
	aliases              []string
	supportsAlbumSource  bool
	supportsAlbumTarget  bool
	supportsSongSource   bool
	supportsSongTarget   bool
	runtimeSongURLParser songURLParser
}

func (c serviceCapability) describe() ServiceCapabilities {
	return ServiceCapabilities{
		Aliases:                     append([]string(nil), c.aliases...),
		SupportsAlbumSource:         c.supportsAlbumSource,
		SupportsAlbumTarget:         c.supportsAlbumTarget,
		SupportsSongSource:          c.supportsSongSource,
		SupportsSongTarget:          c.supportsSongTarget,
		SupportsRuntimeSongInputURL: c.runtimeSongURLParser != nil,
	}
}

func (c serviceCapability) enabled(config Config) serviceCapability {
	config = normalizedConfig(config)
	switch c.name {
	case ServiceSpotify:
		if !config.SpotifyEnabled() {
			c.supportsAlbumTarget = false
			c.supportsSongTarget = false
		}
	case ServiceTIDAL:
		if !config.TIDALEnabled() {
			c.supportsAlbumTarget = false
			c.supportsSongTarget = false
		}
	}
	return c
}

type serviceAdapterSet struct {
	albumSource resolve.SourceAdapter
	albumTarget resolve.TargetAdapter
	songSource  resolve.SongSourceAdapter
	songTarget  resolve.SongTargetAdapter
}

// serviceBinding describes Ariadne's built-in service support. The capability
// metadata is config-independent and feeds the Supported* helpers, while build
// applies Config-specific credential gating to the adapter set used by the
// Enabled* helpers and default resolver wiring.
type serviceBinding struct {
	capability serviceCapability
	build      func(client *http.Client, config Config) serviceAdapterSet
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
// and Adapter construction.
type providerCatalog struct {
	bindings              []serviceBinding
	order                 serviceOrder
	capabilitiesByService map[ServiceName]serviceCapability
	runtimeSongURLParsers []songURLParser
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
		runtimeSongURLParsers: make([]songURLParser, 0, len(bindings)),
	}

	for _, binding := range catalog.bindings {
		service := binding.capability.name
		if _, exists := catalog.capabilitiesByService[service]; exists {
			panic("duplicate default service binding: " + string(service))
		}
		catalog.capabilitiesByService[service] = binding.capability
		if binding.capability.runtimeSongURLParser != nil {
			catalog.runtimeSongURLParsers = append(catalog.runtimeSongURLParsers, binding.capability.runtimeSongURLParser)
		}
	}
	return catalog
}

func builtinServiceAliases(service ServiceName) []string {
	return services.AliasesFor(toInternalServiceName(service))
}

// defaultServiceOrder preserves intentional priority differences between
// supported service lists and enabled runtime wiring. Amazon Music appears in
// albumSources and songSources because its URLs parse in both pipelines, while
// runtime fetch remains deferred. YouTube Music appears in album sources,
// album targets, and song sources; song target search is still omitted. Spotify
// and TIDAL stay behind the public-web targets in target ordering because their
// official APIs are credential-gated in the Enabled* view.
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

func (c providerCatalog) supportsSongTarget(service ServiceName) bool {
	capability, ok := c.serviceCapability(service)
	return ok && capability.supportsSongTarget
}

func (c providerCatalog) supportsEnabledSongTarget(config Config, service ServiceName) bool {
	capability, ok := c.enabledServiceCapability(config, service)
	return ok && capability.supportsSongTarget
}

func (c providerCatalog) supportsTarget(service ServiceName) bool {
	capability, ok := c.serviceCapability(service)
	return ok && supportsAnyTarget(capability)
}

func (c providerCatalog) supportsEnabledTarget(config Config, service ServiceName) bool {
	capability, ok := c.enabledServiceCapability(config, service)
	return ok && supportsAnyTarget(capability)
}

func (c providerCatalog) supportedTargetServices() []ServiceName {
	return c.supportedServices(c.order.albumTargets, supportsAnyTarget)
}

func (c providerCatalog) enabledTargetServices(config Config) []ServiceName {
	return c.enabledServices(config, c.order.albumTargets, supportsAnyTarget)
}

func (c providerCatalog) supportedSongTargetServices() []ServiceName {
	return c.supportedServices(c.order.songTargets, func(capability serviceCapability) bool {
		return capability.supportsSongTarget
	})
}

func (c providerCatalog) enabledSongTargetServices(config Config) []ServiceName {
	return c.enabledServices(config, c.order.songTargets, func(capability serviceCapability) bool {
		return capability.supportsSongTarget
	})
}

func (c providerCatalog) supportsRuntimeSongInputURL(raw string) bool {
	for _, parseSongURL := range c.runtimeSongURLParsers {
		parsed, err := parseSongURL(raw)
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

func (c providerCatalog) resolverAdapters(client *http.Client, config Config) providerResolverAdapters {
	sets := c.buildAdapters(client, config)
	return providerResolverAdapters{
		albumSources: c.albumSourceAdapters(sets),
		albumTargets: c.albumTargetAdapters(sets, config.TargetServices),
		songSources:  c.songSourceAdapters(sets),
		songTargets:  c.songTargetAdapters(sets, config.TargetServices),
	}
}

func (c providerCatalog) buildAdapters(client *http.Client, config Config) map[ServiceName]serviceAdapterSet {
	sets := make(map[ServiceName]serviceAdapterSet, len(c.bindings))
	for _, binding := range c.bindings {
		service := binding.capability.name
		sets[service] = binding.build(client, config)
	}
	return sets
}

func (c providerCatalog) albumSourceAdapters(sets map[ServiceName]serviceAdapterSet) []resolve.SourceAdapter {
	return orderedAdapters(
		sets,
		c.order.albumSources,
		func(set serviceAdapterSet) resolve.SourceAdapter { return set.albumSource },
	)
}

func (c providerCatalog) albumTargetAdapters(sets map[ServiceName]serviceAdapterSet, services []ServiceName) []resolve.TargetAdapter {
	targets := orderedAdapters(
		sets,
		c.order.albumTargets,
		func(set serviceAdapterSet) resolve.TargetAdapter { return set.albumTarget },
	)
	return filterAdaptersByServiceName(targets, services)
}

func (c providerCatalog) songSourceAdapters(sets map[ServiceName]serviceAdapterSet) []resolve.SongSourceAdapter {
	return orderedAdapters(
		sets,
		c.order.songSources,
		func(set serviceAdapterSet) resolve.SongSourceAdapter { return set.songSource },
	)
}

func (c providerCatalog) songTargetAdapters(sets map[ServiceName]serviceAdapterSet, services []ServiceName) []resolve.SongTargetAdapter {
	targets := orderedAdapters(
		sets,
		c.order.songTargets,
		func(set serviceAdapterSet) resolve.SongTargetAdapter { return set.songTarget },
	)
	return filterAdaptersByServiceName(targets, services)
}

func supportsAnyTarget(capability serviceCapability) bool {
	return capability.supportsAlbumTarget || capability.supportsSongTarget
}
