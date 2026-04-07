package config

import (
	"os"
	"strings"
	"time"

	"github.com/xmbshwll/ariadne/internal/httpx"
)

const defaultAppleMusicStorefront = "us"

type Config struct {
	Spotify     Spotify
	AppleMusic  AppleMusic
	TIDAL       TIDAL
	HTTPTimeout time.Duration
}

type Spotify struct {
	ClientID     string
	ClientSecret string
}

func (s Spotify) Enabled() bool {
	return s.ClientID != "" && s.ClientSecret != ""
}

type AppleMusic struct {
	Storefront     string
	KeyID          string
	TeamID         string
	PrivateKeyPath string
}

func (a AppleMusic) AuthEnabled() bool {
	return a.KeyID != "" && a.TeamID != "" && a.PrivateKeyPath != ""
}

type TIDAL struct {
	ClientID     string
	ClientSecret string
}

func (t TIDAL) Enabled() bool {
	return t.ClientID != "" && t.ClientSecret != ""
}

func Load() Config {
	return LoadFromEnv(os.Getenv)
}

func LoadFromEnv(getenv func(string) string) Config {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}

	return Config{
		Spotify: Spotify{
			ClientID:     strings.TrimSpace(getenv("SPOTIFY_CLIENT_ID")),
			ClientSecret: strings.TrimSpace(getenv("SPOTIFY_CLIENT_SECRET")),
		},
		AppleMusic: AppleMusic{
			Storefront:     normalizedStorefront(getenv("APPLE_MUSIC_STOREFRONT")),
			KeyID:          strings.TrimSpace(getenv("APPLE_MUSIC_KEY_ID")),
			TeamID:         strings.TrimSpace(getenv("APPLE_MUSIC_TEAM_ID")),
			PrivateKeyPath: strings.TrimSpace(getenv("APPLE_MUSIC_PRIVATE_KEY_PATH")),
		},
		TIDAL: TIDAL{
			ClientID:     strings.TrimSpace(getenv("TIDAL_CLIENT_ID")),
			ClientSecret: strings.TrimSpace(getenv("TIDAL_CLIENT_SECRET")),
		},
		HTTPTimeout: normalizedHTTPTimeout(getenv("ARIADNE_HTTP_TIMEOUT")),
	}
}

func normalizedStorefront(value string) string {
	storefront := strings.ToLower(strings.TrimSpace(value))
	if storefront == "" {
		return defaultAppleMusicStorefront
	}
	return storefront
}

func normalizedHTTPTimeout(value string) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return httpx.DefaultTimeout()
	}
	timeout, err := time.ParseDuration(value)
	if err != nil || timeout <= 0 {
		return httpx.DefaultTimeout()
	}
	return timeout
}
