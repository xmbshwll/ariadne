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

type serviceRoles struct {
	albumSource bool
	albumTarget bool
	songSource  bool
	songTarget  bool
}

var (
	allRuntimeRoles   = serviceRoles{albumSource: true, albumTarget: true, songSource: true, songTarget: true}
	albumRuntimeRoles = serviceRoles{albumSource: true, albumTarget: true}
	parseOnlyRoles    = serviceRoles{}
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

func serviceBindingFor(service ServiceName, roles serviceRoles, parser songURLParser, build serviceAdapterBuilder) serviceBinding {
	return serviceBinding{
		capability: serviceCapabilityFor(service, roles, parser),
		build:      build,
	}
}

func credentialedTargetServiceBinding(service ServiceName, roles serviceRoles, parser songURLParser, targetSearchEnabled func(Config) bool, build serviceAdapterBuilder) serviceBinding {
	binding := serviceBindingFor(service, roles, parser, build)
	binding.capability = binding.capability.withTargetSearchEnabled(targetSearchEnabled)
	return binding
}

func serviceCapabilityFor(service ServiceName, roles serviceRoles, parser songURLParser) serviceCapability {
	return serviceCapability{
		name:                 service,
		aliases:              builtinServiceAliases(service),
		supportsAlbumSource:  roles.albumSource,
		supportsAlbumTarget:  roles.albumTarget,
		supportsSongSource:   roles.songSource,
		supportsSongTarget:   roles.songTarget,
		runtimeSongURLParser: parser,
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
	return serviceBindingFor(ServiceAppleMusic, allRuntimeRoles, applemusicadapter.ParseSongURL, func(client *http.Client, config Config) serviceAdapterSet {
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
	})
}

func bandcampServiceBinding() serviceBinding {
	return serviceBindingFor(ServiceBandcamp, allRuntimeRoles, bandcampadapter.ParseSongURL, func(client *http.Client, _ Config) serviceAdapterSet {
		return fullRuntimeAdapterSet(bandcampadapter.New(client))
	})
}

func deezerServiceBinding() serviceBinding {
	return serviceBindingFor(ServiceDeezer, allRuntimeRoles, deezeradapter.ParseSongURL, func(client *http.Client, _ Config) serviceAdapterSet {
		return fullRuntimeAdapterSet(deezeradapter.New(client))
	})
}

func soundCloudServiceBinding() serviceBinding {
	return serviceBindingFor(ServiceSoundCloud, allRuntimeRoles, soundcloudadapter.ParseSongURL, func(client *http.Client, _ Config) serviceAdapterSet {
		return fullRuntimeAdapterSet(soundcloudadapter.New(client))
	})
}

func spotifyServiceBinding() serviceBinding {
	return credentialedTargetServiceBinding(ServiceSpotify, allRuntimeRoles, spotifyadapter.ParseSongURL, Config.SpotifyEnabled, func(client *http.Client, config Config) serviceAdapterSet {
		adapter := spotifyadapter.New(
			client,
			spotifyadapter.WithCredentials(config.Spotify.ClientID, config.Spotify.ClientSecret),
		)
		return credentialedFullRuntimeAdapterSet(adapter, config.SpotifyEnabled())
	})
}

func tidalServiceBinding() serviceBinding {
	return credentialedTargetServiceBinding(ServiceTIDAL, allRuntimeRoles, tidaladapter.ParseSongURL, Config.TIDALEnabled, func(client *http.Client, config Config) serviceAdapterSet {
		adapter := tidaladapter.New(
			client,
			tidaladapter.WithCredentials(config.TIDAL.ClientID, config.TIDAL.ClientSecret),
		)
		return credentialedFullRuntimeAdapterSet(adapter, config.TIDALEnabled())
	})
}

func youTubeMusicServiceBinding() serviceBinding {
	return serviceBindingFor(ServiceYouTubeMusic, albumRuntimeRoles, youtubemusicadapter.ParseSongURL, func(client *http.Client, _ Config) serviceAdapterSet {
		return albumRuntimeAdapterSet(youtubemusicadapter.New(client))
	})
}

func amazonMusicServiceBinding() serviceBinding {
	return serviceBindingFor(ServiceAmazonMusic, parseOnlyRoles, amazonmusicadapter.ParseSongURL, func(*http.Client, Config) serviceAdapterSet {
		return serviceAdapterSet{}
	})
}
