package ariadne

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testLibrarySourceURL = "https://fixture.test/source"

var (
	errUnsupportedLibrarySource = errors.New("unsupported")
	errLibraryTargetBoom        = errors.New("target boom")
)

func TestLoadConfigFromEnv(t *testing.T) {
	config := LoadConfigFromEnv(func(key string) string {
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
	assert.Equal(t, []ServiceName{ServiceSpotify, ServiceAppleMusic}, config.TargetServices)
}

func TestLoadConfigFromEnvCanonicalizesTargetServiceAliases(t *testing.T) {
	config := LoadConfigFromEnv(func(key string) string {
		if key == "ARIADNE_TARGET_SERVICES" {
			return " spotify , ytmusic , amazonMusic , unknown "
		}
		return ""
	})

	assert.Equal(t, []ServiceName{ServiceSpotify, ServiceYouTubeMusic}, config.TargetServices)
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	assert.Equal(t, "us", config.AppleMusicStorefront)
	assert.NotEqual(t, ScoreWeights{}, config.ScoreWeights)
	assert.NotEqual(t, SongScoreWeights{}, config.SongScoreWeights)
	assert.Equal(t, 15*time.Second, config.HTTPTimeout)
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
				return Config{
					Spotify: SpotifyConfig{ClientID: " ", ClientSecret: "secret"},
				}.SpotifyEnabled()
			},
		},
		{
			name: "spotify client secret whitespace",
			fn: func() bool {
				return Config{
					Spotify: SpotifyConfig{ClientID: "id", ClientSecret: " "},
				}.SpotifyEnabled()
			},
		},
		{
			name: "tidal client id whitespace",
			fn: func() bool {
				return Config{
					TIDAL: TIDALConfig{ClientID: " ", ClientSecret: "secret"},
				}.TIDALEnabled()
			},
		},
		{
			name: "tidal client secret whitespace",
			fn: func() bool {
				return Config{
					TIDAL: TIDALConfig{ClientID: "id", ClientSecret: " "},
				}.TIDALEnabled()
			},
		},
		{
			name: "spotify trims valid credentials",
			ok:   true,
			fn: func() bool {
				return Config{
					Spotify: SpotifyConfig{ClientID: " id ", ClientSecret: " secret "},
				}.SpotifyEnabled()
			},
		},
		{
			name: "tidal trims valid credentials",
			ok:   true,
			fn: func() bool {
				return Config{
					TIDAL: TIDALConfig{ClientID: " id ", ClientSecret: " secret "},
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

func TestDescribeService(t *testing.T) {
	tests := []struct {
		name                     string
		service                  ServiceName
		aliases                  []string
		supportsAlbumSource      bool
		supportsAlbumTarget      bool
		supportsSongSource       bool
		supportsSongTarget       bool
		supportsRuntimeSongInput bool
	}{
		{name: "spotify full runtime", service: ServiceSpotify, aliases: []string{"spotify"}, supportsAlbumSource: true, supportsAlbumTarget: true, supportsSongSource: true, supportsSongTarget: true, supportsRuntimeSongInput: true},
		{name: "youtube music defers song hydration", service: ServiceYouTubeMusic, supportsAlbumSource: true, supportsAlbumTarget: true, supportsRuntimeSongInput: true},
		{name: "amazon music is source only", service: ServiceAmazonMusic, supportsAlbumSource: true, supportsSongSource: true, supportsRuntimeSongInput: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capabilities, ok := DescribeService(tt.service)
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

func TestDescribeEnabledService(t *testing.T) {
	tests := []struct {
		name                string
		config              Config
		service             ServiceName
		supportsAlbumTarget bool
		supportsSongTarget  bool
	}{
		{name: "spotify without credentials disables targets", config: Config{}, service: ServiceSpotify},
		{name: "spotify with credentials enables targets", config: Config{Spotify: SpotifyConfig{ClientID: "id", ClientSecret: "secret"}}, service: ServiceSpotify, supportsAlbumTarget: true, supportsSongTarget: true},
		{name: "tidal without credentials disables targets", config: Config{}, service: ServiceTIDAL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capabilities, ok := DescribeEnabledService(tt.config, tt.service)
			require.True(t, ok)
			assert.Equal(t, tt.supportsAlbumTarget, capabilities.SupportsAlbumTarget)
			assert.Equal(t, tt.supportsSongTarget, capabilities.SupportsSongTarget)
		})
	}
}

func TestSupportedServiceLists(t *testing.T) {
	assert.Equal(t, []ServiceName{
		ServiceAppleMusic,
		ServiceBandcamp,
		ServiceDeezer,
		ServiceSoundCloud,
		ServiceYouTubeMusic,
		ServiceSpotify,
		ServiceTIDAL,
	}, SupportedTargetServices())
	assert.Equal(t, []ServiceName{
		ServiceAppleMusic,
		ServiceBandcamp,
		ServiceDeezer,
		ServiceSoundCloud,
		ServiceSpotify,
		ServiceTIDAL,
	}, SupportedSongTargetServices())
}

const (
	unknownTargetServiceMessage     = "\"unknown\" (expected one of the supported target services: appleMusic, bandcamp, deezer, soundcloud, youtubeMusic, spotify, tidal)"
	amazonMusicTargetServiceMessage = "\"amazonMusic\" (expected one of the supported target services: appleMusic, bandcamp, deezer, soundcloud, youtubeMusic, spotify, tidal)"
	spotifyTargetServiceMessage     = "\"spotify\" (expected one of the supported target services: appleMusic, bandcamp, deezer, soundcloud, youtubeMusic, spotify, tidal)"
	youTubeMusicSongTargetMessage   = "\"youtubeMusic\" (enabled for songs: appleMusic, bandcamp, deezer, soundcloud)"
	amazonMusicSongTargetMessage    = "\"amazonMusic\" (enabled for songs: appleMusic, bandcamp, deezer, soundcloud)"
	tidalSongTargetMessage          = "\"tidal\" (enabled for songs: appleMusic, bandcamp, deezer, soundcloud)"
)

func TestProviderCatalogTargetServiceRequests(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		raw    string
		want   TargetServiceRequestDecision
	}{
		{name: "available alias", raw: "apple-music", want: TargetServiceRequestDecision{Service: ServiceAppleMusic, Status: TargetServiceRequestAvailable}},
		{name: "unknown", raw: "unknown", want: TargetServiceRequestDecision{Status: TargetServiceRequestUnknown, Message: unknownTargetServiceMessage}},
		{name: "parse only", raw: "amazonMusic", want: TargetServiceRequestDecision{Service: ServiceAmazonMusic, Status: TargetServiceRequestParseOnly, Message: amazonMusicTargetServiceMessage}},
		{name: "credentials required", raw: "spotify", want: TargetServiceRequestDecision{Service: ServiceSpotify, Status: TargetServiceRequestCredentialsRequired, Message: spotifyTargetServiceMessage}},
		{name: "credentials configured", config: Config{Spotify: SpotifyConfig{ClientID: "id", ClientSecret: "secret"}}, raw: "spotify", want: TargetServiceRequestDecision{Service: ServiceSpotify, Status: TargetServiceRequestAvailable}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, EvaluateTargetServiceRequest(tt.config, tt.raw))
		})
	}
}

func TestProviderCatalogSongTargetServiceRequests(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		service ServiceName
		want    TargetServiceRequestDecision
	}{
		{name: "available", service: ServiceAppleMusic, want: TargetServiceRequestDecision{Service: ServiceAppleMusic, Status: TargetServiceRequestAvailable}},
		{name: "unsupported song target", service: ServiceYouTubeMusic, want: TargetServiceRequestDecision{Service: ServiceYouTubeMusic, Status: TargetServiceRequestUnsupported, Message: youTubeMusicSongTargetMessage}},
		{name: "parse only", service: ServiceAmazonMusic, want: TargetServiceRequestDecision{Service: ServiceAmazonMusic, Status: TargetServiceRequestParseOnly, Message: amazonMusicSongTargetMessage}},
		{name: "credentials required", service: ServiceTIDAL, want: TargetServiceRequestDecision{Service: ServiceTIDAL, Status: TargetServiceRequestCredentialsRequired, Message: tidalSongTargetMessage}},
		{name: "credentials configured", config: Config{TIDAL: TIDALConfig{ClientID: "id", ClientSecret: "secret"}}, service: ServiceTIDAL, want: TargetServiceRequestDecision{Service: ServiceTIDAL, Status: TargetServiceRequestAvailable}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, EvaluateSongTargetService(tt.config, tt.service))
		})
	}
}

