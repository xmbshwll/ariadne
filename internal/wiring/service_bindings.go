package wiring

import (
	"net/http"

	"github.com/xmbshwll/ariadne/internal/adapters"
	amazonmusicadapter "github.com/xmbshwll/ariadne/internal/adapters/amazonmusic"
	applemusicadapter "github.com/xmbshwll/ariadne/internal/adapters/applemusic"
	bandcampadapter "github.com/xmbshwll/ariadne/internal/adapters/bandcamp"
	deezeradapter "github.com/xmbshwll/ariadne/internal/adapters/deezer"
	soundcloudadapter "github.com/xmbshwll/ariadne/internal/adapters/soundcloud"
	spotifyadapter "github.com/xmbshwll/ariadne/internal/adapters/spotify"
	tidaladapter "github.com/xmbshwll/ariadne/internal/adapters/tidal"
	youtubemusicadapter "github.com/xmbshwll/ariadne/internal/adapters/youtubemusic"
	"github.com/xmbshwll/ariadne/internal/config"
	"github.com/xmbshwll/ariadne/internal/model"
)

var defaultBindings = []bindingSpec{
	appleMusicServiceBinding(),
	bandcampServiceBinding(),
	deezerServiceBinding(),
	soundCloudServiceBinding(),
	spotifyServiceBinding(),
	tidalServiceBinding(),
	youTubeMusicServiceBinding(),
	amazonMusicServiceBinding(),
}

// bindingSpec declares one Music Service for the Provider Catalog: how to look
// it up, whether Credential Tokens gate its Target Search role, and how to build
// the adapter. The Capabilities come from the built adapter itself, so a service
// states its support once.
type bindingSpec struct {
	service             model.ServiceName
	aliases             []string
	targetSearchEnabled func(config.Config) bool
	build               adapterBuilder
}

func appleMusicServiceBinding() bindingSpec {
	return bindingSpec{
		service: model.ServiceAppleMusic,
		aliases: []string{"applemusic"},
		build: func(client *http.Client, cfg config.Config) adapters.Adapter {
			return applemusicadapter.New(
				client,
				applemusicadapter.WithDefaultStorefront(cfg.AppleMusic.Storefront),
				applemusicadapter.WithDeveloperTokenAuth(
					cfg.AppleMusic.KeyID,
					cfg.AppleMusic.TeamID,
					cfg.AppleMusic.PrivateKeyPath,
				),
			)
		},
	}
}

func bandcampServiceBinding() bindingSpec {
	return bindingSpec{
		service: model.ServiceBandcamp,
		aliases: []string{"bandcamp"},
		build: func(client *http.Client, _ config.Config) adapters.Adapter {
			return bandcampadapter.New(client)
		},
	}
}

func deezerServiceBinding() bindingSpec {
	return bindingSpec{
		service: model.ServiceDeezer,
		aliases: []string{"deezer"},
		build: func(client *http.Client, _ config.Config) adapters.Adapter {
			return deezeradapter.New(client)
		},
	}
}

func soundCloudServiceBinding() bindingSpec {
	return bindingSpec{
		service: model.ServiceSoundCloud,
		aliases: []string{"soundcloud"},
		build: func(client *http.Client, _ config.Config) adapters.Adapter {
			return soundcloudadapter.New(client)
		},
	}
}

func spotifyServiceBinding() bindingSpec {
	return bindingSpec{
		service:             model.ServiceSpotify,
		aliases:             []string{"spotify"},
		targetSearchEnabled: spotifyEnabled,
		build: func(client *http.Client, cfg config.Config) adapters.Adapter {
			return spotifyadapter.New(
				client,
				spotifyadapter.WithCredentials(cfg.Spotify.ClientID, cfg.Spotify.ClientSecret),
			)
		},
	}
}

func tidalServiceBinding() bindingSpec {
	return bindingSpec{
		service:             model.ServiceTIDAL,
		aliases:             []string{"tidal"},
		targetSearchEnabled: tidalEnabled,
		build: func(client *http.Client, cfg config.Config) adapters.Adapter {
			return tidaladapter.New(
				client,
				tidaladapter.WithCredentials(cfg.TIDAL.ClientID, cfg.TIDAL.ClientSecret),
			)
		},
	}
}

func youTubeMusicServiceBinding() bindingSpec {
	return bindingSpec{
		service: model.ServiceYouTubeMusic,
		aliases: []string{"youtubemusic", "ytmusic"},
		build: func(client *http.Client, _ config.Config) adapters.Adapter {
			return youtubemusicadapter.New(client)
		},
	}
}

func amazonMusicServiceBinding() bindingSpec {
	return bindingSpec{
		service: model.ServiceAmazonMusic,
		aliases: []string{"amazonmusic", "amazon"},
		build: func(*http.Client, config.Config) adapters.Adapter {
			return amazonmusicadapter.New(nil)
		},
	}
}
