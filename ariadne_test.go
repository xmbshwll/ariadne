package ariadne_test

import (
	"context"
	"errors"
	"testing"
	"time"

	ariadne "github.com/xmbshwll/ariadne"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testLibrarySourceURL = "https://fixture.test/source"

var (
	errUnsupportedLibrarySource = errors.New("unsupported")
	errLibraryTargetBoom        = errors.New("target boom")
)

func TestLoadConfigFromEnv(t *testing.T) {
	config := ariadne.LoadConfigFromEnv(func(key string) string {
		switch key {
		case "SPOTIFY_CLIENT_ID":
			return " spotify-client "
		case "SPOTIFY_CLIENT_SECRET":
			return " spotify-secret "
		case "APPLE_MUSIC_STOREFRONT":
			return " GB "
		case "APPLE_MUSIC_KEY_ID":
			return " music-key "
		case "APPLE_MUSIC_TEAM_ID":
			return " team-id "
		case "APPLE_MUSIC_PRIVATE_KEY_PATH":
			return " /tmp/AuthKey_TEST.p8 "
		case "TIDAL_CLIENT_ID":
			return " tidal-client "
		case "TIDAL_CLIENT_SECRET":
			return " tidal-secret "
		case "ARIADNE_HTTP_TIMEOUT":
			return " 45s "
		case "ARIADNE_TARGET_SERVICES":
			return " spotify, appleMusic, spotify "
		default:
			return ""
		}
	})

	assert.Equal(t, "gb", config.AppleMusicStorefront)
	assert.Equal(t, "spotify-client", config.Spotify.ClientID)
	assert.Equal(t, "spotify-secret", config.Spotify.ClientSecret)
	assert.Equal(t, "music-key", config.AppleMusic.KeyID)
	assert.Equal(t, "team-id", config.AppleMusic.TeamID)
	assert.Equal(t, "/tmp/AuthKey_TEST.p8", config.AppleMusic.PrivateKeyPath)
	assert.Equal(t, "tidal-client", config.TIDAL.ClientID)
	assert.Equal(t, "tidal-secret", config.TIDAL.ClientSecret)
	assert.Equal(t, 45*time.Second, config.HTTPTimeout)
	assert.Equal(t, []ariadne.ServiceName{ariadne.ServiceSpotify, ariadne.ServiceAppleMusic}, config.TargetServices)
}

func TestLoadConfigFromEnvCanonicalizesTargetServiceAliases(t *testing.T) {
	config := ariadne.LoadConfigFromEnv(func(key string) string {
		if key == "ARIADNE_TARGET_SERVICES" {
			return " spotify , ytmusic , amazonMusic , unknown "
		}
		return ""
	})

	assert.Equal(t, []ariadne.ServiceName{ariadne.ServiceSpotify, ariadne.ServiceYouTubeMusic}, config.TargetServices)
}

func TestDefaultConfig(t *testing.T) {
	config := ariadne.DefaultConfig()
	assert.Equal(t, "us", config.AppleMusicStorefront)
	assert.NotEqual(t, ariadne.ScoreWeights{}, config.ScoreWeights)
	assert.NotEqual(t, ariadne.SongScoreWeights{}, config.SongScoreWeights)
	assert.Equal(t, 15*time.Second, config.HTTPTimeout)
}

func TestAlbumEntityShapeSharesTheAnyEntityShapeTargetOrder(t *testing.T) {
	// Every built-in Target Service that searches songs also searches albums, so
	// both shapes resolve against the same Target Search order. A song-only target
	// would need its own order list, which is why the equivalence is pinned here.
	assert.Equal(t,
		ariadne.TargetServices(nil, ariadne.EntityShapeAny),
		ariadne.TargetServices(nil, ariadne.EntityShapeAlbum),
	)

	for _, raw := range []string{"appleMusic", "bandcamp", "deezer", "soundcloud", "youtubeMusic"} {
		decision := ariadne.EvaluateTarget(ariadne.Config{}, raw, ariadne.EntityShapeAlbum)
		assert.Equal(t, ariadne.TargetServiceRequestAvailable, decision.Status, raw)
	}
}

func TestCatalogQueriesTreatWhitespaceCredentialsAsMissing(t *testing.T) {
	// The Provider Catalog must agree with New: trimmed credentials are missing
	// credentials, otherwise the Catalog reports a Target Service as available
	// while the built Resolver silently omits it.
	config := ariadne.Config{
		Spotify: ariadne.SpotifyConfig{ClientID: "   ", ClientSecret: "   "},
		TIDAL:   ariadne.TIDALConfig{ClientID: "   ", ClientSecret: "   "},
	}

	for _, service := range []string{"spotify", "tidal"} {
		decision := ariadne.EvaluateTarget(config, service, ariadne.EntityShapeAny)
		assert.Equal(t, ariadne.TargetServiceRequestCredentialsRequired, decision.Status, service)
		assert.NotEmpty(t, decision.CredentialHint, service)
	}

	enabled, ok := ariadne.DescribeEnabled(config, ariadne.ServiceSpotify)
	assert.True(t, ok)
	assert.False(t, enabled.SupportsAlbumTarget)

	for _, service := range []ariadne.ServiceName{ariadne.ServiceSpotify, ariadne.ServiceTIDAL} {
		assert.NotContains(t, ariadne.TargetServices(&config, ariadne.EntityShapeAny), service)
	}
}

func TestCredentialEnablementTrimsWhitespace(t *testing.T) {
	tests := []struct {
		name string
		ok   bool
		fn   func() bool
	}{
		{
			name: "spotify client id whitespace",
			fn: func() bool {
				return ariadne.Config{
					Spotify: ariadne.SpotifyConfig{ClientID: " ", ClientSecret: "secret"},
				}.SpotifyEnabled()
			},
		},
		{
			name: "spotify client secret whitespace",
			fn: func() bool {
				return ariadne.Config{
					Spotify: ariadne.SpotifyConfig{ClientID: "id", ClientSecret: " "},
				}.SpotifyEnabled()
			},
		},
		{
			name: "tidal client id whitespace",
			fn: func() bool {
				return ariadne.Config{
					TIDAL: ariadne.TIDALConfig{ClientID: " ", ClientSecret: "secret"},
				}.TIDALEnabled()
			},
		},
		{
			name: "tidal client secret whitespace",
			fn: func() bool {
				return ariadne.Config{
					TIDAL: ariadne.TIDALConfig{ClientID: "id", ClientSecret: " "},
				}.TIDALEnabled()
			},
		},
		{
			name: "spotify trims valid credentials",
			ok:   true,
			fn: func() bool {
				return ariadne.Config{
					Spotify: ariadne.SpotifyConfig{ClientID: " id ", ClientSecret: " secret "},
				}.SpotifyEnabled()
			},
		},
		{
			name: "tidal trims valid credentials",
			ok:   true,
			fn: func() bool {
				return ariadne.Config{
					TIDAL: ariadne.TIDALConfig{ClientID: " id ", ClientSecret: " secret "},
				}.TIDALEnabled()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.ok, tt.fn())
		})
	}
}