func TestEnabledServiceLists(t *testing.T) {
	assert.Equal(t, []ServiceName{
		ServiceAppleMusic,
		ServiceBandcamp,
		ServiceDeezer,
		ServiceSoundCloud,
		ServiceYouTubeMusic,
	}, EnabledTargetServices(Config{}))
	assert.Equal(t, []ServiceName{
		ServiceAppleMusic,
		ServiceBandcamp,
		ServiceDeezer,
		ServiceSoundCloud,
	}, EnabledSongTargetServices(Config{}))

	config := Config{
		Spotify: SpotifyConfig{ClientID: "id", ClientSecret: "secret"},
		TIDAL:   TIDALConfig{ClientID: "tidal-id", ClientSecret: "tidal-secret"},
	}
	assert.Equal(t, SupportedTargetServices(), EnabledTargetServices(config))
	assert.Equal(t, SupportedSongTargetServices(), EnabledSongTargetServices(config))
	assert.True(t, SupportsEnabledTarget(config, ServiceSpotify))
	assert.True(t, SupportsEnabledSongTarget(config, ServiceTIDAL))
	assert.False(t, SupportsEnabledTarget(Config{}, ServiceSpotify))
	assert.False(t, SupportsEnabledSongTarget(Config{}, ServiceTIDAL))
}

func TestNormalizedConfigDefaultsSongWeights(t *testing.T) {
	config := normalizedConfig(Config{})
	assert.NotEqual(t, SongScoreWeights{}, config.SongScoreWeights)
}

