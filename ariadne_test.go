package ariadne_test

import (
	"context"
	"errors"
	"testing"
	"time"

	ariadne "github.com/xmbshwll/ariadne"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/adapters"
	"github.com/xmbshwll/ariadne/internal/resolve"
	"github.com/xmbshwll/ariadne/internal/score"
	"github.com/xmbshwll/ariadne/internal/wiring"
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
	albumWeights, songWeights := ariadne.ConfigWeights(config)
	assert.NotEqual(t, score.Weights{}, albumWeights)
	assert.NotEqual(t, score.SongWeights{}, songWeights)
	assert.Equal(t, 15*time.Second, config.HTTPTimeout)
}

func TestNormalizedConfigDefaultsSongWeights(t *testing.T) {
	_, songWeights := ariadne.ConfigWeights(ariadne.NormalizedConfig(ariadne.Config{}))
	assert.NotEqual(t, score.SongWeights{}, songWeights)
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
	resolver := ariadne.NewWithAdapters(ariadne.TestAdapters{
		AlbumSources: []adapters.Adapter{newLibrarySourceAdapter()},
		AlbumTargets: []adapters.Adapter{newLibraryTargetAdapter()},
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

func TestResolveSurfacesAdapterContractViolations(t *testing.T) {
	tests := []struct {
		name     string
		resolve  func() error
		sentinel error
	}{
		{name: "album source returns nil parsed url", sentinel: resolve.ErrSourceAdapterReturnedNilParsedURL, resolve: func() error {
			resolver := ariadne.NewWithAdapters(ariadne.TestAdapters{AlbumSources: []adapters.Adapter{newNilParsedSourceAdapter()}, AlbumTargets: []adapters.Adapter{newLibraryTargetAdapter()}})
			_, err := resolver.ResolveAlbum(context.Background(), testLibrarySourceURL)
			return err
		}},
		{name: "album source returns nil album", sentinel: resolve.ErrSourceAdapterReturnedNilAlbum, resolve: func() error {
			resolver := ariadne.NewWithAdapters(ariadne.TestAdapters{AlbumSources: []adapters.Adapter{newNilAlbumSourceAdapter()}, AlbumTargets: []adapters.Adapter{newLibraryTargetAdapter()}})
			_, err := resolver.ResolveAlbum(context.Background(), testLibrarySourceURL)
			return err
		}},
		{name: "song source returns nil song", sentinel: resolve.ErrSourceAdapterReturnedNilSong, resolve: func() error {
			resolver := ariadne.NewWithAdapters(ariadne.TestAdapters{SongSources: []adapters.Adapter{newNilSongSourceAdapter()}, SongTargets: []adapters.Adapter{newLibrarySongTargetAdapter()}})
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
			assert.True(t, wiring.Default.SupportsRuntimeSongInputURL(tt.inputURL))

			resolution, err := resolver.ResolveSong(context.Background(), tt.inputURL)
			require.Error(t, err)
			assert.Nil(t, resolution)
			assert.ErrorIs(t, err, ariadne.ErrRuntimeDeferred)
			assert.ErrorIs(t, err, tt.serviceErr)
		})
	}
}

func TestResolveAlbumPreservesCustomTargetErrors(t *testing.T) {
	resolver := ariadne.NewWithAdapters(ariadne.TestAdapters{AlbumSources: []adapters.Adapter{newLibrarySourceAdapter()}, AlbumTargets: []adapters.Adapter{newFailingLibraryTargetAdapter()}})

	resolution, err := resolver.ResolveAlbum(context.Background(), testLibrarySourceURL)
	require.NoError(t, err)
	require.NotNil(t, resolution)
	assert.ErrorIs(t, resolution.Matches[ariadne.ServiceSpotify].Err, errLibraryTargetBoom)
}

func newTestEntityResolver() *ariadne.Resolver {
	return ariadne.NewWithAdapters(ariadne.TestAdapters{
		AlbumSources: []adapters.Adapter{newLibrarySourceAdapter()},
		AlbumTargets: []adapters.Adapter{newLibraryTargetAdapter()},
		SongSources:  []adapters.Adapter{newLibrarySongSourceAdapter()},
		SongTargets:  []adapters.Adapter{newLibrarySongTargetAdapter()},
	})
}