func TestDescribe(t *testing.T) {
	tests := []struct {
		name                     string
		service                  ariadne.ServiceName
		aliases                  []string
		supportsAlbumSource      bool
		supportsAlbumTarget      bool
		supportsSongSource       bool
		supportsSongTarget       bool
		supportsRuntimeSongInput bool
	}{
		{name: "spotify full runtime", service: ariadne.ServiceSpotify, aliases: []string{"spotify"}, supportsAlbumSource: true, supportsAlbumTarget: true, supportsSongSource: true, supportsSongTarget: true, supportsRuntimeSongInput: true},
		{name: "youtube music defers song hydration", service: ariadne.ServiceYouTubeMusic, supportsAlbumSource: true, supportsAlbumTarget: true, supportsRuntimeSongInput: true},
		{name: "amazon music is source only", service: ariadne.ServiceAmazonMusic, supportsAlbumSource: true, supportsSongSource: true, supportsRuntimeSongInput: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capabilities, ok := ariadne.Describe(tt.service)
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
		config              ariadne.Config
		service             ariadne.ServiceName
		supportsAlbumTarget bool
		supportsSongTarget  bool
	}{
		{name: "spotify without credentials disables targets", config: ariadne.Config{}, service: ariadne.ServiceSpotify},
		{name: "spotify with credentials enables targets", config: ariadne.Config{Spotify: ariadne.SpotifyConfig{ClientID: "id", ClientSecret: "secret"}}, service: ariadne.ServiceSpotify, supportsAlbumTarget: true, supportsSongTarget: true},
		{name: "tidal without credentials disables targets", config: ariadne.Config{}, service: ariadne.ServiceTIDAL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capabilities, ok := ariadne.DescribeEnabled(tt.config, tt.service)
			require.True(t, ok)
			assert.Equal(t, tt.supportsAlbumTarget, capabilities.SupportsAlbumTarget)
			assert.Equal(t, tt.supportsSongTarget, capabilities.SupportsSongTarget)
		})
	}
}

