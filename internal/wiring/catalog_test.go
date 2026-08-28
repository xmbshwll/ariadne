package wiring_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xmbshwll/ariadne/internal/config"
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/wiring"
)

func TestAlbumEntityShapeSharesTheAnyEntityShapeTargetOrder(t *testing.T) {
	// Every built-in Target Service that searches songs also searches albums, so
	// both shapes resolve against the same Target Search order. A song-only target
	// would need its own order list, which is why the equivalence is pinned here.
	assert.Equal(t,
		wiring.Default.TargetServices(nil, wiring.EntityShapeAny),
		wiring.Default.TargetServices(nil, wiring.EntityShapeAlbum),
	)

	for _, raw := range []string{"appleMusic", "bandcamp", "deezer", "soundcloud", "youtubeMusic"} {
		decision := wiring.Default.EvaluateTarget(config.Config{}, raw, wiring.EntityShapeAlbum)
		assert.Equal(t, wiring.TargetServiceRequestAvailable, decision.Status, raw)
	}
}

func TestCatalogQueriesTreatWhitespaceCredentialsAsMissing(t *testing.T) {
	// The Provider Catalog must agree with New: trimmed credentials are missing
	// credentials, otherwise the Catalog reports a Target Service as available
	// while the built Resolver silently omits it.
	cfg := config.Config{
		Spotify: config.Spotify{ClientID: "   ", ClientSecret: "   "},
		TIDAL:   config.TIDAL{ClientID: "   ", ClientSecret: "   "},
	}

	for _, service := range []string{"spotify", "tidal"} {
		decision := wiring.Default.EvaluateTarget(cfg, service, wiring.EntityShapeAny)
		assert.Equal(t, wiring.TargetServiceRequestCredentialsRequired, decision.Status, service)
		assert.NotEmpty(t, decision.CredentialHint, service)
	}

	enabled, ok := wiring.Default.DescribeEnabledService(cfg, model.ServiceSpotify)
	assert.True(t, ok)
	assert.False(t, enabled.SupportsAlbumTarget)

	for _, service := range []model.ServiceName{model.ServiceSpotify, model.ServiceTIDAL} {
		assert.NotContains(t, wiring.Default.TargetServices(&cfg, wiring.EntityShapeAny), service)
	}
}

func TestDescribe(t *testing.T) {
	tests := []struct {
		name                     string
		service                  model.ServiceName
		aliases                  []string
		supportsAlbumSource      bool
		supportsAlbumTarget      bool
		supportsSongSource       bool
		supportsSongTarget       bool
		supportsRuntimeSongInput bool
	}{
		{name: "spotify full runtime", service: model.ServiceSpotify, aliases: []string{"spotify"}, supportsAlbumSource: true, supportsAlbumTarget: true, supportsSongSource: true, supportsSongTarget: true, supportsRuntimeSongInput: true},
		// YouTube Music recognizes song URLs and defers only song hydration, so
		// it takes song Source Input like Amazon Music does; Support for a Source
		// role means "this service can supply the Entity Shape", not "hydration is
		// implemented".
		{name: "youtube music defers song hydration", service: model.ServiceYouTubeMusic, supportsAlbumSource: true, supportsAlbumTarget: true, supportsSongSource: true, supportsRuntimeSongInput: true},
		{name: "amazon music is source only", service: model.ServiceAmazonMusic, supportsAlbumSource: true, supportsSongSource: true, supportsRuntimeSongInput: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capabilities, ok := wiring.Default.DescribeService(tt.service)
			require.True(t, ok)
			if tt.aliases != nil {
				assert.Equal(t, tt.aliases, capabilities.Aliases)
			}
			assert.Equal(t, tt.supportsAlbumSource, capabilities.SupportsAlbumSource)
			assert.Equal(t, tt.supportsAlbumTarget, capabilities.SupportsAlbumTarget)
			assert.Equal(t, tt.supportsSongSource, capabilities.SupportsSongSource)
			assert.Equal(t, tt.supportsSongTarget, capabilities.SupportsSongTarget)
			assert.Equal(t, tt.supportsRuntimeSongInput, capabilities.SupportsRuntimeSongInputURL)
		})
	}
}

