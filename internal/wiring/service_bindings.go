package wiring

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
	"github.com/xmbshwll/ariadne/internal/config"
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/resolve"
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

type capabilitySet struct {
	AlbumSource bool
	AlbumTarget bool
	SongSource  bool
	SongTarget  bool
}

var (
	allRuntimeCapabilities   = capabilitySet{AlbumSource: true, AlbumTarget: true, SongSource: true, SongTarget: true}
	albumRuntimeCapabilities = capabilitySet{AlbumSource: true, AlbumTarget: true}
	sourceOnlyCapabilities   = capabilitySet{AlbumSource: true, SongSource: true}
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

type bindingSpec struct {
	service             model.ServiceName
	aliases             []string
	capabilities        capabilitySet
	targetSearchEnabled func(config.Config) bool
	build               adapterBuilder
}

func (s bindingSpec) capability() capabilitySpec {
	return capabilitySpec{
		name:                s.service,
		aliases:             append([]string(nil), s.aliases...),
		supportsAlbumSource: s.capabilities.AlbumSource,
		supportsAlbumTarget: s.capabilities.AlbumTarget,
		supportsSongSource:  s.capabilities.SongSource,
		supportsSongTarget:  s.capabilities.SongTarget,
		targetSearchEnabled: s.targetSearchEnabled,
	}
}

func newServiceBinding(spec bindingSpec) binding {
	return binding{
		capability: spec.capability(),
		build:      spec.build,
	}
}

func fullRuntimeAdapterSet(adapter fullRuntimeAdapter) adapterSet {
	return adapterSet{
		AlbumSource: adapter,
		AlbumTarget: adapter,
		SongSource:  adapter,
		SongTarget:  adapter,
	}
}

func albumRuntimeAdapterSet(adapter albumRuntimeAdapter) adapterSet {
	return adapterSet{AlbumSource: adapter, AlbumTarget: adapter}
}

func sourceRuntimeAdapterSet(adapter sourceRuntimeAdapter) adapterSet {
	return adapterSet{AlbumSource: adapter, SongSource: adapter}
}

func credentialedFullRuntimeAdapterSet(adapter fullRuntimeAdapter, targetSearchEnabled bool) adapterSet {
	set := sourceRuntimeAdapterSet(adapter)
	if targetSearchEnabled {
		set.AlbumTarget = adapter
		set.SongTarget = adapter
	}
	return set
}

func appleMusicServiceBinding() binding {
	return newServiceBinding(bindingSpec{
		service:      model.ServiceAppleMusic,
		aliases:      []string{"applemusic"},
		capabilities: allRuntimeCapabilities,
		build: func(client *http.Client, cfg config.Config) adapterSet {
			adapter := applemusicadapter.New(
				client,
				applemusicadapter.WithDefaultStorefront(cfg.AppleMusic.Storefront),
				applemusicadapter.WithDeveloperTokenAuth(
					cfg.AppleMusic.KeyID,
					cfg.AppleMusic.TeamID,
					cfg.AppleMusic.PrivateKeyPath,
				),
			)
			return fullRuntimeAdapterSet(adapter)
		},
	})
}

func bandcampServiceBinding() binding {
	return newServiceBinding(bindingSpec{
		service:      model.ServiceBandcamp,
		aliases:      []string{"bandcamp"},
		capabilities: allRuntimeCapabilities,
		build: func(client *http.Client, _ config.Config) adapterSet {
			return fullRuntimeAdapterSet(bandcampadapter.New(client))
		},
	})
}

func deezerServiceBinding() binding {
	return newServiceBinding(bindingSpec{
		service:      model.ServiceDeezer,
		aliases:      []string{"deezer"},
		capabilities: allRuntimeCapabilities,
		build: func(client *http.Client, _ config.Config) adapterSet {
			return fullRuntimeAdapterSet(deezeradapter.New(client))
		},
	})
}

func soundCloudServiceBinding() binding {
	return newServiceBinding(bindingSpec{
		service:      model.ServiceSoundCloud,
		aliases:      []string{"soundcloud"},
		capabilities: allRuntimeCapabilities,
		build: func(client *http.Client, _ config.Config) adapterSet {
			return fullRuntimeAdapterSet(soundcloudadapter.New(client))
		},
	})
}

func spotifyServiceBinding() binding {
	return newServiceBinding(bindingSpec{
		service:             model.ServiceSpotify,
		aliases:             []string{"spotify"},
		capabilities:        allRuntimeCapabilities,
		targetSearchEnabled: spotifyEnabled,
		build: func(client *http.Client, cfg config.Config) adapterSet {
			adapter := spotifyadapter.New(
				client,
				spotifyadapter.WithCredentials(cfg.Spotify.ClientID, cfg.Spotify.ClientSecret),
			)
			return credentialedFullRuntimeAdapterSet(adapter, cfg.Spotify.Enabled())
		},
	})
}

func tidalServiceBinding() binding {
	return newServiceBinding(bindingSpec{
		service:             model.ServiceTIDAL,
		aliases:             []string{"tidal"},
		capabilities:        allRuntimeCapabilities,
		targetSearchEnabled: tidalEnabled,
		build: func(client *http.Client, cfg config.Config) adapterSet {
			adapter := tidaladapter.New(
				client,
				tidaladapter.WithCredentials(cfg.TIDAL.ClientID, cfg.TIDAL.ClientSecret),
			)
			return credentialedFullRuntimeAdapterSet(adapter, cfg.TIDAL.Enabled())
		},
	})
}

func youTubeMusicServiceBinding() binding {
	return newServiceBinding(bindingSpec{
		service:      model.ServiceYouTubeMusic,
		aliases:      []string{"youtubemusic", "ytmusic"},
		capabilities: albumRuntimeCapabilities,
		build: func(client *http.Client, _ config.Config) adapterSet {
			adapter := youtubemusicadapter.New(client)
			set := albumRuntimeAdapterSet(adapter)
			set.SongSource = adapter
			return set
		},
	})
}

func amazonMusicServiceBinding() binding {
	return newServiceBinding(bindingSpec{
		service:      model.ServiceAmazonMusic,
		aliases:      []string{"amazonmusic", "amazon"},
		capabilities: sourceOnlyCapabilities,
		build: func(*http.Client, config.Config) adapterSet {
			return sourceRuntimeAdapterSet(amazonmusicadapter.New(nil))
		},
	})
}
