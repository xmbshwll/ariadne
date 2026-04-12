package ariadne

import (
	"strings"
	"time"

	internalconfig "github.com/xmbshwll/ariadne/internal/config"
	"github.com/xmbshwll/ariadne/internal/httpx"
	"github.com/xmbshwll/ariadne/internal/score"
)

// ScoreWeights configures how ranking signals contribute to match scores.
type ScoreWeights struct {
	// UPCExact is added for an exact UPC match.
	UPCExact int
	// ISRCStrongOverlap is added when ISRC overlap is strong.
	ISRCStrongOverlap int
	// ISRCPartialScale scales partial ISRC overlap scores.
	ISRCPartialScale int
	// TrackTitleStrong is added when track-title overlap is strong.
	TrackTitleStrong int
	// TrackTitlePartial scales partial track-title overlap scores.
	TrackTitlePartial int
	// TitleExact is added for an exact normalized title match.
	TitleExact int
	// CoreTitleExact is added for a core-title match after edition markers are removed.
	CoreTitleExact int
	// PrimaryArtistExact is added for an exact primary artist match.
	PrimaryArtistExact int
	// ArtistOverlap is added for any non-primary artist overlap.
	ArtistOverlap int
	// TrackCountExact is added for an exact track-count match.
	TrackCountExact int
	// TrackCountNear is added when track counts differ by one.
	TrackCountNear int
	// TrackCountMismatch is applied when track counts differ significantly.
	TrackCountMismatch int
	// ReleaseDateExact is added for an exact release-date match.
	ReleaseDateExact int
	// ReleaseYearExact is added for a same-year release-date match.
	ReleaseYearExact int
	// DurationNear is added when total durations are close.
	DurationNear int
	// LabelExact is added for an exact normalized label match.
	LabelExact int
	// ExplicitMismatch is applied when explicit flags differ.
	ExplicitMismatch int
	// EditionMismatch is applied when edition hints disagree.
	EditionMismatch int
	// EditionMarkerPenalty is applied per unmatched edition marker in the title.
	EditionMarkerPenalty int
}

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
	// ScoreWeights controls how the album ranking algorithm weights matching signals.
	ScoreWeights ScoreWeights
	// SongScoreWeights controls how the song ranking algorithm weights matching signals.
	SongScoreWeights SongScoreWeights
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
func (c Config) SpotifyEnabled() bool {
	clientID := strings.TrimSpace(c.Spotify.ClientID)
	clientSecret := strings.TrimSpace(c.Spotify.ClientSecret)
	return clientID != "" && clientSecret != ""
}

// TIDALEnabled reports whether TIDAL credential-gated features are available.
func (c Config) TIDALEnabled() bool {
	clientID := strings.TrimSpace(c.TIDAL.ClientID)
	clientSecret := strings.TrimSpace(c.TIDAL.ClientSecret)
	return clientID != "" && clientSecret != ""
}

// DefaultScoreWeights returns the built-in album ranking weights.
func DefaultScoreWeights() ScoreWeights {
	return fromInternalScoreWeights(score.DefaultWeights())
}

// SongScoreWeights configures how ranking signals contribute to song match scores.
type SongScoreWeights struct {
	ISRCExact            int
	TitleExact           int
	CoreTitleExact       int
	PrimaryArtistExact   int
	ArtistOverlap        int
	DurationNear         int
	AlbumTitleExact      int
	ReleaseDateExact     int
	ReleaseYearExact     int
	TrackNumberExact     int
	ExplicitMismatch     int
	EditionMismatch      int
	EditionMarkerPenalty int
}

// DefaultSongScoreWeights returns the built-in song ranking weights.
func DefaultSongScoreWeights() SongScoreWeights {
	return fromInternalSongScoreWeights(score.DefaultSongWeights())
}

const (
	// MatchScoreStrong is the minimum score for the highest-confidence band.
	MatchScoreStrong = 100
	// MatchScoreProbable is the minimum score for likely-good matches.
	MatchScoreProbable = 70
	// MatchScoreWeak is the minimum score for low-confidence but retained matches.
	MatchScoreWeak = 50
)

// MatchStrengthForScore maps a raw score into a confidence band.
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

// DefaultConfig returns the library defaults without reading the environment.
func DefaultConfig() Config {
	return Config{
		AppleMusicStorefront: "us",
		HTTPTimeout:          httpx.DefaultTimeout(),
		ScoreWeights:         DefaultScoreWeights(),
		SongScoreWeights:     DefaultSongScoreWeights(),
	}
}

// LoadConfig loads library configuration from the current environment.
func LoadConfig() Config {
	return configFromInternal(internalconfig.Load())
}

// LoadConfigFromEnv loads library configuration from a caller-provided getenv function.
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
		TargetServices:       fromInternalServiceNames(cfg.TargetServices),
	})
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
	if config.ScoreWeights == (ScoreWeights{}) {
		config.ScoreWeights = DefaultScoreWeights()
	}
	if config.SongScoreWeights == (SongScoreWeights{}) {
		config.SongScoreWeights = DefaultSongScoreWeights()
	}
	return config
}

func normalizeHTTPTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return httpx.DefaultTimeout()
	}
	return timeout
}

func normalizedTargetServices(services []ServiceName) []ServiceName {
	if len(services) == 0 {
		return nil
	}

	normalized := make([]ServiceName, 0, len(services))
	seen := make(map[ServiceName]struct{}, len(services))
	for _, service := range services {
		service = ServiceName(strings.TrimSpace(string(service)))
		if service == "" {
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