func TestDescribeEnabled(t *testing.T) {
	tests := []struct {
		name                string
		cfg                 config.Config
		service             model.ServiceName
		supportsAlbumTarget bool
		supportsSongTarget  bool
	}{
		{name: "spotify without credentials disables targets", cfg: config.Config{}, service: model.ServiceSpotify},
		{name: "spotify with credentials enables targets", cfg: config.Config{Spotify: config.Spotify{ClientID: "id", ClientSecret: "secret"}}, service: model.ServiceSpotify, supportsAlbumTarget: true, supportsSongTarget: true},
		{name: "tidal without credentials disables targets", cfg: config.Config{}, service: model.ServiceTIDAL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capabilities, ok := wiring.Default.DescribeEnabledService(tt.cfg, tt.service)
			require.True(t, ok)
			assert.Equal(t, tt.supportsAlbumTarget, capabilities.SupportsAlbumTarget)
			assert.Equal(t, tt.supportsSongTarget, capabilities.SupportsSongTarget)
		})
	}
}

func TestSupportedServiceLists(t *testing.T) {
	assert.Equal(t, []model.ServiceName{
		model.ServiceAppleMusic,
		model.ServiceBandcamp,
		model.ServiceDeezer,
		model.ServiceSoundCloud,
		model.ServiceYouTubeMusic,
		model.ServiceSpotify,
		model.ServiceTIDAL,
	}, wiring.Default.TargetServices(nil, wiring.EntityShapeAny))
	assert.Equal(t, []model.ServiceName{
		model.ServiceAppleMusic,
		model.ServiceBandcamp,
		model.ServiceDeezer,
		model.ServiceSoundCloud,
		model.ServiceSpotify,
		model.ServiceTIDAL,
	}, wiring.Default.TargetServices(nil, wiring.EntityShapeSong))
}

const (
	unknownTargetServiceMessage     = "\"unknown\" (expected one of the supported target services: appleMusic, bandcamp, deezer, soundcloud, youtubeMusic, spotify, tidal)"
	amazonMusicTargetServiceMessage = "\"amazonMusic\" (expected one of the supported target services: appleMusic, bandcamp, deezer, soundcloud, youtubeMusic, spotify, tidal)"
	spotifyTargetServiceMessage     = "\"spotify\" (expected one of the supported target services: appleMusic, bandcamp, deezer, soundcloud, youtubeMusic, spotify, tidal)"
	spotifyCredentialHint           = "SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET must be set"
	tidalCredentialHint             = "TIDAL_CLIENT_ID and TIDAL_CLIENT_SECRET must be set"
	youTubeMusicSongTargetMessage   = "\"youtubeMusic\" (enabled for songs: appleMusic, bandcamp, deezer, soundcloud)"
	amazonMusicSongTargetMessage    = "\"amazonMusic\" (enabled for songs: appleMusic, bandcamp, deezer, soundcloud)"
	tidalSongTargetMessage          = "\"tidal\" (enabled for songs: appleMusic, bandcamp, deezer, soundcloud)"
)

func TestProviderCatalogTargetServiceRequests(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.Config
		raw  string
		want wiring.TargetServiceRequestDecision
	}{
		{name: "available alias", raw: "apple-music", want: wiring.TargetServiceRequestDecision{Service: model.ServiceAppleMusic, Status: wiring.TargetServiceRequestAvailable}},
		{name: "unknown", raw: "unknown", want: wiring.TargetServiceRequestDecision{Status: wiring.TargetServiceRequestUnknown, Message: unknownTargetServiceMessage}},
		{name: "parse only", raw: "amazonMusic", want: wiring.TargetServiceRequestDecision{Service: model.ServiceAmazonMusic, Status: wiring.TargetServiceRequestParseOnly, Message: amazonMusicTargetServiceMessage}},
		{name: "credentials required", raw: "spotify", want: wiring.TargetServiceRequestDecision{Service: model.ServiceSpotify, Status: wiring.TargetServiceRequestCredentialsRequired, Message: spotifyTargetServiceMessage, CredentialHint: spotifyCredentialHint}},
		{name: "credentials configured", cfg: config.Config{Spotify: config.Spotify{ClientID: "id", ClientSecret: "secret"}}, raw: "spotify", want: wiring.TargetServiceRequestDecision{Service: model.ServiceSpotify, Status: wiring.TargetServiceRequestAvailable}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, wiring.Default.EvaluateTarget(tt.cfg, tt.raw, wiring.EntityShapeAny))
		})
	}
}