func TestSupportedServiceLists(t *testing.T) {
	assert.Equal(t, []ariadne.ServiceName{
		ariadne.ServiceAppleMusic,
		ariadne.ServiceBandcamp,
		ariadne.ServiceDeezer,
		ariadne.ServiceSoundCloud,
		ariadne.ServiceYouTubeMusic,
		ariadne.ServiceSpotify,
		ariadne.ServiceTIDAL,
	}, ariadne.TargetServices(nil, ariadne.EntityShapeAny))
	assert.Equal(t, []ariadne.ServiceName{
		ariadne.ServiceAppleMusic,
		ariadne.ServiceBandcamp,
		ariadne.ServiceDeezer,
		ariadne.ServiceSoundCloud,
		ariadne.ServiceSpotify,
		ariadne.ServiceTIDAL,
	}, ariadne.TargetServices(nil, ariadne.EntityShapeSong))
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
		name   string
		config ariadne.Config
		raw    string
		want   ariadne.TargetServiceRequestDecision
	}{
		{name: "available alias", raw: "apple-music", want: ariadne.TargetServiceRequestDecision{Service: ariadne.ServiceAppleMusic, Status: ariadne.TargetServiceRequestAvailable}},
		{name: "unknown", raw: "unknown", want: ariadne.TargetServiceRequestDecision{Status: ariadne.TargetServiceRequestUnknown, Message: unknownTargetServiceMessage}},
		{name: "parse only", raw: "amazonMusic", want: ariadne.TargetServiceRequestDecision{Service: ariadne.ServiceAmazonMusic, Status: ariadne.TargetServiceRequestParseOnly, Message: amazonMusicTargetServiceMessage}},
		{name: "credentials required", raw: "spotify", want: ariadne.TargetServiceRequestDecision{Service: ariadne.ServiceSpotify, Status: ariadne.TargetServiceRequestCredentialsRequired, Message: spotifyTargetServiceMessage, CredentialHint: spotifyCredentialHint}},
		{name: "credentials configured", config: ariadne.Config{Spotify: ariadne.SpotifyConfig{ClientID: "id", ClientSecret: "secret"}}, raw: "spotify", want: ariadne.TargetServiceRequestDecision{Service: ariadne.ServiceSpotify, Status: ariadne.TargetServiceRequestAvailable}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ariadne.EvaluateTarget(tt.config, tt.raw, ariadne.EntityShapeAny))
		})
	}
}

