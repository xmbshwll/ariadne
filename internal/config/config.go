package config

import (
	"os"
	"strings"
	"time"

	"github.com/xmbshwll/ariadne/internal/httpx"
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/services"
)

const defaultAppleMusicStorefront = "us"

type Config struct {
	Spotify        Spotify
	AppleMusic     AppleMusic
	TIDAL          TIDAL
	HTTPTimeout    time.Duration
	TargetServices []model.ServiceName
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
			Storefront:     normalizedStorefront(trimmed("APPLE_MUSIC_STOREFRONT")),
			KeyID:          trimmed("APPLE_MUSIC_KEY_ID"),
			TeamID:         trimmed("APPLE_MUSIC_TEAM_ID"),
			PrivateKeyPath: trimmed("APPLE_MUSIC_PRIVATE_KEY_PATH"),
		},
		TIDAL: TIDAL{
			ClientID:     trimmed("TIDAL_CLIENT_ID"),
			ClientSecret: trimmed("TIDAL_CLIENT_SECRET"),
		},
		HTTPTimeout:    normalizedHTTPTimeout(trimmed("ARIADNE_HTTP_TIMEOUT")),
		TargetServices: normalizedTargetServices(trimmed("ARIADNE_TARGET_SERVICES")),
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

func normalizedTargetServices(value string) []model.ServiceName {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	resolved := make([]model.ServiceName, 0)
	seen := make(map[model.ServiceName]struct{})
	for part := range strings.SplitSeq(value, ",") {
		service, ok := services.LookupTarget(part)
		if !ok {
			continue
		}
		if _, ok := seen[service]; ok {
			continue
		}
		seen[service] = struct{}{}
		resolved = append(resolved, service)
	}
	if len(resolved) == 0 {
		return nil
	}
	return resolved
}
