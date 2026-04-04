package config

import "testing"

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

	if cfg.Spotify.ClientID != "client-id" {
		t.Fatalf("spotify client id = %q", cfg.Spotify.ClientID)
	}
	if cfg.Spotify.ClientSecret != "secret" {
		t.Fatalf("spotify client secret = %q", cfg.Spotify.ClientSecret)
	}
	if !cfg.Spotify.Enabled() {
		t.Fatalf("expected spotify config to be enabled")
	}
	if cfg.AppleMusic.Storefront != "de" {
		t.Fatalf("apple storefront = %q", cfg.AppleMusic.Storefront)
	}
	if cfg.AppleMusic.KeyID != "music-key" {
		t.Fatalf("apple key id = %q", cfg.AppleMusic.KeyID)
	}
	if cfg.AppleMusic.TeamID != "team-id" {
		t.Fatalf("apple team id = %q", cfg.AppleMusic.TeamID)
	}
	if cfg.AppleMusic.PrivateKeyPath != "/tmp/AuthKey_ABC123.p8" {
		t.Fatalf("apple private key path = %q", cfg.AppleMusic.PrivateKeyPath)
	}
	if !cfg.AppleMusic.AuthEnabled() {
		t.Fatalf("expected apple music auth to be enabled")
	}
	if cfg.TIDAL.ClientID != "tidal-client" {
		t.Fatalf("tidal client id = %q", cfg.TIDAL.ClientID)
	}
	if cfg.TIDAL.ClientSecret != "tidal-secret" {
		t.Fatalf("tidal client secret = %q", cfg.TIDAL.ClientSecret)
	}
	if !cfg.TIDAL.Enabled() {
		t.Fatalf("expected tidal config to be enabled")
	}
}

func TestLoadFromEnvDefaults(t *testing.T) {
	cfg := LoadFromEnv(nil)
	if cfg.Spotify.Enabled() {
		t.Fatalf("expected spotify config to be disabled")
	}
	if cfg.AppleMusic.Storefront != "us" {
		t.Fatalf("apple storefront = %q, want us", cfg.AppleMusic.Storefront)
	}
	if cfg.AppleMusic.AuthEnabled() {
		t.Fatalf("expected apple music auth to be disabled")
	}
	if cfg.TIDAL.Enabled() {
		t.Fatalf("expected tidal config to be disabled")
	}
}
