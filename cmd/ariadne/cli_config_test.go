package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne"
)

var (
	errUnknownOops     = errors.New("unknown command \"oops\" for \"ariadne\"")
	errDifferentCLIArg = errors.New("different error")
)

func TestArgsWithoutNamedFlagsConsumesExplicitEmptyValue(t *testing.T) {
	args := []string{"--config", "", "resolve", "https://fixture.test/source"}
	assert.Equal(t, []string{"resolve", "https://fixture.test/source"}, argsWithoutNamedFlags(args, "--config"))
}

func TestConfigPathFromArgsPreservesExplicitEmptyValue(t *testing.T) {
	assert.Equal(t, "", configPathFromArgs([]string{"--config", "", "resolve", "https://fixture.test/source"}))
	assert.Equal(t, "", configPathFromArgs([]string{"--config=", "resolve", "https://fixture.test/source"}))
}

func TestIsUnknownCommandError(t *testing.T) {
	assert.False(t, isUnknownCommandError(nil))
	assert.True(t, isUnknownCommandError(errUnknownOops))
	assert.False(t, isUnknownCommandError(errDifferentCLIArg))
}

func TestParseRequestedServicesSkipsEmptySegments(t *testing.T) {
	services, err := parseRequestedServices(" deezer, ,bandcamp,, ", ariadne.Config{})
	require.NoError(t, err)
	assert.Equal(t, []ariadne.ServiceName{ariadne.ServiceDeezer, ariadne.ServiceBandcamp}, services)
}

func TestParseMatchStrengthNormalizesAliases(t *testing.T) {
	for _, raw := range []string{"very_weak", "very-weak", "veryweak", " VERY_WEAK "} {
		strength, err := parseMatchStrength(raw)
		require.NoError(t, err)
		assert.Equal(t, ariadne.MatchStrengthVeryWeak, strength)
	}
}

func TestLoadCLIConfigFromDotEnv(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".env")
	content := strings.Join([]string{
		"SPOTIFY_CLIENT_ID=spotify-client",
		"SPOTIFY_CLIENT_SECRET=spotify-secret",
		"APPLE_MUSIC_STOREFRONT=gb",
		"APPLE_MUSIC_KEY_ID=apple-key",
		"APPLE_MUSIC_TEAM_ID=apple-team",
		"APPLE_MUSIC_PRIVATE_KEY_PATH=/tmp/AuthKey_TEST.p8",
		"TIDAL_CLIENT_ID=tidal-client",
		"TIDAL_CLIENT_SECRET=tidal-secret",
		"ARIADNE_HTTP_TIMEOUT=45s",
		"ARIADNE_TARGET_SERVICES=spotify,appleMusic,spotify",
	}, "\n")
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o644))

	cfg, err := loadCLIConfigWithLogger(configPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "spotify-client", cfg.Spotify.ClientID)
	assert.Equal(t, "spotify-secret", cfg.Spotify.ClientSecret)
	assert.Equal(t, "gb", cfg.AppleMusicStorefront)
	assert.Equal(t, "apple-key", cfg.AppleMusic.KeyID)
	assert.Equal(t, "apple-team", cfg.AppleMusic.TeamID)
	assert.Equal(t, "/tmp/AuthKey_TEST.p8", cfg.AppleMusic.PrivateKeyPath)
	assert.Equal(t, "tidal-client", cfg.TIDAL.ClientID)
	assert.Equal(t, "tidal-secret", cfg.TIDAL.ClientSecret)
	assert.Equal(t, 45*time.Second, cfg.HTTPTimeout)
	assert.Equal(t, []ariadne.ServiceName{ariadne.ServiceSpotify, ariadne.ServiceAppleMusic}, cfg.TargetServices)
}

func TestLoadCLIConfigEnvironmentOverridesFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(configPath, []byte("APPLE_MUSIC_STOREFRONT=gb\nSPOTIFY_CLIENT_ID=file-client\n"), 0o644))
	t.Setenv("APPLE_MUSIC_STOREFRONT", "de")
	t.Setenv("SPOTIFY_CLIENT_ID", "env-client")
	t.Setenv("ARIADNE_HTTP_TIMEOUT", "30s")

	cfg, err := loadCLIConfigWithLogger(configPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "de", cfg.AppleMusicStorefront)
	assert.Equal(t, "env-client", cfg.Spotify.ClientID)
	assert.Equal(t, 30*time.Second, cfg.HTTPTimeout)
}

func TestParseResolveArgsPreservesConfiguredTargetServices(t *testing.T) {
	resolveConfig, err := parseResolveArgs(
		[]string{"https://www.deezer.com/album/12047952"},
		ariadne.Config{
			Spotify:        ariadne.SpotifyConfig{ClientID: "client-id", ClientSecret: "client-secret"},
			TargetServices: []ariadne.ServiceName{ariadne.ServiceSpotify, ariadne.ServiceAppleMusic},
		},
	)
	require.NoError(t, err)
	assert.Equal(t, []ariadne.ServiceName{ariadne.ServiceSpotify, ariadne.ServiceAppleMusic}, resolveConfig.resolverConfig.TargetServices)
}

