package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
				"Amazon Music URLs and YouTube Music song URLs are recognized for parsing,",
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
				"spotify target search requires credentials: SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET must be set.",
				"tidal target search requires credentials: TIDAL_CLIENT_ID and TIDAL_CLIENT_SECRET must be set.",
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

func TestRunHelpRendersConfigDefaultAndTargetServiceValues(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run([]string{"help"}, &stdout, &stderr)

	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "    Default: .env")
	assert.Contains(t, stdout.String(), "    Values: comma-separated list drawn from appleMusic, applemusic, bandcamp, deezer, soundcloud, youtubeMusic, youtubemusic, ytmusic, spotify, tidal.")
	assert.Empty(t, stderr.String())
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
	withFixtureResolver(t, fixtureResolverForCLI{
		album: fixtureAlbumResolution(fixtureSourceAlbum, nil),
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
	withFixtureResolver(t, fixtureResolverForCLI{
		album: fixtureAlbumResolution(fixtureSourceAlbum, nil),
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

// White-box: the help functions assemble the resolve help text; the fixtures
// are catalog facts, asserted here so the rendered help cannot silently drift
// from what the Provider Catalog decides.

func TestHelpServiceCaveats(t *testing.T) {
	caveats := helpServiceCaveats()

	tests := []struct {
		name        string
		wantSub     string
		wantMissing string
	}{
		{
			name:    "spotify names its missing Credential Token",
			wantSub: "spotify target search requires credentials: SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET must be set.",
		},
		{
			name:    "tidal names its missing Credential Token",
			wantSub: "tidal target search requires credentials: TIDAL_CLIENT_ID and TIDAL_CLIENT_SECRET must be set.",
		},
		{
			name:        "services without caveats are absent",
			wantMissing: "appleMusic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantSub != "" {
				assert.Contains(t, caveats, tt.wantSub, tt.name)
			}
			if tt.wantMissing != "" {
				assert.NotContains(t, caveats, tt.wantMissing, tt.name)
			}
		})
	}
}

func TestHelpSongHydrationNote(t *testing.T) {
	note := helpSongHydrationNote()

	tests := []struct {
		name    string
		wantSub string
		missing []string
	}{
		{
			name:    "lists the catalog's song targets",
			wantSub: "  - Song resolution currently hydrates appleMusic, bandcamp, deezer, soundcloud, spotify, tidal.",
		},
		{
			name:    "omits services without song hydration",
			missing: []string{"amazonMusic", "youtubeMusic"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantSub != "" {
				assert.Contains(t, note, tt.wantSub, tt.name)
			}
			for _, absent := range tt.missing {
				assert.NotContains(t, note, " "+absent+",", tt.name)
				assert.NotContains(t, note, " "+absent+".", tt.name)
			}
		})
	}
}

func TestHelpServiceNotes(t *testing.T) {
	notes := strings.Join(helpServiceNotes(), "\n")

	tests := []struct {
		name       string
		wantSub    string
		wantAbsent string
	}{
		{
			name:    "defers youtube music hydration",
			wantSub: "  - youtubeMusic song URLs are recognized, but runtime hydration remains deferred.",
		},
		{
			name:    "defers amazon music hydration",
			wantSub: "  - amazonMusic song URLs are recognized, but runtime hydration remains deferred.",
		},
		{
			name:       "does not defer a song target",
			wantAbsent: "spotify song URLs are recognized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantSub != "" {
				assert.Contains(t, notes, tt.wantSub, tt.name)
			}
			if tt.wantAbsent != "" {
				assert.NotContains(t, notes, tt.wantAbsent, tt.name)
			}
		})
	}
}

// TestCurrentVersionPinsTheShapeOfTheVersionLine checks what the build info
// can and cannot know: module versions when built from published modules,
// "devel" placeholders when built from a checkout.
func TestCurrentVersionPinsTheShapeOfTheVersionLine(t *testing.T) {
	version := currentVersion()

	tests := []struct {
		name    string
		got     string
		wantSet bool
	}{
		{name: "cli version is set", got: version.CLI, wantSet: true},
		{name: "library version is set", got: version.Library, wantSet: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NotEmpty(t, tt.got, tt.name)
		})
	}

	// Inside `go test`, the main module version is "(devel)", so the CLI side
	// reports the devel placeholder; the library side may carry a real version
	// from the workspace.
	t.Run("the rendered line names both modules", func(t *testing.T) {
		line := version.String()
		assert.Contains(t, line, "ariadne CLI", line)
		assert.Contains(t, line, "library", line)
	})
}

// TestRunVersionFlagRendersTheVersionLine pins the --version flag and the
// version subcommand cobra derives from Version.
func TestRunVersionFlagRendersTheVersionLine(t *testing.T) {
	// Only the flag: cobra derives no `version` subcommand from cmd.Version,
	// and inventing one would duplicate what --version prints.
	tests := []struct {
		name string
		args []string
	}{
		{name: "--version flag", args: []string{"--version"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := run(tt.args, &stdout, &stderr)
			require.NoError(t, err, tt.name)

			assert.Contains(t, stdout.String(), "ariadne CLI", tt.name)
			assert.Contains(t, stdout.String(), "library", tt.name)
			assert.False(t, strings.Contains(stdout.String(), "%!"), tt.name)
		})
	}
}