func TestProviderCatalogSongTargetServiceRequests(t *testing.T) {
	tests := []struct {
		name    string
		config  ariadne.Config
		service ariadne.ServiceName
		want    ariadne.TargetServiceRequestDecision
	}{
		{name: "available", service: ariadne.ServiceAppleMusic, want: ariadne.TargetServiceRequestDecision{Service: ariadne.ServiceAppleMusic, Status: ariadne.TargetServiceRequestAvailable}},
		{name: "unsupported song target", service: ariadne.ServiceYouTubeMusic, want: ariadne.TargetServiceRequestDecision{Service: ariadne.ServiceYouTubeMusic, Status: ariadne.TargetServiceRequestUnsupported, Message: youTubeMusicSongTargetMessage}},
		{name: "parse only", service: ariadne.ServiceAmazonMusic, want: ariadne.TargetServiceRequestDecision{Service: ariadne.ServiceAmazonMusic, Status: ariadne.TargetServiceRequestParseOnly, Message: amazonMusicSongTargetMessage}},
		{name: "credentials required", service: ariadne.ServiceTIDAL, want: ariadne.TargetServiceRequestDecision{Service: ariadne.ServiceTIDAL, Status: ariadne.TargetServiceRequestCredentialsRequired, Message: tidalSongTargetMessage, CredentialHint: tidalCredentialHint}},
		{name: "credentials configured", config: ariadne.Config{TIDAL: ariadne.TIDALConfig{ClientID: "id", ClientSecret: "secret"}}, service: ariadne.ServiceTIDAL, want: ariadne.TargetServiceRequestDecision{Service: ariadne.ServiceTIDAL, Status: ariadne.TargetServiceRequestAvailable}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ariadne.EvaluateTarget(tt.config, string(tt.service), ariadne.EntityShapeSong))
		})
	}
}

// TestCredentialHintCoversEveryCredentialGatedService pins the invariant the CLI
// relies on: a decision that asks for credentials always explains which
// Credential Token is missing, so the CLI never has to name a service itself.
func TestCredentialHintCoversEveryCredentialGatedService(t *testing.T) {
	services := []ariadne.ServiceName{
		ariadne.ServiceAppleMusic, ariadne.ServiceSpotify, ariadne.ServiceDeezer,
		ariadne.ServiceSoundCloud, ariadne.ServiceBandcamp, ariadne.ServiceYouTubeMusic,
		ariadne.ServiceTIDAL, ariadne.ServiceAmazonMusic,
	}
	for _, service := range services {
		for _, entity := range []ariadne.EntityShape{ariadne.EntityShapeAny, ariadne.EntityShapeSong} {
			decision := ariadne.EvaluateTarget(ariadne.Config{}, string(service), entity)
			if decision.Status != ariadne.TargetServiceRequestCredentialsRequired {
				continue
			}
			assert.Contains(t, decision.CredentialHint, "must be set", "%s/%s", service, entity)
		}
	}
}

func TestEnabledServiceLists(t *testing.T) {
	empty := ariadne.Config{}
	assert.Equal(t, []ariadne.ServiceName{
		ariadne.ServiceAppleMusic,
		ariadne.ServiceBandcamp,
		ariadne.ServiceDeezer,
		ariadne.ServiceSoundCloud,
		ariadne.ServiceYouTubeMusic,
	}, ariadne.TargetServices(&empty, ariadne.EntityShapeAny))
	assert.Equal(t, []ariadne.ServiceName{
		ariadne.ServiceAppleMusic,
		ariadne.ServiceBandcamp,
		ariadne.ServiceDeezer,
		ariadne.ServiceSoundCloud,
	}, ariadne.TargetServices(&empty, ariadne.EntityShapeSong))

	config := ariadne.Config{
		Spotify: ariadne.SpotifyConfig{ClientID: "id", ClientSecret: "secret"},
		TIDAL:   ariadne.TIDALConfig{ClientID: "tidal-id", ClientSecret: "tidal-secret"},
	}
	assert.Equal(t, []ariadne.ServiceName{
		ariadne.ServiceAppleMusic,
		ariadne.ServiceBandcamp,
		ariadne.ServiceDeezer,
		ariadne.ServiceSoundCloud,
		ariadne.ServiceYouTubeMusic,
		ariadne.ServiceSpotify,
		ariadne.ServiceTIDAL,
	}, ariadne.TargetServices(&config, ariadne.EntityShapeAny))
	assert.Equal(t, []ariadne.ServiceName{
		ariadne.ServiceAppleMusic,
		ariadne.ServiceBandcamp,
		ariadne.ServiceDeezer,
		ariadne.ServiceSoundCloud,
		ariadne.ServiceSpotify,
		ariadne.ServiceTIDAL,
	}, ariadne.TargetServices(&config, ariadne.EntityShapeSong))

	spotify, ok := ariadne.DescribeEnabled(config, ariadne.ServiceSpotify)
	require.True(t, ok)
	assert.True(t, spotify.SupportsAlbumTarget)
	tidal, ok := ariadne.DescribeEnabled(config, ariadne.ServiceTIDAL)
	require.True(t, ok)
	assert.True(t, tidal.SupportsSongTarget)
	spotify, ok = ariadne.DescribeEnabled(empty, ariadne.ServiceSpotify)
	require.True(t, ok)
	assert.False(t, spotify.SupportsAlbumTarget)
	tidal, ok = ariadne.DescribeEnabled(empty, ariadne.ServiceTIDAL)
	require.True(t, ok)
	assert.False(t, tidal.SupportsSongTarget)
}