func TestParseResolveArgsValidatesConfiguredTargetServices(t *testing.T) {
	_, err := parseResolveArgs(
		[]string{"https://www.deezer.com/album/12047952"},
		ariadne.Config{TargetServices: []ariadne.ServiceName{ariadne.ServiceSpotify}},
	)
	require.ErrorIs(t, err, errSpotifyTargetCredentials)
}

func TestLoadCLIConfigRejectsNonPositiveHTTPTimeout(t *testing.T) {
	tests := []struct {
		name        string
		timeout     string
		wantMessage string
	}{
		{name: "zero", timeout: "0s", wantMessage: "invalid ARIADNE_HTTP_TIMEOUT \"0s\": ARIADNE_HTTP_TIMEOUT must be positive"},
		{name: "negative", timeout: "-5s", wantMessage: "invalid ARIADNE_HTTP_TIMEOUT \"-5s\": ARIADNE_HTTP_TIMEOUT must be positive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			configPath := filepath.Join(dir, ".env")
			require.NoError(t, os.WriteFile(configPath, []byte("ARIADNE_HTTP_TIMEOUT="+tt.timeout+"\n"), 0o644))

			_, err := loadCLIConfigWithLogger(configPath, nil)
			require.Error(t, err)
			assert.EqualError(t, err, tt.wantMessage)
		})
	}
}

