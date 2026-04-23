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
	"github.com/xmbshwll/ariadne/internal/parse"
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

func appleMusicServiceBinding() serviceBinding {
	return serviceBinding{
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
	}
}

func bandcampServiceBinding() serviceBinding {
	return serviceBinding{
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
	}
}

func deezerServiceBinding() serviceBinding {
	return serviceBinding{
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
	}
}

func soundCloudServiceBinding() serviceBinding {
	return serviceBinding{
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
	}
}

func spotifyServiceBinding() serviceBinding {
	return serviceBinding{
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
	}
}

func tidalServiceBinding() serviceBinding {
	return serviceBinding{
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
	}
}

func youTubeMusicServiceBinding() serviceBinding {
	return serviceBinding{
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
	}
}

func amazonMusicServiceBinding() serviceBinding {
	return serviceBinding{
		capability: serviceCapability{
			name:                ServiceAmazonMusic,
			aliases:             builtinServiceAliases(ServiceAmazonMusic),
			supportsAlbumSource: true,
		},
		build: func(client *http.Client, _ Config) serviceAdapterSet {
			adapter := amazonmusicadapter.New(client)
			return serviceAdapterSet{albumSource: adapter}
		},
	}
}
