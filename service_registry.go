package ariadne

import (
	"net/http"

	amazonmusicadapter "github.com/xmbshwll/ariadne/internal/adapters/amazonmusic"
	applemusicadapter "github.com/xmbshwll/ariadne/internal/adapters/applemusic"
	bandcampadapter "github.com/xmbshwll/ariadne/internal/adapters/bandcamp"
	deezeradapter "github.com/xmbshwll/ariadne/internal/adapters/deezer"
	soundcloudadapter "github.com/xmbshwll/ariadne/internal/adapters/soundcloud"
	spotifyadapter "github.com/xmbshwll/ariadne/internal/adapters/spotify"
	tidaladapter "github.com/xmbshwll/ariadne/internal/adapters/tidal"
	youtubemusicadapter "github.com/xmbshwll/ariadne/internal/adapters/youtubemusic"
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/parse"
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

var defaultServiceBindings = []serviceBinding{
	{
		capability: serviceCapability{
			name:                 ServiceAppleMusic,
			aliases:              builtinServiceAliases(ServiceAppleMusic),
			supportsAlbumSource:  true,
			supportsAlbumTarget:  true,
			supportsSongSource:   true,
			supportsSongTarget:   true,
			runtimeSongURLParser: parse.AppleMusicSongURL,
		},
		build: func(client *http.Client, config Config) serviceAdapterSet {
			adapter := applemusicadapter.New(
				client,
				applemusicadapter.WithDefaultStorefront(config.AppleMusicStorefront),
				applemusicadapter.WithDeveloperTokenAuth(
					config.AppleMusic.KeyID,
					config.AppleMusic.TeamID,
					config.AppleMusic.PrivateKeyPath,
				),
			)
			return serviceAdapterSet{
				albumSource: adapter,
				albumTarget: adapter,
				songSource:  adapter,
				songTarget:  adapter,
			}
		},
	},
	{
		capability: serviceCapability{
			name:                 ServiceBandcamp,
			aliases:              builtinServiceAliases(ServiceBandcamp),
			supportsAlbumSource:  true,
			supportsAlbumTarget:  true,
			supportsSongSource:   true,
			supportsSongTarget:   true,
			runtimeSongURLParser: parse.BandcampSongURL,
		},
		build: func(client *http.Client, _ Config) serviceAdapterSet {
			adapter := bandcampadapter.New(client)
			return serviceAdapterSet{albumSource: adapter, albumTarget: adapter, songSource: adapter, songTarget: adapter}
		},
	},
	{
		capability: serviceCapability{
			name:                 ServiceDeezer,
			aliases:              builtinServiceAliases(ServiceDeezer),
			supportsAlbumSource:  true,
			supportsAlbumTarget:  true,
			supportsSongSource:   true,
			supportsSongTarget:   true,
			runtimeSongURLParser: parse.DeezerSongURL,
		},
		build: func(client *http.Client, _ Config) serviceAdapterSet {
			adapter := deezeradapter.New(client)
			return serviceAdapterSet{albumSource: adapter, albumTarget: adapter, songSource: adapter, songTarget: adapter}
		},
	},
	{
		capability: serviceCapability{
			name:                 ServiceSoundCloud,
			aliases:              builtinServiceAliases(ServiceSoundCloud),
			supportsAlbumSource:  true,
			supportsAlbumTarget:  true,
			supportsSongSource:   true,
			supportsSongTarget:   true,
			runtimeSongURLParser: parse.SoundCloudSongURL,
		},
		build: func(client *http.Client, _ Config) serviceAdapterSet {
			adapter := soundcloudadapter.New(client)
			return serviceAdapterSet{albumSource: adapter, albumTarget: adapter, songSource: adapter, songTarget: adapter}
		},
	},
	{
		capability: serviceCapability{
			name:                 ServiceSpotify,
			aliases:              builtinServiceAliases(ServiceSpotify),
			supportsAlbumSource:  true,
			supportsAlbumTarget:  true,
			supportsSongSource:   true,
			supportsSongTarget:   true,
			runtimeSongURLParser: parse.SpotifySongURL,
		},
		build: func(client *http.Client, config Config) serviceAdapterSet {
			adapter := spotifyadapter.New(
				client,
				spotifyadapter.WithCredentials(config.Spotify.ClientID, config.Spotify.ClientSecret),
			)
			set := serviceAdapterSet{albumSource: adapter, songSource: adapter}
			if config.SpotifyEnabled() {
				set.albumTarget = adapter
				set.songTarget = adapter
			}
			return set
		},
	},
	{
		capability: serviceCapability{
			name:                 ServiceTIDAL,
			aliases:              builtinServiceAliases(ServiceTIDAL),
			supportsAlbumSource:  true,
			supportsAlbumTarget:  true,
			supportsSongSource:   true,
			supportsSongTarget:   true,
			runtimeSongURLParser: parse.TIDALSongURL,
		},
		build: func(client *http.Client, config Config) serviceAdapterSet {
			adapter := tidaladapter.New(
				client,
				tidaladapter.WithCredentials(config.TIDAL.ClientID, config.TIDAL.ClientSecret),
			)
			set := serviceAdapterSet{albumSource: adapter, songSource: adapter}
			if config.TIDALEnabled() {
				set.albumTarget = adapter
				set.songTarget = adapter
			}
			return set
		},
	},
	{
		capability: serviceCapability{
			name:                ServiceYouTubeMusic,
			aliases:             builtinServiceAliases(ServiceYouTubeMusic),
			supportsAlbumSource: true,
			supportsAlbumTarget: true,
		},
		build: func(client *http.Client, _ Config) serviceAdapterSet {
			adapter := youtubemusicadapter.New(client)
			return serviceAdapterSet{albumSource: adapter, albumTarget: adapter}
		},
	},
	{
		capability: serviceCapability{
			name:                ServiceAmazonMusic,
			aliases:             builtinServiceAliases(ServiceAmazonMusic),
			supportsAlbumSource: true,
		},
		build: func(client *http.Client, _ Config) serviceAdapterSet {
			adapter := amazonmusicadapter.New(client)
			return serviceAdapterSet{albumSource: adapter}
		},
	},
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