func TestNormalizedConfigDefaultsSongWeights(t *testing.T) {
	config := ariadne.NormalizedConfig(ariadne.Config{})
	assert.NotEqual(t, ariadne.SongScoreWeights{}, config.SongScoreWeights)
}

func TestMatchStrengthForScore(t *testing.T) {
	tests := []struct {
		score int
		want  ariadne.MatchStrength
	}{
		{score: 120, want: ariadne.MatchStrengthStrong},
		{score: 80, want: ariadne.MatchStrengthProbable},
		{score: 50, want: ariadne.MatchStrengthWeak},
		{score: 49, want: ariadne.MatchStrengthVeryWeak},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, ariadne.MatchStrengthForScore(tt.score))
	}
}

func TestNewWithAdaptersResolveAlbum(t *testing.T) {
	resolver := ariadne.NewWithAdapters(ariadne.AdapterSet{
		AlbumSources: []ariadne.SourceAdapter{newLibrarySourceAdapter()},
		AlbumTargets: []ariadne.TargetAdapter{newLibraryTargetAdapter()},
	})

	resolution, err := resolver.ResolveAlbum(context.Background(), testLibrarySourceURL)
	require.NoError(t, err)
	assert.Equal(t, ariadne.ServiceDeezer, resolution.Source.Service)
	match := resolution.Matches[ariadne.ServiceSpotify]
	require.NotNil(t, match.Best)
	assert.Equal(t, "spotify-1", match.Best.Candidate.CandidateID)
}

func TestNewWithAdaptersResolveSong(t *testing.T) {
	resolver := newTestEntityResolver()

	resolution, err := resolver.ResolveSong(context.Background(), "https://fixture.test/songs/1")
	require.NoError(t, err)
	assert.Equal(t, ariadne.ServiceSpotify, resolution.Source.Service)
	match := resolution.Matches[ariadne.ServiceAppleMusic]
	require.NotNil(t, match.Best)
	assert.Equal(t, "apple-song-1", match.Best.Candidate.CandidateID)
}

func TestResolverResolveDispatchesByEntityType(t *testing.T) {
	resolver := newTestEntityResolver()

	albumEntity, err := resolver.Resolve(context.Background(), testLibrarySourceURL)
	require.NoError(t, err)
	require.NotNil(t, albumEntity.Album)
	assert.Nil(t, albumEntity.Song)
	assert.Equal(t, "album", albumEntity.Parsed.EntityType)

	songEntity, err := resolver.Resolve(context.Background(), "https://fixture.test/songs/1")
	require.NoError(t, err)
	require.NotNil(t, songEntity.Song)
	assert.Nil(t, songEntity.Album)
	assert.Equal(t, "song", songEntity.Parsed.EntityType)
}

func TestResolveReturnsErrorForUninitializedResolver(t *testing.T) {
	tests := []struct {
		name    string
		resolve func() error
	}{
		{name: "nil resolver album", resolve: func() error {
			var resolver *ariadne.Resolver
			_, err := resolver.ResolveAlbum(context.Background(), testLibrarySourceURL)
			return err
		}},
		{name: "missing song resolver", resolve: func() error {
			resolver := &ariadne.Resolver{}
			_, err := resolver.ResolveSong(context.Background(), "https://fixture.test/songs/1")
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.ErrorIs(t, tt.resolve(), ariadne.ErrResolverNotInitialized)
		})
	}
}

