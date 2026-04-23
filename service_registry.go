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

func builtinServiceAliases(service ServiceName) []string {
	return services.AliasesFor(toInternalServiceName(service))
}

// defaultServiceOrder preserves intentional priority differences between
// supported service lists and enabled runtime wiring. Amazon Music appears only
// in albumSources because song runtime resolution is deferred, YouTube Music is
// omitted from song lists because it is album-only today, and Spotify/TIDAL
// stay behind the public-web targets in target ordering because their official
// APIs are credential-gated in the Enabled* view.
var defaultServiceOrder = struct {
	albumSources []ServiceName
	albumTargets []ServiceName
	songSources  []ServiceName
	songTargets  []ServiceName
}{
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

func buildDefaultServiceAdapters(client *http.Client, config Config) map[ServiceName]serviceAdapterSet {
	sets := make(map[ServiceName]serviceAdapterSet, len(defaultServiceBindings))
	for _, binding := range defaultServiceBindings {
		service := binding.capability.name
		if _, exists := sets[service]; exists {
			panic("duplicate default service binding: " + string(service))
		}
		sets[service] = binding.build(client, config)
	}
	return sets
}
