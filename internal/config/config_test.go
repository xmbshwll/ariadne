package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadFromEnv(t *testing.T) {
	cfg := LoadFromEnv(func(key string) string {
		switch key {
		case "SPOTIFY_CLIENT_ID":
			return "  client-id  "
		case "SPOTIFY_CLIENT_SECRET":
			return " secret "
		case "APPLE_MUSIC_STOREFRONT":
			return " DE "
		case "APPLE_MUSIC_KEY_ID":
			return " music-key "
		case "APPLE_MUSIC_TEAM_ID":
			return " team-id "
		case "APPLE_MUSIC_PRIVATE_KEY_PATH":
			return " /tmp/AuthKey_ABC123.p8 "
		case "TIDAL_CLIENT_ID":
			return " tidal-client "
		case "TIDAL_CLIENT_SECRET":
			return " tidal-secret "
		default:
			return ""
		}
	})

	assert.Equal(t, "client-id", cfg.Spotify.ClientID)
	assert.Equal(t, "secret", cfg.Spotify.ClientSecret)
	assert.True(t, cfg.Spotify.Enabled())
	assert.Equal(t, "de", cfg.AppleMusic.Storefront)
	assert.Equal(t, "music-key", cfg.AppleMusic.KeyID)
	assert.Equal(t, "team-id", cfg.AppleMusic.TeamID)
	assert.Equal(t, "/tmp/AuthKey_ABC123.p8", cfg.AppleMusic.PrivateKeyPath)
	assert.True(t, cfg.AppleMusic.AuthEnabled())
	assert.Equal(t, "tidal-client", cfg.TIDAL.ClientID)
	assert.Equal(t, "tidal-secret", cfg.TIDAL.ClientSecret)
	assert.True(t, cfg.TIDAL.Enabled())
}

func TestLoadFromEnvDefaults(t *testing.T) {
	cfg := LoadFromEnv(nil)
	assert.False(t, cfg.Spotify.Enabled())
	assert.Equal(t, "us", cfg.AppleMusic.Storefront)
	assert.False(t, cfg.AppleMusic.AuthEnabled())
	assert.False(t, cfg.TIDAL.Enabled())
}