func TestResolveReturnsPublicSentinelWhenCustomSourceViolatesContract(t *testing.T) {
	tests := []struct {
		name     string
		resolve  func() error
		sentinel error
	}{
		{name: "album source returns nil parsed url", sentinel: ariadne.ErrSourceAdapterReturnedNilParsedURL, resolve: func() error {
			resolver := ariadne.NewWithAdapters(ariadne.AdapterSet{AlbumSources: []ariadne.SourceAdapter{newNilParsedSourceAdapter()}, AlbumTargets: []ariadne.TargetAdapter{newLibraryTargetAdapter()}})
			_, err := resolver.ResolveAlbum(context.Background(), testLibrarySourceURL)
			return err
		}},
		{name: "album source returns nil album", sentinel: ariadne.ErrSourceAdapterReturnedNilAlbum, resolve: func() error {
			resolver := ariadne.NewWithAdapters(ariadne.AdapterSet{AlbumSources: []ariadne.SourceAdapter{newNilAlbumSourceAdapter()}, AlbumTargets: []ariadne.TargetAdapter{newLibraryTargetAdapter()}})
			_, err := resolver.ResolveAlbum(context.Background(), testLibrarySourceURL)
			return err
		}},
		{name: "song source returns nil song", sentinel: ariadne.ErrSourceAdapterReturnedNilSong, resolve: func() error {
			resolver := ariadne.NewWithAdapters(ariadne.AdapterSet{SongSources: []ariadne.SongSourceAdapter{newNilSongSourceAdapter()}, SongTargets: []ariadne.SongTargetAdapter{newLibrarySongTargetAdapter()}})
			_, err := resolver.ResolveSong(context.Background(), "https://fixture.test/songs/1")
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.ErrorIs(t, tt.resolve(), tt.sentinel)
		})
	}
}

func TestResolveSongReturnsDeferredRuntimeForParseOnlyServices(t *testing.T) {
	resolver := ariadne.New(ariadne.DefaultConfig())
	tests := []struct {
		name       string
		inputURL   string
		serviceErr error
	}{
		{
			name:       "youtube music",
			inputURL:   "https://music.youtube.com/watch?v=dQw4w9WgXcQ",
			serviceErr: ariadne.ErrYouTubeMusicDeferred,
		},
		{
			name:       "amazon music",
			inputURL:   "https://music.amazon.com/albums/B0064UPU4G?trackAsin=B0064TRACK",
			serviceErr: ariadne.ErrAmazonMusicDeferred,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, ariadne.SupportsRuntimeSongInputURL(tt.inputURL))

			resolution, err := resolver.ResolveSong(context.Background(), tt.inputURL)
			require.Error(t, err)
			assert.Nil(t, resolution)
			assert.ErrorIs(t, err, ariadne.ErrRuntimeDeferred)
			assert.ErrorIs(t, err, tt.serviceErr)
		})
	}
}

func TestResolveAlbumPreservesCustomTargetErrors(t *testing.T) {
	resolver := ariadne.NewWithAdapters(ariadne.AdapterSet{AlbumSources: []ariadne.SourceAdapter{newLibrarySourceAdapter()}, AlbumTargets: []ariadne.TargetAdapter{newFailingLibraryTargetAdapter()}})

	resolution, err := resolver.ResolveAlbum(context.Background(), testLibrarySourceURL)
	require.NoError(t, err)
	require.NotNil(t, resolution)
	assert.ErrorIs(t, resolution.Matches[ariadne.ServiceSpotify].Err, errLibraryTargetBoom)
}

func newTestEntityResolver() *ariadne.Resolver {
	return ariadne.NewWithAdapters(ariadne.AdapterSet{
		AlbumSources: []ariadne.SourceAdapter{newLibrarySourceAdapter()},
		AlbumTargets: []ariadne.TargetAdapter{newLibraryTargetAdapter()},
		SongSources:  []ariadne.SongSourceAdapter{newLibrarySongSourceAdapter()},
		SongTargets:  []ariadne.SongTargetAdapter{newLibrarySongTargetAdapter()},
	})
}
