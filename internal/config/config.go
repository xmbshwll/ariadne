package config

import (
	"os"
	"strings"
	"time"

	"github.com/xmbshwll/ariadne/internal/httpx"
)

const defaultAppleMusicStorefront = "us"

type Config struct {
	Spotify           Spotify
	AppleMusic        AppleMusic
	TIDAL             TIDAL
	HTTPTimeout       time.Duration
	TargetServicesRaw string
}

type Spotify struct {
	ClientID     string
	ClientSecret string
}

func (s Spotify) Enabled() bool {
	return strings.TrimSpace(s.ClientID) != "" && strings.TrimSpace(s.ClientSecret) != ""
}

type AppleMusic struct {
	Storefront     string
	KeyID          string
	TeamID         string
	PrivateKeyPath string
}

func (a AppleMusic) AuthEnabled() bool {
	return strings.TrimSpace(a.KeyID) != "" && strings.TrimSpace(a.TeamID) != "" && strings.TrimSpace(a.PrivateKeyPath) != ""
}

type TIDAL struct {
	ClientID     string
	ClientSecret string
}

func (t TIDAL) Enabled() bool {
	return strings.TrimSpace(t.ClientID) != "" && strings.TrimSpace(t.ClientSecret) != ""
}

func Load() Config {
	return LoadFromLookup(os.Getenv)
}

func LoadFromEnv(getenv func(string) string) Config {
	return LoadFromLookup(getenv)
}

func LoadFromLookup(lookup func(string) string) Config {
	if lookup == nil {
		lookup = func(string) string { return "" }
	}

	trimmed := func(key string) string {
		return strings.TrimSpace(lookup(key))
	}

	return Config{
		Spotify: Spotify{
			ClientID:     trimmed("SPOTIFY_CLIENT_ID"),
			ClientSecret: trimmed("SPOTIFY_CLIENT_SECRET"),
		},
		AppleMusic: AppleMusic{
			Storefront:     NormalizeStorefront(trimmed("APPLE_MUSIC_STOREFRONT")),
			KeyID:          trimmed("APPLE_MUSIC_KEY_ID"),
			TeamID:         trimmed("APPLE_MUSIC_TEAM_ID"),
			PrivateKeyPath: trimmed("APPLE_MUSIC_PRIVATE_KEY_PATH"),
		},
		TIDAL: TIDAL{
			ClientID:     trimmed("TIDAL_CLIENT_ID"),
			ClientSecret: trimmed("TIDAL_CLIENT_SECRET"),
		},
		HTTPTimeout:       normalizedHTTPTimeout(trimmed("ARIADNE_HTTP_TIMEOUT")),
		TargetServicesRaw: trimmed("ARIADNE_TARGET_SERVICES"),
	}
}

// NormalizeStorefront trims and lower-cases a storefront, defaulting to the
// built-in "us". It is the one rule for storefronts, shared by the library's
// config normalization and the CLI's Catalog queries so the two cannot drift.
func NormalizeStorefront(value string) string {
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