func TestParseResolveArgs(t *testing.T) {
	baseConfig := ariadne.LoadConfigFromEnv(func(key string) string {
		switch key {
		case "APPLE_MUSIC_STOREFRONT":
			return "de"
		default:
			return ""
		}
	})

	tests := []struct {
		name                  string
		args                  []string
		wantURL               string
		wantStorefront        string
		wantFormat            string
		wantMinStrength       ariadne.MatchStrength
		wantServices          []ariadne.ServiceName
		wantHTTPTimeout       time.Duration
		wantResolutionTimeout time.Duration
		wantVerbose           bool
		wantForceSong         bool
		wantForceAlbum        bool
		wantErrContains       string
	}{
		{
			name:            "uses base config storefront",
			args:            []string{"https://www.deezer.com/album/12047952"},
			wantURL:         "https://www.deezer.com/album/12047952",
			wantStorefront:  "de",
			wantFormat:      "json",
			wantMinStrength: ariadne.MatchStrengthVeryWeak,
		},
		{
			name:            "verbose flag",
			args:            []string{"--verbose", "https://www.deezer.com/album/12047952"},
			wantURL:         "https://www.deezer.com/album/12047952",
			wantStorefront:  "de",
			wantFormat:      "json",
			wantMinStrength: ariadne.MatchStrengthVeryWeak,
			wantVerbose:     true,
		},
		{
			name:            "yaml format",
			args:            []string{"--format=yaml", "https://www.deezer.com/album/12047952"},
			wantURL:         "https://www.deezer.com/album/12047952",
			wantStorefront:  "de",
			wantFormat:      "yaml",
			wantMinStrength: ariadne.MatchStrengthVeryWeak,
		},
		{
			name:            "service filter",
			args:            []string{"--services=deezer,bandcamp", "https://www.deezer.com/album/12047952"},
			wantURL:         "https://www.deezer.com/album/12047952",
			wantStorefront:  "de",
			wantFormat:      "json",
			wantMinStrength: ariadne.MatchStrengthVeryWeak,
			wantServices:    []ariadne.ServiceName{ariadne.ServiceDeezer, ariadne.ServiceBandcamp},
		},
		{
			name:            "service filter aliases",
			args:            []string{"--services=apple-music,yt_music", "https://www.deezer.com/album/12047952"},
			wantURL:         "https://www.deezer.com/album/12047952",
			wantStorefront:  "de",
			wantFormat:      "json",
			wantMinStrength: ariadne.MatchStrengthVeryWeak,
			wantServices:    []ariadne.ServiceName{ariadne.ServiceAppleMusic, ariadne.ServiceYouTubeMusic},
		},
		{
			name:            "flag overrides env storefront",
			args:            []string{"--apple-music-storefront=gb", "https://www.deezer.com/album/12047952"},
			wantURL:         "https://www.deezer.com/album/12047952",
			wantStorefront:  "gb",
			wantFormat:      "json",
			wantMinStrength: ariadne.MatchStrengthVeryWeak,
		},
		{
			name:            "missing url",
			args:            []string{"--apple-music-storefront=gb"},
			wantErrContains: "usage: ariadne resolve [--log-level=debug] [--song|--album] [--verbose] [--format=json|yaml|csv] [--services=spotify,deezer] [--min-strength=probable] [--apple-music-storefront=us] [--resolution-timeout=20s] <url>",
		},
		{
			name:            "force song",
			args:            []string{"--song", "https://open.spotify.com/track/123"},
			wantURL:         "https://open.spotify.com/track/123",
			wantStorefront:  "de",
			wantFormat:      "json",
			wantMinStrength: ariadne.MatchStrengthVeryWeak,
			wantForceSong:   true,
		},
		{
			name:            "force album",
			args:            []string{"--album", "https://www.deezer.com/album/12047952"},
			wantURL:         "https://www.deezer.com/album/12047952",
			wantStorefront:  "de",
			wantFormat:      "json",
			wantMinStrength: ariadne.MatchStrengthVeryWeak,
			wantForceAlbum:  true,
		},
		{
			name:            "conflicting entity flags",
			args:            []string{"--song", "--album", "https://open.spotify.com/track/123"},
			wantErrContains: "--song and --album are mutually exclusive",
		},
		{
			name:            "unsupported service",
			args:            []string{"--services=amazonMusic", "https://www.deezer.com/album/12047952"},
			wantErrContains: "amazonMusic is not available as a target service",
		},
		{
			name:            "unsupported song target service",
			args:            []string{"--song", "--services=youtubeMusic", "https://open.spotify.com/track/123"},
			wantErrContains: "target service is not available for song resolution \"youtubeMusic\"",
		},
		{
			name:            "unsupported auto song target service",
			args:            []string{"--services=youtubeMusic", "https://open.spotify.com/track/123"},
			wantErrContains: "target service is not available for song resolution \"youtubeMusic\"",
		},
		{
			name:            "min strength",
			args:            []string{"--min-strength=probable", "https://www.deezer.com/album/12047952"},
			wantURL:         "https://www.deezer.com/album/12047952",
			wantStorefront:  "de",
			wantFormat:      "json",
			wantMinStrength: ariadne.MatchStrengthProbable,
		},
		{
			name:            "http timeout flag",
			args:            []string{"--http-timeout=45s", "https://www.deezer.com/album/12047952"},
			wantURL:         "https://www.deezer.com/album/12047952",
			wantStorefront:  "de",
			wantFormat:      "json",
			wantMinStrength: ariadne.MatchStrengthVeryWeak,
			wantHTTPTimeout: 45 * time.Second,
		},
		{
			name:                  "resolution timeout flag",
			args:                  []string{"--resolution-timeout=45s", "https://www.deezer.com/album/12047952"},
			wantURL:               "https://www.deezer.com/album/12047952",
			wantStorefront:        "de",
			wantFormat:            "json",
			wantMinStrength:       ariadne.MatchStrengthVeryWeak,
			wantResolutionTimeout: 45 * time.Second,
		},
		{
			name:            "invalid format",
			args:            []string{"--format=xml", "https://www.deezer.com/album/12047952"},
			wantErrContains: "unsupported format \"xml\"",
		},
		{
			name:            "invalid min strength",
			args:            []string{"--min-strength=excellent", "https://www.deezer.com/album/12047952"},
			wantErrContains: "unsupported min-strength \"excellent\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolveConfig, err := parseResolveArgs(tt.args, baseConfig)
			if tt.wantErrContains != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrContains)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantURL, resolveConfig.inputURL)
			assert.Equal(t, tt.wantStorefront, resolveConfig.resolverConfig.AppleMusicStorefront)
			assert.Equal(t, tt.wantFormat, resolveConfig.format)
			assert.Equal(t, tt.wantMinStrength, resolveConfig.minStrength)
			if tt.wantMinStrength == "" {
				assert.Equal(t, ariadne.MatchStrengthVeryWeak, resolveConfig.minStrength)
			}
			wantHTTPTimeout := tt.wantHTTPTimeout
			if wantHTTPTimeout == 0 {
				wantHTTPTimeout = 15 * time.Second
			}
			assert.Equal(t, wantHTTPTimeout, resolveConfig.resolverConfig.HTTPTimeout)
			wantResolutionTimeout := tt.wantResolutionTimeout
			if wantResolutionTimeout == 0 {
				wantResolutionTimeout = defaultResolveTimeout
			}
			assert.Equal(t, wantResolutionTimeout, resolveConfig.resolutionTimeout)
			assert.Len(t, resolveConfig.resolverConfig.TargetServices, len(tt.wantServices))
			for i, service := range tt.wantServices {
				assert.Equal(t, service, resolveConfig.resolverConfig.TargetServices[i])
			}
			assert.Equal(t, tt.wantVerbose, resolveConfig.verbose)
			assert.Equal(t, tt.wantForceSong, resolveConfig.forceSong)
			assert.Equal(t, tt.wantForceAlbum, resolveConfig.forceAlbum)
		})
	}
}
