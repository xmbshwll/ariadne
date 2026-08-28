package ariadne

import (
	"strings"
	"time"

	internalconfig "github.com/xmbshwll/ariadne/internal/config"
	"github.com/xmbshwll/ariadne/internal/httpx"
	"github.com/xmbshwll/ariadne/internal/score"
	"github.com/xmbshwll/ariadne/internal/wiring"
)

// Config configures the default library resolver.
type Config struct {
	// Spotify holds Spotify credentials for official source and target access.
	Spotify SpotifyConfig
	// AppleMusic holds Apple Music developer token configuration.
	AppleMusic AppleMusicConfig
	// TIDAL holds TIDAL credentials for official source and target access.
	TIDAL TIDALConfig
	// AppleMusicStorefront is the default storefront used for Apple Music operations.
	AppleMusicStorefront string
	// HTTPTimeout is the per-request timeout used by the default HTTP client.
	// When zero or negative, Ariadne uses its built-in default.
	HTTPTimeout time.Duration
	// TargetServices limits the default resolver to the listed target services.
	// When empty, Ariadne uses all available default targets.
	TargetServices []ServiceName
	// scoreWeights and songScoreWeights are the built-in Scoring signals. They
	// stay internal: ranking is Ariadne's decision, not a caller knob.
	scoreWeights     score.Weights
	songScoreWeights score.SongWeights
}

// SpotifyConfig holds Spotify app credentials used for target search and preferred source fetches.
type SpotifyConfig struct {
	// ClientID is the Spotify application client ID.
	ClientID string
	// ClientSecret is the Spotify application client secret.
	ClientSecret string
}

// AppleMusicConfig holds Apple Music key material used to generate MusicKit developer tokens.
type AppleMusicConfig struct {
	// KeyID is the Apple Music private key identifier.
	KeyID string
	// TeamID is the Apple Developer team identifier.
	TeamID string
	// PrivateKeyPath is the path to the Apple Music .p8 signing key.
	PrivateKeyPath string
}

// TIDALConfig holds TIDAL client credentials used for official catalog access.
type TIDALConfig struct {
	// ClientID is the TIDAL application client ID.
	ClientID string
	// ClientSecret is the TIDAL application client secret.
	ClientSecret string
}

// SpotifyEnabled reports whether Spotify credential-gated features are available.
// The check is the internal config rule, so the public answer can never disagree
// with what the Provider Catalog enables.
func (c Config) SpotifyEnabled() bool {
	return internalConfig(c).Spotify.Enabled()
}

// TIDALEnabled reports whether TIDAL credential-gated features are available.
func (c Config) TIDALEnabled() bool {
	return internalConfig(c).TIDAL.Enabled()
}

const (
	// MatchScoreStrong is the minimum score for the highest-confidence band.
	MatchScoreStrong = 100
	// MatchScoreProbable is the minimum score for likely-good matches.
	MatchScoreProbable = 70
	// MatchScoreWeak is the minimum score for low-confidence but retained matches.
	MatchScoreWeak = 50
)

// MatchStrengthForScore maps a raw score to a user-facing confidence band.
func MatchStrengthForScore(score int) MatchStrength {
	switch {
	case score >= MatchScoreStrong:
		return MatchStrengthStrong
	case score >= MatchScoreProbable:
		return MatchStrengthProbable
	case score >= MatchScoreWeak:
		return MatchStrengthWeak
	default:
		return MatchStrengthVeryWeak
	}
}

// DefaultConfig returns a Config with built-in defaults applied.
func DefaultConfig() Config {
	return Config{
		AppleMusicStorefront: "us",
		HTTPTimeout:          httpx.DefaultTimeout(),
		scoreWeights:         score.DefaultWeights(),
		songScoreWeights:     score.DefaultSongWeights(),
	}
}

// LoadConfig loads configuration from the default sources.
func LoadConfig() Config {
	return configFromInternal(internalconfig.Load())
}

// LoadConfigFromEnv loads configuration using the supplied getenv function.
func LoadConfigFromEnv(getenv func(string) string) Config {
	return configFromInternal(internalconfig.LoadFromEnv(getenv))
}

