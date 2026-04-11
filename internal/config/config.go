package config

import (
	"os"
	"strings"
	"time"

	"github.com/xmbshwll/ariadne/internal/httpx"
	"github.com/xmbshwll/ariadne/internal/model"
)

var targetServiceLookupNormalizer = strings.NewReplacer("-", "", "_", "")

var targetServicesByLookupKey = map[string]model.ServiceName{
	normalizeTargetServiceLookupKey(string(model.ServiceAppleMusic)):   model.ServiceAppleMusic,
	normalizeTargetServiceLookupKey("applemusic"):                      model.ServiceAppleMusic,
	normalizeTargetServiceLookupKey(string(model.ServiceBandcamp)):     model.ServiceBandcamp,
	normalizeTargetServiceLookupKey(string(model.ServiceDeezer)):       model.ServiceDeezer,
	normalizeTargetServiceLookupKey(string(model.ServiceSoundCloud)):   model.ServiceSoundCloud,
	normalizeTargetServiceLookupKey(string(model.ServiceSpotify)):      model.ServiceSpotify,
	normalizeTargetServiceLookupKey(string(model.ServiceTIDAL)):        model.ServiceTIDAL,
	normalizeTargetServiceLookupKey(string(model.ServiceYouTubeMusic)): model.ServiceYouTubeMusic,
	normalizeTargetServiceLookupKey("youtubemusic"):                    model.ServiceYouTubeMusic,
	normalizeTargetServiceLookupKey("ytmusic"):                         model.ServiceYouTubeMusic,
}

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

	return Config{
		Spotify: Spotify{
			ClientID:     strings.TrimSpace(lookup("SPOTIFY_CLIENT_ID")),
			ClientSecret: strings.TrimSpace(lookup("SPOTIFY_CLIENT_SECRET")),
		},
		AppleMusic: AppleMusic{
			Storefront:     normalizedStorefront(lookup("APPLE_MUSIC_STOREFRONT")),
			KeyID:          strings.TrimSpace(lookup("APPLE_MUSIC_KEY_ID")),
			TeamID:         strings.TrimSpace(lookup("APPLE_MUSIC_TEAM_ID")),
			PrivateKeyPath: strings.TrimSpace(lookup("APPLE_MUSIC_PRIVATE_KEY_PATH")),
		},
		TIDAL: TIDAL{
			ClientID:     strings.TrimSpace(lookup("TIDAL_CLIENT_ID")),
			ClientSecret: strings.TrimSpace(lookup("TIDAL_CLIENT_SECRET")),
		},
		HTTPTimeout:    normalizedHTTPTimeout(lookup("ARIADNE_HTTP_TIMEOUT")),
		TargetServices: normalizedTargetServices(lookup("ARIADNE_TARGET_SERVICES")),
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

	services := make([]model.ServiceName, 0)
	seen := make(map[model.ServiceName]struct{})
	for part := range strings.SplitSeq(value, ",") {
		service, ok := targetServicesByLookupKey[normalizeTargetServiceLookupKey(part)]
		if !ok {
			continue
		}
		if _, ok := seen[service]; ok {
			continue
		}
		seen[service] = struct{}{}
		services = append(services, service)
	}
	if len(services) == 0 {
		return nil
	}
	return services
}

func normalizeTargetServiceLookupKey(value string) string {
	return targetServiceLookupNormalizer.Replace(strings.ToLower(strings.TrimSpace(value)))
}
