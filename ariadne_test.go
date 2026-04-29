package ariadne

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
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

func TestFromInternalServiceNamesPreservesNilVsEmpty(t *testing.T) {
	assert.Nil(t, fromInternalServiceNames(nil))
	assert.Equal(t, []ServiceName{}, fromInternalServiceNames([]model.ServiceName{}))
}

func TestDescribeService(t *testing.T) {
	spotify, ok := DescribeService(ServiceSpotify)
	require.True(t, ok)
	assert.Equal(t, []string{"spotify"}, spotify.Aliases)
	assert.True(t, spotify.SupportsAlbumSource)
	assert.True(t, spotify.SupportsAlbumTarget)
	assert.True(t, spotify.SupportsSongSource)
	assert.True(t, spotify.SupportsSongTarget)
	assert.True(t, spotify.SupportsRuntimeSongInputURL)

	youTubeMusic, ok := DescribeService(ServiceYouTubeMusic)
	require.True(t, ok)
	assert.True(t, youTubeMusic.SupportsAlbumSource)
	assert.True(t, youTubeMusic.SupportsAlbumTarget)
	assert.True(t, youTubeMusic.SupportsSongSource)
	assert.False(t, youTubeMusic.SupportsSongTarget)
	assert.True(t, youTubeMusic.SupportsRuntimeSongInputURL)

	amazon, ok := DescribeService(ServiceAmazonMusic)
	require.True(t, ok)
	assert.True(t, amazon.SupportsAlbumSource)
	assert.False(t, amazon.SupportsAlbumTarget)
	assert.True(t, amazon.SupportsSongSource)
	assert.False(t, amazon.SupportsSongTarget)
	assert.True(t, amazon.SupportsRuntimeSongInputURL)
}

func TestDescribeEnabledService(t *testing.T) {
	spotify, ok := DescribeEnabledService(Config{}, ServiceSpotify)
	require.True(t, ok)
	assert.False(t, spotify.SupportsAlbumTarget)
	assert.False(t, spotify.SupportsSongTarget)

	spotify, ok = DescribeEnabledService(Config{Spotify: SpotifyConfig{ClientID: "id", ClientSecret: "secret"}}, ServiceSpotify)
	require.True(t, ok)
	assert.True(t, spotify.SupportsAlbumTarget)
	assert.True(t, spotify.SupportsSongTarget)

	tidal, ok := DescribeEnabledService(Config{}, ServiceTIDAL)
	require.True(t, ok)
	assert.False(t, tidal.SupportsAlbumTarget)
	assert.False(t, tidal.SupportsSongTarget)
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

func TestResolveAlbumReturnsErrorForNilResolver(t *testing.T) {
	var resolver *Resolver

	resolution, err := resolver.ResolveAlbum(context.Background(), testLibrarySourceURL)
	require.Error(t, err)
	assert.Nil(t, resolution)
	assert.ErrorIs(t, err, ErrResolverNotInitialized)
}

func TestResolveSongReturnsErrorForMissingSongResolver(t *testing.T) {
	resolver := &Resolver{}

	resolution, err := resolver.ResolveSong(context.Background(), "https://fixture.test/songs/1")
	require.Error(t, err)
	assert.Nil(t, resolution)
	assert.ErrorIs(t, err, ErrResolverNotInitialized)
}

func TestResolveAlbumReturnsPublicSentinelWhenCustomSourceReturnsNilParsedURL(t *testing.T) {
	resolver := NewWithAdapters([]SourceAdapter{newNilParsedSourceAdapter()}, []TargetAdapter{newLibraryTargetAdapter()})

	resolution, err := resolver.ResolveAlbum(context.Background(), testLibrarySourceURL)
	require.Error(t, err)
	assert.Nil(t, resolution)
	assert.ErrorIs(t, err, ErrSourceAdapterReturnedNilParsedURL)
}

func TestResolveAlbumReturnsPublicSentinelWhenCustomSourceReturnsNilAlbum(t *testing.T) {
	resolver := NewWithAdapters([]SourceAdapter{newNilAlbumSourceAdapter()}, []TargetAdapter{newLibraryTargetAdapter()})

	resolution, err := resolver.ResolveAlbum(context.Background(), testLibrarySourceURL)
	require.Error(t, err)
	assert.Nil(t, resolution)
	assert.ErrorIs(t, err, ErrSourceAdapterReturnedNilAlbum)
}

func TestResolveSongReturnsPublicSentinelWhenCustomSourceReturnsNilSong(t *testing.T) {
	resolver := NewWithEntityAdapters(nil, nil, []SongSourceAdapter{newNilSongSourceAdapter()}, []SongTargetAdapter{newLibrarySongTargetAdapter()})

	resolution, err := resolver.ResolveSong(context.Background(), "https://fixture.test/songs/1")
	require.Error(t, err)
	assert.Nil(t, resolution)
	assert.ErrorIs(t, err, ErrSourceAdapterReturnedNilSong)
}

func TestResolveAlbumPreservesCustomTargetErrors(t *testing.T) {
	resolver := NewWithAdapters([]SourceAdapter{newLibrarySourceAdapter()}, []TargetAdapter{newFailingLibraryTargetAdapter()})

	resolution, err := resolver.ResolveAlbum(context.Background(), testLibrarySourceURL)
	require.Error(t, err)
	assert.Nil(t, resolution)
	assert.ErrorIs(t, err, errLibraryTargetBoom)
}

func newTestEntityResolver() *Resolver {
	return NewWithEntityAdapters(
		[]SourceAdapter{newLibrarySourceAdapter()},
		[]TargetAdapter{newLibraryTargetAdapter()},
		[]SongSourceAdapter{newLibrarySongSourceAdapter()},
		[]SongTargetAdapter{newLibrarySongTargetAdapter()},
	)
}
