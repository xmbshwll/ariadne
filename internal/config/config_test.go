package config_test

import (
	"testing"

	config "github.com/xmbshwll/ariadne/internal/config"

	"github.com/stretchr/testify/assert"
)

// TestLoadFromLookup pins the env parsing and trimming rules shared by every
// config entry point: values are trimmed, the storefront is normalized, and a
// nil lookup behaves like an empty environment.
func TestLoadFromLookup(t *testing.T) {
	env := map[string]string{
		"SPOTIFY_CLIENT_ID":            "  client-id  ",
		"SPOTIFY_CLIENT_SECRET":        " secret ",
		"APPLE_MUSIC_STOREFRONT":       " DE ",
		"APPLE_MUSIC_KEY_ID":           " music-key ",
		"APPLE_MUSIC_TEAM_ID":          " team-id ",
		"APPLE_MUSIC_PRIVATE_KEY_PATH": " /tmp/AuthKey_ABC123.p8 ",
		"TIDAL_CLIENT_ID":              " tidal-client ",
		"TIDAL_CLIENT_SECRET":          " tidal-secret ",
		"ARIADNE_TARGET_SERVICES":      " spotify, appleMusic , spotify ",
	}
	lookup := func(key string) string {
		if value, ok := env[key]; ok {
			return value
		}
		return ""
	}

	tests := []struct {
		name           string
		lookup         func(string) string
		wantSpotify    bool
		wantAppleAuth  bool
		wantTIDAL      bool
		wantStorefront string
	}{
		{
			name:           "values are trimmed and normalized",
			lookup:         lookup,
			wantSpotify:    true,
			wantAppleAuth:  true,
			wantTIDAL:      true,
			wantStorefront: "de",
		},
		{
			name:           "a nil lookup behaves like an empty environment",
			lookup:         nil,
			wantSpotify:    false,
			wantAppleAuth:  false,
			wantTIDAL:      false,
			wantStorefront: "us",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.LoadFromLookup(tt.lookup)

			assert.Equal(t, tt.wantSpotify, cfg.Spotify.Enabled(), tt.name)
			assert.Equal(t, tt.wantAppleAuth, cfg.AppleMusic.AuthEnabled(), tt.name)
			assert.Equal(t, tt.wantTIDAL, cfg.TIDAL.Enabled(), tt.name)
			assert.Equal(t, tt.wantStorefront, cfg.AppleMusic.Storefront, tt.name)
		})
	}

	cfg := config.LoadFromLookup(lookup)
	assert.Equal(t, "client-id", cfg.Spotify.ClientID)
	assert.Equal(t, "music-key", cfg.AppleMusic.KeyID)
	assert.Equal(t, "/tmp/AuthKey_ABC123.p8", cfg.AppleMusic.PrivateKeyPath)
	assert.Equal(t, "tidal-secret", cfg.TIDAL.ClientSecret)
	assert.Equal(t, "spotify, appleMusic , spotify", cfg.TargetServicesRaw)
}

// TestLoadReadsTheProcessEnvironment covers the os.Getenv entry point the CLI
// validation tools use. It runs serially (no t.Parallel) because it mutates
// the process environment.
func TestLoadReadsTheProcessEnvironment(t *testing.T) {
	t.Setenv("SPOTIFY_CLIENT_ID", "  client-id  ")
	t.Setenv("SPOTIFY_CLIENT_SECRET", " secret ")
	t.Setenv("APPLE_MUSIC_STOREFRONT", " GB ")
	t.Setenv("TIDAL_CLIENT_SECRET", "tidal-secret")
	t.Setenv("ARIADNE_TARGET_SERVICES", " spotify, appleMusic ")

	tests := []struct {
		name           string
		wantSpotify    bool
		wantTIDAL      bool
		wantStorefront string
	}{
		{name: "loads and trims the real environment", wantSpotify: true, wantTIDAL: false, wantStorefront: "gb"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Load()
			assert.Equal(t, tt.wantSpotify, cfg.Spotify.Enabled(), tt.name)
			assert.Equal(t, tt.wantTIDAL, cfg.TIDAL.Enabled(), tt.name)
			assert.Equal(t, "client-id", cfg.Spotify.ClientID, tt.name)
			assert.Equal(t, tt.wantStorefront, cfg.AppleMusic.Storefront, tt.name)
			assert.Equal(t, "spotify, appleMusic", cfg.TargetServicesRaw, tt.name)
		})
	}
}
