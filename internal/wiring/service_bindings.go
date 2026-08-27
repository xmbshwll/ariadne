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

var defaultBindings = []binding{
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
// it up, which Capabilities its adapter declares, whether Credential Tokens gate
// its Target Search role, and how to build the adapter.
type bindingSpec struct {
	service             model.ServiceName
	aliases             []string
	capabilities        adapters.Capabilities
	targetSearchEnabled func(config.Config) bool
	build               adapterBuilder
}

func (s bindingSpec) capability() capabilitySpec {
	return capabilitySpec{
		name:                s.service,
		aliases:             append([]string(nil), s.aliases...),
		capabilities:        s.capabilities,
		targetSearchEnabled: s.targetSearchEnabled,
	}
}

func newServiceBinding(spec bindingSpec) binding {
	return binding{
		capability: spec.capability(),
		build:      spec.build,
	}
}

func appleMusicServiceBinding() binding {
	return newServiceBinding(bindingSpec{
		service:      model.ServiceAppleMusic,
		aliases:      []string{"applemusic"},
		capabilities: applemusicadapter.Capabilities(),
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
	})
}

func bandcampServiceBinding() binding {
	return newServiceBinding(bindingSpec{
		service:      model.ServiceBandcamp,
		aliases:      []string{"bandcamp"},
		capabilities: bandcampadapter.Capabilities(),
		build: func(client *http.Client, _ config.Config) adapters.Adapter {
			return bandcampadapter.New(client)
		},
	})
}

func deezerServiceBinding() binding {
	return newServiceBinding(bindingSpec{
		service:      model.ServiceDeezer,
		aliases:      []string{"deezer"},
		capabilities: deezeradapter.Capabilities(),
		build: func(client *http.Client, _ config.Config) adapters.Adapter {
			return deezeradapter.New(client)
		},
	})
}

func soundCloudServiceBinding() binding {
	return newServiceBinding(bindingSpec{
		service:      model.ServiceSoundCloud,
		aliases:      []string{"soundcloud"},
		capabilities: soundcloudadapter.Capabilities(),
		build: func(client *http.Client, _ config.Config) adapters.Adapter {
			return soundcloudadapter.New(client)
		},
	})
}

func spotifyServiceBinding() binding {
	return newServiceBinding(bindingSpec{
		service:             model.ServiceSpotify,
		aliases:             []string{"spotify"},
		capabilities:        spotifyadapter.Capabilities(),
		targetSearchEnabled: spotifyEnabled,
		build: func(client *http.Client, cfg config.Config) adapters.Adapter {
			return spotifyadapter.New(
				client,
				spotifyadapter.WithCredentials(cfg.Spotify.ClientID, cfg.Spotify.ClientSecret),
			)
		},
	})
}

func tidalServiceBinding() binding {
	return newServiceBinding(bindingSpec{
		service:             model.ServiceTIDAL,
		aliases:             []string{"tidal"},
		capabilities:        tidaladapter.Capabilities(),
		targetSearchEnabled: tidalEnabled,
		build: func(client *http.Client, cfg config.Config) adapters.Adapter {
			return tidaladapter.New(
				client,
				tidaladapter.WithCredentials(cfg.TIDAL.ClientID, cfg.TIDAL.ClientSecret),
			)
		},
	})
}

func youTubeMusicServiceBinding() binding {
	return newServiceBinding(bindingSpec{
		service:      model.ServiceYouTubeMusic,
		aliases:      []string{"youtubemusic", "ytmusic"},
		capabilities: youtubemusicadapter.Capabilities(),
		build: func(client *http.Client, _ config.Config) adapters.Adapter {
			return youtubemusicadapter.New(client)
		},
	})
}

func amazonMusicServiceBinding() binding {
	return newServiceBinding(bindingSpec{
		service:      model.ServiceAmazonMusic,
		aliases:      []string{"amazonmusic", "amazon"},
		capabilities: amazonmusicadapter.Capabilities(),
		build: func(*http.Client, config.Config) adapters.Adapter {
			return amazonmusicadapter.New(nil)
		},
	})
}