func TestProviderCatalogSongTargetServiceRequests(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.Config
		service model.ServiceName
		want    wiring.TargetServiceRequestDecision
	}{
		{name: "available", service: model.ServiceAppleMusic, want: wiring.TargetServiceRequestDecision{Service: model.ServiceAppleMusic, Status: wiring.TargetServiceRequestAvailable}},
		{name: "unsupported song target", service: model.ServiceYouTubeMusic, want: wiring.TargetServiceRequestDecision{Service: model.ServiceYouTubeMusic, Status: wiring.TargetServiceRequestUnsupported, Message: youTubeMusicSongTargetMessage}},
		{name: "parse only", service: model.ServiceAmazonMusic, want: wiring.TargetServiceRequestDecision{Service: model.ServiceAmazonMusic, Status: wiring.TargetServiceRequestParseOnly, Message: amazonMusicSongTargetMessage}},
		{name: "credentials required", service: model.ServiceTIDAL, want: wiring.TargetServiceRequestDecision{Service: model.ServiceTIDAL, Status: wiring.TargetServiceRequestCredentialsRequired, Message: tidalSongTargetMessage, CredentialHint: tidalCredentialHint}},
		{name: "credentials configured", cfg: config.Config{TIDAL: config.TIDAL{ClientID: "id", ClientSecret: "secret"}}, service: model.ServiceTIDAL, want: wiring.TargetServiceRequestDecision{Service: model.ServiceTIDAL, Status: wiring.TargetServiceRequestAvailable}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, wiring.Default.EvaluateTarget(tt.cfg, string(tt.service), wiring.EntityShapeSong))
		})
	}
}

// TestCredentialHintCoversEveryCredentialGatedService pins the invariant the CLI
// relies on: a decision that asks for credentials always explains which
// Credential Token is missing, so the CLI never has to name a service itself.
func TestCredentialHintCoversEveryCredentialGatedService(t *testing.T) {
	services := []model.ServiceName{
		model.ServiceAppleMusic, model.ServiceSpotify, model.ServiceDeezer,
		model.ServiceSoundCloud, model.ServiceBandcamp, model.ServiceYouTubeMusic,
		model.ServiceTIDAL, model.ServiceAmazonMusic,
	}
	for _, service := range services {
		for _, entity := range []wiring.EntityShape{wiring.EntityShapeAny, wiring.EntityShapeSong} {
			decision := wiring.Default.EvaluateTarget(config.Config{}, string(service), entity)
			if decision.Status != wiring.TargetServiceRequestCredentialsRequired {
				continue
			}
			assert.Contains(t, decision.CredentialHint, "must be set", "%s/%s", service, entity)
		}
	}
}

func TestEnabledServiceLists(t *testing.T) {
	empty := config.Config{}
	assert.Equal(t, []model.ServiceName{
		model.ServiceAppleMusic,
		model.ServiceBandcamp,
		model.ServiceDeezer,
		model.ServiceSoundCloud,
		model.ServiceYouTubeMusic,
	}, wiring.Default.TargetServices(&empty, wiring.EntityShapeAny))
	assert.Equal(t, []model.ServiceName{
		model.ServiceAppleMusic,
		model.ServiceBandcamp,
		model.ServiceDeezer,
		model.ServiceSoundCloud,
	}, wiring.Default.TargetServices(&empty, wiring.EntityShapeSong))

	cfg := config.Config{
		Spotify: config.Spotify{ClientID: "id", ClientSecret: "secret"},
		TIDAL:   config.TIDAL{ClientID: "tidal-id", ClientSecret: "tidal-secret"},
	}
	assert.Equal(t, []model.ServiceName{
		model.ServiceAppleMusic,
		model.ServiceBandcamp,
		model.ServiceDeezer,
		model.ServiceSoundCloud,
		model.ServiceYouTubeMusic,
		model.ServiceSpotify,
		model.ServiceTIDAL,
	}, wiring.Default.TargetServices(&cfg, wiring.EntityShapeAny))
	assert.Equal(t, []model.ServiceName{
		model.ServiceAppleMusic,
		model.ServiceBandcamp,
		model.ServiceDeezer,
		model.ServiceSoundCloud,
		model.ServiceSpotify,
		model.ServiceTIDAL,
	}, wiring.Default.TargetServices(&cfg, wiring.EntityShapeSong))

	spotify, ok := wiring.Default.DescribeEnabledService(cfg, model.ServiceSpotify)
	require.True(t, ok)
	assert.True(t, spotify.SupportsAlbumTarget)
	tidal, ok := wiring.Default.DescribeEnabledService(cfg, model.ServiceTIDAL)
	require.True(t, ok)
	assert.True(t, tidal.SupportsSongTarget)
	spotify, ok = wiring.Default.DescribeEnabledService(empty, model.ServiceSpotify)
	require.True(t, ok)
	assert.False(t, spotify.SupportsAlbumTarget)
	tidal, ok = wiring.Default.DescribeEnabledService(empty, model.ServiceTIDAL)
	require.True(t, ok)
	assert.False(t, tidal.SupportsSongTarget)
}