func TestMatchStrengthForScore(t *testing.T) {
	tests := []struct {
		score int
		want  MatchStrength
	}{
		{score: 120, want: MatchStrengthStrong},
		{score: 80, want: MatchStrengthProbable},
		{score: 50, want: MatchStrengthWeak},
		{score: 49, want: MatchStrengthVeryWeak},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, MatchStrengthForScore(tt.score))
	}
}

func TestNewWithAdaptersResolveAlbum(t *testing.T) {
	resolver := NewWithAdapters(
		[]SourceAdapter{newLibrarySourceAdapter()},
		[]TargetAdapter{newLibraryTargetAdapter()},
	)

	resolution, err := resolver.ResolveAlbum(context.Background(), testLibrarySourceURL)
	require.NoError(t, err)
	assert.Equal(t, ServiceDeezer, resolution.Source.Service)
	match := resolution.Matches[ServiceSpotify]
	require.NotNil(t, match.Best)
	assert.Equal(t, "spotify-1", match.Best.Candidate.CandidateID)
}

func TestNewWithEntityAdaptersResolveSong(t *testing.T) {
	resolver := newTestEntityResolver()

	resolution, err := resolver.ResolveSong(context.Background(), "https://fixture.test/songs/1")
	require.NoError(t, err)
	assert.Equal(t, ServiceSpotify, resolution.Source.Service)
	match := resolution.Matches[ServiceAppleMusic]
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
			var resolver *Resolver
			_, err := resolver.ResolveAlbum(context.Background(), testLibrarySourceURL)
			return err
		}},
		{name: "missing song resolver", resolve: func() error {
			resolver := &Resolver{}
			_, err := resolver.ResolveSong(context.Background(), "https://fixture.test/songs/1")
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.ErrorIs(t, tt.resolve(), ErrResolverNotInitialized)
		})
	}
}

func TestResolveReturnsPublicSentinelWhenCustomSourceViolatesContract(t *testing.T) {
	tests := []struct {
		name     string
		resolve  func() error
		sentinel error
	}{
		{name: "album source returns nil parsed url", sentinel: ErrSourceAdapterReturnedNilParsedURL, resolve: func() error {
			resolver := NewWithAdapters([]SourceAdapter{newNilParsedSourceAdapter()}, []TargetAdapter{newLibraryTargetAdapter()})
			_, err := resolver.ResolveAlbum(context.Background(), testLibrarySourceURL)
			return err
		}},
		{name: "album source returns nil album", sentinel: ErrSourceAdapterReturnedNilAlbum, resolve: func() error {
			resolver := NewWithAdapters([]SourceAdapter{newNilAlbumSourceAdapter()}, []TargetAdapter{newLibraryTargetAdapter()})
			_, err := resolver.ResolveAlbum(context.Background(), testLibrarySourceURL)
			return err
		}},
		{name: "song source returns nil song", sentinel: ErrSourceAdapterReturnedNilSong, resolve: func() error {
			resolver := NewWithEntityAdapters(nil, nil, []SongSourceAdapter{newNilSongSourceAdapter()}, []SongTargetAdapter{newLibrarySongTargetAdapter()})
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
	resolver := New(DefaultConfig())
	tests := []struct {
		name       string
		inputURL   string
		serviceErr error
	}{
		{
			name:       "youtube music",
			inputURL:   "https://music.youtube.com/watch?v=dQw4w9WgXcQ",
			serviceErr: ErrYouTubeMusicDeferred,
		},
		{
			name:       "amazon music",
			inputURL:   "https://music.amazon.com/albums/B0064UPU4G?trackAsin=B0064TRACK",
			serviceErr: ErrAmazonMusicDeferred,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, SupportsRuntimeSongInputURL(tt.inputURL))

			resolution, err := resolver.ResolveSong(context.Background(), tt.inputURL)
			require.Error(t, err)
			assert.Nil(t, resolution)
			assert.ErrorIs(t, err, ErrRuntimeDeferred)
			assert.ErrorIs(t, err, tt.serviceErr)
		})
	}
}

func TestResolveAlbumPreservesCustomTargetErrors(t *testing.T) {
	resolver := NewWithAdapters([]SourceAdapter{newLibrarySourceAdapter()}, []TargetAdapter{newFailingLibraryTargetAdapter()})

	resolution, err := resolver.ResolveAlbum(context.Background(), testLibrarySourceURL)
	require.NoError(t, err)
	require.NotNil(t, resolution)
	assert.ErrorIs(t, resolution.Matches[ServiceSpotify].Err, errLibraryTargetBoom)
}

func newTestEntityResolver() *Resolver {
	return NewWithEntityAdapters(
		[]SourceAdapter{newLibrarySourceAdapter()},
		[]TargetAdapter{newLibraryTargetAdapter()},
		[]SongSourceAdapter{newLibrarySongSourceAdapter()},
		[]SongTargetAdapter{newLibrarySongTargetAdapter()},
	)
}
