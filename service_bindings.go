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
	service             ServiceName
	aliases             []string
	capabilities        serviceCapabilitySet
	targetSearchEnabled func(Config) bool
	build               serviceAdapterBuilder
}

func (s serviceBindingSpec) capability() serviceCapability {
	return serviceCapability{
		name:                s.service,
		aliases:             append([]string(nil), s.aliases...),
		supportsAlbumSource: s.capabilities.albumSource,
		supportsAlbumTarget: s.capabilities.albumTarget,
		supportsSongSource:  s.capabilities.songSource,
		supportsSongTarget:  s.capabilities.songTarget,
		targetSearchEnabled: s.targetSearchEnabled,
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
		service:      ServiceAppleMusic,
		aliases:      []string{"applemusic"},
		capabilities: allRuntimeCapabilities,
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
		service:      ServiceBandcamp,
		aliases:      []string{"bandcamp"},
		capabilities: allRuntimeCapabilities,
		build: func(client *http.Client, _ Config) serviceAdapterSet {
			return fullRuntimeAdapterSet(bandcampadapter.New(client))
		},
	})
}

func deezerServiceBinding() serviceBinding {
	return newServiceBinding(serviceBindingSpec{
		service:      ServiceDeezer,
		aliases:      []string{"deezer"},
		capabilities: allRuntimeCapabilities,
		build: func(client *http.Client, _ Config) serviceAdapterSet {
			return fullRuntimeAdapterSet(deezeradapter.New(client))
		},
	})
}

func soundCloudServiceBinding() serviceBinding {
	return newServiceBinding(serviceBindingSpec{
		service:      ServiceSoundCloud,
		aliases:      []string{"soundcloud"},
		capabilities: allRuntimeCapabilities,
		build: func(client *http.Client, _ Config) serviceAdapterSet {
			return fullRuntimeAdapterSet(soundcloudadapter.New(client))
		},
	})
}

func spotifyServiceBinding() serviceBinding {
	return newServiceBinding(serviceBindingSpec{
		service:             ServiceSpotify,
		aliases:             []string{"spotify"},
		capabilities:        allRuntimeCapabilities,
		targetSearchEnabled: Config.SpotifyEnabled,
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
		service:             ServiceTIDAL,
		aliases:             []string{"tidal"},
		capabilities:        allRuntimeCapabilities,
		targetSearchEnabled: Config.TIDALEnabled,
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
		service:      ServiceYouTubeMusic,
		aliases:      []string{"youtubemusic", "ytmusic"},
		capabilities: albumRuntimeCapabilities,
		build: func(client *http.Client, _ Config) serviceAdapterSet {
			adapter := youtubemusicadapter.New(client)
			set := albumRuntimeAdapterSet(adapter)
			set.songSource = adapter
			return set
		},
	})
}

func amazonMusicServiceBinding() serviceBinding {
	return newServiceBinding(serviceBindingSpec{
		service:      ServiceAmazonMusic,
		aliases:      []string{"amazonmusic", "amazon"},
		capabilities: sourceOnlyCapabilities,
		build: func(*http.Client, Config) serviceAdapterSet {
			return sourceRuntimeAdapterSet(amazonmusicadapter.New(nil))
		},
	})
}
