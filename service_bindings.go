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
	"github.com/xmbshwll/ariadne/internal/resolve"
)

var defaultServiceBindings = []serviceBinding{
	appleMusicServiceBinding(),
	bandcampServiceBinding(),
	deezerServiceBinding(),
	soundCloudServiceBinding(),
	spotifyServiceBinding(),
	tidalServiceBinding(),
	youTubeMusicServiceBinding(),
	amazonMusicServiceBinding(),
}

type serviceCapabilitySet struct {
	albumSource bool
	albumTarget bool
	songSource  bool
	songTarget  bool
}

var (
	allRuntimeCapabilities   = serviceCapabilitySet{albumSource: true, albumTarget: true, songSource: true, songTarget: true}
	albumRuntimeCapabilities = serviceCapabilitySet{albumSource: true, albumTarget: true}
	sourceOnlyCapabilities   = serviceCapabilitySet{albumSource: true, songSource: true}
)

type fullRuntimeAdapter interface {
	resolve.SourceAdapter
	resolve.TargetAdapter
	resolve.SongSourceAdapter
	resolve.SongTargetAdapter
}

type albumRuntimeAdapter interface {
	resolve.SourceAdapter
	resolve.TargetAdapter
}

type sourceRuntimeAdapter interface {
	resolve.SourceAdapter
	resolve.SongSourceAdapter
}

type serviceBindingSpec struct {
	service              ServiceName
	capabilities         serviceCapabilitySet
	runtimeSongURLParser songURLParser
	targetSearchEnabled  func(Config) bool
	build                serviceAdapterBuilder
}

func (s serviceBindingSpec) capability() serviceCapability {
	return serviceCapability{
		name:                 s.service,
		aliases:              builtinServiceAliases(s.service),
		supportsAlbumSource:  s.capabilities.albumSource,
		supportsAlbumTarget:  s.capabilities.albumTarget,
		supportsSongSource:   s.capabilities.songSource,
		supportsSongTarget:   s.capabilities.songTarget,
		runtimeSongURLParser: s.runtimeSongURLParser,
		targetSearchEnabled:  s.targetSearchEnabled,
	}
}

func newServiceBinding(spec serviceBindingSpec) serviceBinding {
	return serviceBinding{
		capability: spec.capability(),
		build:      spec.build,
	}
}

func fullRuntimeAdapterSet(adapter fullRuntimeAdapter) serviceAdapterSet {
	return serviceAdapterSet{
		albumSource: adapter,
		albumTarget: adapter,
		songSource:  adapter,
		songTarget:  adapter,
	}
}

func albumRuntimeAdapterSet(adapter albumRuntimeAdapter) serviceAdapterSet {
	return serviceAdapterSet{albumSource: adapter, albumTarget: adapter}
}

func sourceRuntimeAdapterSet(adapter sourceRuntimeAdapter) serviceAdapterSet {
	return serviceAdapterSet{albumSource: adapter, songSource: adapter}
}

func credentialedFullRuntimeAdapterSet(adapter fullRuntimeAdapter, targetSearchEnabled bool) serviceAdapterSet {
	set := sourceRuntimeAdapterSet(adapter)
	if targetSearchEnabled {
		set.albumTarget = adapter
		set.songTarget = adapter
	}
	return set
}

func appleMusicServiceBinding() serviceBinding {
	return newServiceBinding(serviceBindingSpec{
		service:              ServiceAppleMusic,
		capabilities:         allRuntimeCapabilities,
		runtimeSongURLParser: applemusicadapter.ParseSongURL,
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
			return fullRuntimeAdapterSet(adapter)
		},
	})
}

func bandcampServiceBinding() serviceBinding {
	return newServiceBinding(serviceBindingSpec{
		service:              ServiceBandcamp,
		capabilities:         allRuntimeCapabilities,
		runtimeSongURLParser: bandcampadapter.ParseSongURL,
		build: func(client *http.Client, _ Config) serviceAdapterSet {
			return fullRuntimeAdapterSet(bandcampadapter.New(client))
		},
	})
}

func deezerServiceBinding() serviceBinding {
	return newServiceBinding(serviceBindingSpec{
		service:              ServiceDeezer,
		capabilities:         allRuntimeCapabilities,
		runtimeSongURLParser: deezeradapter.ParseSongURL,
		build: func(client *http.Client, _ Config) serviceAdapterSet {
			return fullRuntimeAdapterSet(deezeradapter.New(client))
		},
	})
}

func soundCloudServiceBinding() serviceBinding {
	return newServiceBinding(serviceBindingSpec{
		service:              ServiceSoundCloud,
		capabilities:         allRuntimeCapabilities,
		runtimeSongURLParser: soundcloudadapter.ParseSongURL,
		build: func(client *http.Client, _ Config) serviceAdapterSet {
			return fullRuntimeAdapterSet(soundcloudadapter.New(client))
		},
	})
}

func spotifyServiceBinding() serviceBinding {
	return newServiceBinding(serviceBindingSpec{
		service:              ServiceSpotify,
		capabilities:         allRuntimeCapabilities,
		runtimeSongURLParser: spotifyadapter.ParseSongURL,
		targetSearchEnabled:  Config.SpotifyEnabled,
		build: func(client *http.Client, config Config) serviceAdapterSet {
			adapter := spotifyadapter.New(
				client,
				spotifyadapter.WithCredentials(config.Spotify.ClientID, config.Spotify.ClientSecret),
			)
			return credentialedFullRuntimeAdapterSet(adapter, config.SpotifyEnabled())
		},
	})
}

func tidalServiceBinding() serviceBinding {
	return newServiceBinding(serviceBindingSpec{
		service:              ServiceTIDAL,
		capabilities:         allRuntimeCapabilities,
		runtimeSongURLParser: tidaladapter.ParseSongURL,
		targetSearchEnabled:  Config.TIDALEnabled,
		build: func(client *http.Client, config Config) serviceAdapterSet {
			adapter := tidaladapter.New(
				client,
				tidaladapter.WithCredentials(config.TIDAL.ClientID, config.TIDAL.ClientSecret),
			)
			return credentialedFullRuntimeAdapterSet(adapter, config.TIDALEnabled())
		},
	})
}

func youTubeMusicServiceBinding() serviceBinding {
	return newServiceBinding(serviceBindingSpec{
		service:              ServiceYouTubeMusic,
		capabilities:         albumRuntimeCapabilities,
		runtimeSongURLParser: youtubemusicadapter.ParseSongURL,
		build: func(client *http.Client, _ Config) serviceAdapterSet {
			return albumRuntimeAdapterSet(youtubemusicadapter.New(client))
		},
	})
}

func amazonMusicServiceBinding() serviceBinding {
	return newServiceBinding(serviceBindingSpec{
		service:              ServiceAmazonMusic,
		capabilities:         sourceOnlyCapabilities,
		runtimeSongURLParser: amazonmusicadapter.ParseSongURL,
		build: func(*http.Client, Config) serviceAdapterSet {
			return sourceRuntimeAdapterSet(amazonmusicadapter.New(nil))
		},
	})
}