func configFromInternal(cfg internalconfig.Config) Config {
	return normalizedConfig(Config{
		Spotify: SpotifyConfig{
			ClientID:     cfg.Spotify.ClientID,
			ClientSecret: cfg.Spotify.ClientSecret,
		},
		AppleMusic: AppleMusicConfig{
			KeyID:          cfg.AppleMusic.KeyID,
			TeamID:         cfg.AppleMusic.TeamID,
			PrivateKeyPath: cfg.AppleMusic.PrivateKeyPath,
		},
		TIDAL: TIDALConfig{
			ClientID:     cfg.TIDAL.ClientID,
			ClientSecret: cfg.TIDAL.ClientSecret,
		},
		AppleMusicStorefront: cfg.AppleMusic.Storefront,
		HTTPTimeout:          cfg.HTTPTimeout,
		TargetServices:       targetServicesFromConfigValue(cfg.TargetServicesRaw),
	})
}

// internalConfig converts the public Config DTO into the shape the Provider
// Catalog consumes. It is the single public-to-internal seam, so it normalizes:
// every Catalog query and the default adapter build see trimmed credentials and
// defaults exactly as New does.
func internalConfig(config Config) internalconfig.Config {
	config = normalizedConfig(config)
	return internalconfig.Config{
		Spotify:     internalconfig.Spotify{ClientID: config.Spotify.ClientID, ClientSecret: config.Spotify.ClientSecret},
		AppleMusic:  internalconfig.AppleMusic{Storefront: config.AppleMusicStorefront, KeyID: config.AppleMusic.KeyID, TeamID: config.AppleMusic.TeamID, PrivateKeyPath: config.AppleMusic.PrivateKeyPath},
		TIDAL:       internalconfig.TIDAL{ClientID: config.TIDAL.ClientID, ClientSecret: config.TIDAL.ClientSecret},
		HTTPTimeout: config.HTTPTimeout,
	}
}

func normalizedConfig(config Config) Config {
	config.AppleMusicStorefront = strings.ToLower(strings.TrimSpace(config.AppleMusicStorefront))
	if config.AppleMusicStorefront == "" {
		config.AppleMusicStorefront = "us"
	}
	config.Spotify.ClientID = strings.TrimSpace(config.Spotify.ClientID)
	config.Spotify.ClientSecret = strings.TrimSpace(config.Spotify.ClientSecret)
	config.AppleMusic.KeyID = strings.TrimSpace(config.AppleMusic.KeyID)
	config.AppleMusic.TeamID = strings.TrimSpace(config.AppleMusic.TeamID)
	config.AppleMusic.PrivateKeyPath = strings.TrimSpace(config.AppleMusic.PrivateKeyPath)
	config.TIDAL.ClientID = strings.TrimSpace(config.TIDAL.ClientID)
	config.TIDAL.ClientSecret = strings.TrimSpace(config.TIDAL.ClientSecret)
	config.HTTPTimeout = normalizeHTTPTimeout(config.HTTPTimeout)
	config.TargetServices = normalizedTargetServices(config.TargetServices)
	if config.scoreWeights == (score.Weights{}) {
		config.scoreWeights = score.DefaultWeights()
	}
	if config.songScoreWeights == (score.SongWeights{}) {
		config.songScoreWeights = score.DefaultSongWeights()
	}
	return config
}

func normalizeHTTPTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return httpx.DefaultTimeout()
	}
	return timeout
}

func targetServicesFromConfigValue(value string) []ServiceName {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	services := make([]ServiceName, 0)
	for part := range strings.SplitSeq(value, ",") {
		services = append(services, ServiceName(part))
	}
	return normalizedTargetServices(services)
}

func normalizedTargetServices(services []ServiceName) []ServiceName {
	if len(services) == 0 {
		return nil
	}

	normalized := make([]ServiceName, 0, len(services))
	seen := make(map[ServiceName]struct{}, len(services))
	for _, service := range services {
		service, ok := wiring.Default.LookupSupportedTargetService(string(service))
		if !ok {
			continue
		}
		if _, ok := seen[service]; ok {
			continue
		}
		seen[service] = struct{}{}
		normalized = append(normalized, service)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}
