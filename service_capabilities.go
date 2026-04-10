package ariadne

import (
	"slices"
	"strings"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/parse"
)

type songURLParser func(string) (*model.ParsedAlbumURL, error)

var serviceLookupNormalizer = strings.NewReplacer("-", "", "_", "")

type serviceCapability struct {
	name                 ServiceName
	aliases              []string
	supportsAlbumSource  bool
	supportsAlbumTarget  bool
	supportsSongSource   bool
	supportsSongTarget   bool
	runtimeSongURLParser songURLParser
}

var serviceCapabilities = []serviceCapability{
	{
		name:                 ServiceAppleMusic,
		aliases:              []string{"applemusic"},
		supportsAlbumSource:  true,
		supportsAlbumTarget:  true,
		supportsSongSource:   true,
		supportsSongTarget:   true,
		runtimeSongURLParser: parse.AppleMusicSongURL,
	},
	{
		name:                 ServiceBandcamp,
		aliases:              []string{"bandcamp"},
		supportsAlbumSource:  true,
		supportsAlbumTarget:  true,
		supportsSongSource:   true,
		supportsSongTarget:   true,
		runtimeSongURLParser: parse.BandcampSongURL,
	},
	{
		name:                 ServiceDeezer,
		aliases:              []string{"deezer"},
		supportsAlbumSource:  true,
		supportsAlbumTarget:  true,
		supportsSongSource:   true,
		supportsSongTarget:   true,
		runtimeSongURLParser: parse.DeezerSongURL,
	},
	{
		name:                 ServiceSoundCloud,
		aliases:              []string{"soundcloud"},
		supportsAlbumSource:  true,
		supportsAlbumTarget:  true,
		supportsSongSource:   true,
		supportsSongTarget:   true,
		runtimeSongURLParser: parse.SoundCloudSongURL,
	},
	{
		name:                 ServiceSpotify,
		aliases:              []string{"spotify"},
		supportsAlbumSource:  true,
		supportsAlbumTarget:  true,
		supportsSongSource:   true,
		supportsSongTarget:   true,
		runtimeSongURLParser: parse.SpotifySongURL,
	},
	{
		name:                 ServiceTIDAL,
		aliases:              []string{"tidal"},
		supportsAlbumSource:  true,
		supportsAlbumTarget:  true,
		supportsSongSource:   true,
		supportsSongTarget:   true,
		runtimeSongURLParser: parse.TIDALSongURL,
	},
	{
		name:                ServiceYouTubeMusic,
		aliases:             []string{"youtubemusic", "ytmusic"},
		supportsAlbumSource: true,
		supportsAlbumTarget: true,
	},
	{
		name:    ServiceAmazonMusic,
		aliases: []string{"amazonmusic", "amazon"},
	},
}

func serviceCapabilityByName(service ServiceName) (serviceCapability, bool) {
	for _, capability := range serviceCapabilities {
		if capability.name == service {
			return capability, true
		}
	}
	return serviceCapability{}, false
}

// LookupServiceName normalizes a service name or alias into the canonical public service name.
func LookupServiceName(raw string) (ServiceName, bool) {
	normalized := normalizeServiceLookupKey(raw)
	if normalized == "" {
		return "", false
	}

	for _, capability := range serviceCapabilities {
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
	for _, capability := range serviceCapabilities {
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
	for _, capability := range serviceCapabilities {
		if !SupportsTarget(capability.name) {
			continue
		}
		services = append(services, capability.name)
	}
	return services
}

func registeredRuntimeSongURLParsers() []songURLParser {
	parsers := []songURLParser{}
	for _, capability := range serviceCapabilities {
		if capability.runtimeSongURLParser == nil {
			continue
		}
		parsers = append(parsers, capability.runtimeSongURLParser)
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
