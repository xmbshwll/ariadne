package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne"
)

var errRootBoom = errors.New("boom")

func TestRootError(t *testing.T) {
	err := fmt.Errorf("outer: %w", fmt.Errorf("middle: %w", errRootBoom))
	assert.ErrorIs(t, rootError(err), errRootBoom)
}

func TestRun(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantErr     string
		wantStdout  []string
		wantStderr  []string
		avoidStdout []string
	}{
		{
			name: "help",
			args: []string{"help"},
			wantStdout: []string{
				"Usage:",
				"ariadne resolve [--log-level=debug] [--song|--album] [--verbose] [--format=json|yaml|csv] [--services=spotify,deezer] [--min-strength=probable] [--apple-music-storefront=us] [--resolution-timeout=20s] <url>",
				"<url>",
				"Values: a supported album URL from Apple Music, Deezer, Spotify, TIDAL",
				"URL from Apple Music, Bandcamp, Deezer, SoundCloud, Spotify, or TIDAL.",
				"YouTube Music and Amazon Music song URLs are recognized for parsing, but",
				"Behavior: when neither --song nor --album is set, Ariadne asks the library",
				"--song",
				"--album",
				"Commands:",
				"resolve  Resolve a supported album or song URL across services.",
				"--config",
				"--log-level",
				"Environment override: ARIADNE_LOG_LEVEL.",
				"Behavior: config file values are loaded first, environment variables override them, and explicit CLI flags override both.",
				"--verbose, -v",
				"--format",
				"--services",
				"--min-strength",
				"--apple-music-storefront",
				"--http-timeout",
				"--resolution-timeout",
				"Spotify target search is enabled only when SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET are set",
				"TIDAL source fetch and target search require TIDAL_CLIENT_ID and TIDAL_CLIENT_SECRET",
				"Amazon Music URLs are recognized for parsing, but runtime resolution remains deferred.",
			},
			avoidStdout: []string{"Global Flags:", "help for resolve", "configuration source (values:"},
		},
		{
			name:       "missing command",
			args:       nil,
			wantErr:    "missing command",
			wantStderr: []string{"Usage:"},
		},
		{
			name:       "unknown command",
			args:       []string{"unknown"},
			wantErr:    "unknown command: unknown",
			wantStderr: []string{"Usage:"},
		},
		{
			name:       "unknown command after config flag",
			args:       []string{"--config", ".env", "unknown"},
			wantErr:    "unknown command: unknown",
			wantStderr: []string{"Usage:"},
		},
		{
			name:        "resolve usage",
			args:        []string{"resolve"},
			wantErr:     "usage: ariadne resolve [--log-level=debug] [--song|--album] [--verbose] [--format=json|yaml|csv] [--services=spotify,deezer] [--min-strength=probable] [--apple-music-storefront=us] [--resolution-timeout=20s] <url>",
			avoidStdout: []string{"{"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			err := run(tt.args, &stdout, &stderr)
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}

			for _, want := range tt.wantStdout {
				assert.Contains(t, stdout.String(), want)
			}
			for _, want := range tt.wantStderr {
				assert.Contains(t, stderr.String(), want)
			}
			for _, avoid := range tt.avoidStdout {
				assert.NotContains(t, stdout.String(), avoid)
			}
		})
	}
}

func TestRunHelpIgnoresMalformedConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".env")
	require.NoError(t, os.WriteFile(configPath, []byte("ARIADNE_HTTP_TIMEOUT=not-a-duration\n"), 0o600))

	tests := []struct {
		name       string
		args       []string
		wantStdout []string
	}{
		{
			name:       "root help",
			args:       []string{"--config", configPath, "help"},
			wantStdout: []string{"Usage:"},
		},
		{
			name:       "subcommand help command",
			args:       []string{"--config", configPath, "help", "resolve"},
			wantStdout: []string{"Resolve a supported music URL across music services.", "--resolution-timeout"},
		},
		{
			name:       "subcommand help flag",
			args:       []string{"--config", configPath, "resolve", "--help"},
			wantStdout: []string{"Resolve a supported music URL across music services.", "--resolution-timeout"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			err := run(tt.args, &stdout, &stderr)
			require.NoError(t, err)
			for _, want := range tt.wantStdout {
				assert.Contains(t, stdout.String(), want)
			}
			assert.Empty(t, stderr.String())
		})
	}
}

func TestRunMissingCommandIgnoresMalformedConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".env")
	require.NoError(t, os.WriteFile(configPath, []byte("ARIADNE_HTTP_TIMEOUT=not-a-duration\n"), 0o600))

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run([]string{"--config", configPath}, &stdout, &stderr)
	require.Error(t, err)
	assert.ErrorIs(t, err, errMissingCommand)
	assert.Contains(t, stderr.String(), "Usage:")
	assert.Empty(t, stdout.String())
}

func TestRunHelpWithLogLevelBeforeCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run([]string{"--log-level", "debug", "help"}, &stdout, &stderr)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Usage:")
	assert.Empty(t, stderr.String())
}

func TestRunRejectsUnsupportedLogLevel(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run([]string{"--log-level", "trace", "resolve", "https://fixture.test/source"}, &stdout, &stderr)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unsupported log level "trace"`)
	assert.Empty(t, stdout.String())
	assert.Empty(t, stderr.String())
}

func TestRunResolveDebugLogIncludesSecretsFromConfig(t *testing.T) {
	withResolverFactory(t, func(_ ariadne.Config) *ariadne.Resolver {
		return ariadne.NewWithAdapters(
			[]ariadne.SourceAdapter{newFixtureSourceAdapterForCLI(map[string]ariadne.CanonicalAlbum{
				"https://fixture.test/source": {
					Service:           ariadne.ServiceDeezer,
					SourceID:          "src-1",
					SourceURL:         "https://fixture.test/source",
					Title:             "Fixture Album",
					NormalizedTitle:   "fixture album",
					Artists:           []string{"Fixture Artist"},
					NormalizedArtists: []string{"fixture artist"},
					ReleaseDate:       "2024-02-03",
					UPC:               "123456789012",
				},
			})},
			nil,
		)
	})

	configPath := filepath.Join(t.TempDir(), ".env")
	require.NoError(t, os.WriteFile(configPath, []byte("SPOTIFY_CLIENT_ID=debug-client\nSPOTIFY_CLIENT_SECRET=debug-secret\nAPPLE_MUSIC_PRIVATE_KEY_PATH=/tmp/debug-key.p8\n"), 0o600))

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run([]string{"--log-level", "debug", "--config", configPath, "resolve", "https://fixture.test/source"}, &stdout, &stderr)
	require.NoError(t, err)
	assert.Contains(t, stderr.String(), `DEBUG config file loaded path=`)
	assert.Contains(t, stderr.String(), `DEBUG effective config SPOTIFY_CLIENT_ID="debug-client" SPOTIFY_CLIENT_SECRET="debug-secret"`)
	assert.Contains(t, stderr.String(), `APPLE_MUSIC_PRIVATE_KEY_PATH="/tmp/debug-key.p8"`)
	assert.Contains(t, stderr.String(), `DEBUG resolve start mode=auto url="https://fixture.test/source"`)
	assert.Contains(t, stderr.String(), `DEBUG resolve complete mode=auto url="https://fixture.test/source"`)
	assert.NotEmpty(t, stdout.String())
}

func TestRunResolveInfoLogDoesNotPrintSecrets(t *testing.T) {
	withResolverFactory(t, func(_ ariadne.Config) *ariadne.Resolver {
		return ariadne.NewWithAdapters(
			[]ariadne.SourceAdapter{newFixtureSourceAdapterForCLI(map[string]ariadne.CanonicalAlbum{
				"https://fixture.test/source": {
					Service:   ariadne.ServiceDeezer,
					SourceID:  "src-1",
					SourceURL: "https://fixture.test/source",
					Title:     "Fixture Album",
				},
			})},
			nil,
		)
	})

	configPath := filepath.Join(t.TempDir(), ".env")
	require.NoError(t, os.WriteFile(configPath, []byte("SPOTIFY_CLIENT_SECRET=info-secret\n"), 0o600))

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run([]string{"--log-level", "info", "--config", configPath, "resolve", "https://fixture.test/source"}, &stdout, &stderr)
	require.NoError(t, err)
	assert.NotContains(t, stderr.String(), `info-secret`)
	assert.NotContains(t, stderr.String(), `DEBUG effective config`)
	assert.Empty(t, stderr.String())
	assert.NotEmpty(t, stdout.String())
}
