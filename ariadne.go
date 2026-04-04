package ariadne

import (
	"context"
	"net/http"
	"strings"

	amazonmusicadapter "github.com/xmbshwll/ariadne/internal/adapters/amazonmusic"
	applemusicadapter "github.com/xmbshwll/ariadne/internal/adapters/applemusic"
	bandcampadapter "github.com/xmbshwll/ariadne/internal/adapters/bandcamp"
	deezeradapter "github.com/xmbshwll/ariadne/internal/adapters/deezer"
	soundcloudadapter "github.com/xmbshwll/ariadne/internal/adapters/soundcloud"
	spotifyadapter "github.com/xmbshwll/ariadne/internal/adapters/spotify"
	tidaladapter "github.com/xmbshwll/ariadne/internal/adapters/tidal"
	youtubemusicadapter "github.com/xmbshwll/ariadne/internal/adapters/youtubemusic"
	internalconfig "github.com/xmbshwll/ariadne/internal/config"
	"github.com/xmbshwll/ariadne/internal/httpx"
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/resolve"
)

// ServiceName identifies a music service known to the library.
type ServiceName = model.ServiceName

// ParsedAlbumURL is the normalized form of a parsed source URL.
type ParsedAlbumURL = model.ParsedAlbumURL

// CanonicalTrack is the normalized track representation shared across services.
type CanonicalTrack = model.CanonicalTrack

// CanonicalAlbum is the normalized album representation shared across services.
type CanonicalAlbum = model.CanonicalAlbum

// CandidateAlbum is one service-specific search result mapped into canonical form.
type CandidateAlbum = model.CandidateAlbum

// ScoredMatch is one ranked candidate returned by the resolver.
type ScoredMatch = resolve.ScoredMatch

// MatchResult is the ranked output for one target service.
type MatchResult = resolve.MatchResult

// Resolution is the full output of resolving one input album URL.
type Resolution = resolve.Resolution

// SourceAdapter fetches canonical album metadata from a parsed source URL.
type SourceAdapter = resolve.SourceAdapter

// TargetAdapter searches a target service for matching albums.
type TargetAdapter = resolve.TargetAdapter

const (
	// ServiceSpotify identifies Spotify.
	ServiceSpotify = model.ServiceSpotify
	// ServiceAppleMusic identifies Apple Music.
	ServiceAppleMusic = model.ServiceAppleMusic
	// ServiceDeezer identifies Deezer.
	ServiceDeezer = model.ServiceDeezer
	// ServiceSoundCloud identifies SoundCloud.
	ServiceSoundCloud = model.ServiceSoundCloud
	// ServiceBandcamp identifies Bandcamp.
	ServiceBandcamp = model.ServiceBandcamp
	// ServiceYouTubeMusic identifies YouTube Music.
	ServiceYouTubeMusic = model.ServiceYouTubeMusic
	// ServiceTIDAL identifies TIDAL.
	ServiceTIDAL = model.ServiceTIDAL
	// ServiceAmazonMusic identifies Amazon Music.
	ServiceAmazonMusic = model.ServiceAmazonMusic
)

var (
	// ErrUnsupportedURL indicates that no registered source adapter recognized the input URL.
	ErrUnsupportedURL = resolve.ErrUnsupportedURL
	// ErrNoSourceAdapters indicates that a resolver was created without source adapters.
	ErrNoSourceAdapters = resolve.ErrNoSourceAdapters
	// ErrAmazonMusicDeferred indicates that Amazon Music URLs are recognized, but runtime resolution remains intentionally deferred.
	ErrAmazonMusicDeferred = amazonmusicadapter.ErrDeferredRuntimeAdapter
)

// Config configures the default library resolver.
type Config struct {
	Spotify              SpotifyConfig
	AppleMusic           AppleMusicConfig
	TIDAL                TIDALConfig
	AppleMusicStorefront string
}

// SpotifyConfig holds Spotify app credentials used for target search and preferred source fetches.
type SpotifyConfig struct {
	ClientID     string
	ClientSecret string
}

// AppleMusicConfig holds Apple Music key material used to generate MusicKit developer tokens.
type AppleMusicConfig struct {
	KeyID          string
	TeamID         string
	PrivateKeyPath string
}

// TIDALConfig holds TIDAL client credentials used for official catalog access.
type TIDALConfig struct {
	ClientID     string
	ClientSecret string
}

// Resolver wraps the internal resolver with a public library-facing API.
type Resolver struct {
	inner *resolve.Resolver
}

// DefaultConfig returns the library defaults without reading the environment.
func DefaultConfig() Config {
	return Config{AppleMusicStorefront: "us"}
}

// LoadConfig loads library configuration from the current environment.
func LoadConfig() Config {
	return configFromInternal(internalconfig.Load())
}

// LoadConfigFromEnv loads library configuration from a caller-provided getenv function.
func LoadConfigFromEnv(getenv func(string) string) Config {
	return configFromInternal(internalconfig.LoadFromEnv(getenv))
}

// New builds a Resolver with the default adapter set and a default HTTP client.
func New(config Config) *Resolver {
	return NewWithClient(httpx.NewClient(), config)
}

// NewWithClient builds a Resolver with the default adapter set and a caller-provided HTTP client.
func NewWithClient(client *http.Client, config Config) *Resolver {
	if client == nil {
		client = http.DefaultClient
	}
	config = normalizedConfig(config)
	return &Resolver{inner: resolve.New(defaultSourceAdapters(client, config), defaultTargetAdapters(client, config))}
}

// NewWithAdapters builds a Resolver from caller-provided source and target adapters.
func NewWithAdapters(sources []SourceAdapter, targets []TargetAdapter) *Resolver {
	return &Resolver{inner: resolve.New(sources, targets)}
}

// ResolveAlbum resolves one input album URL into a canonical source album plus per-service matches.
func (r *Resolver) ResolveAlbum(ctx context.Context, inputURL string) (*Resolution, error) {
	return r.inner.ResolveAlbum(ctx, inputURL)
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
	return config
}

func spotifyCredentialsConfigured(config Config) bool {
	return config.Spotify.ClientID != "" && config.Spotify.ClientSecret != ""
}

func tidalCredentialsConfigured(config Config) bool {
	return config.TIDAL.ClientID != "" && config.TIDAL.ClientSecret != ""
}

func newAppleMusicAdapter(client *http.Client, config Config) *applemusicadapter.Adapter {
	return applemusicadapter.New(
		client,
		applemusicadapter.WithDefaultStorefront(config.AppleMusicStorefront),
		applemusicadapter.WithDeveloperTokenAuth(
			config.AppleMusic.KeyID,
			config.AppleMusic.TeamID,
			config.AppleMusic.PrivateKeyPath,
		),
	)
}

func defaultSourceAdapters(client *http.Client, config Config) []resolve.SourceAdapter {
	amazonMusic := amazonmusicadapter.New(client)
	appleMusic := newAppleMusicAdapter(client, config)
	bandcamp := bandcampadapter.New(client)
	deezer := deezeradapter.New(client)
	soundCloud := soundcloudadapter.New(client)
	youTubeMusic := youtubemusicadapter.New(client)
	spotify := spotifyadapter.New(client, spotifyadapter.WithCredentials(config.Spotify.ClientID, config.Spotify.ClientSecret))
	tidal := tidaladapter.New(client, tidaladapter.WithCredentials(config.TIDAL.ClientID, config.TIDAL.ClientSecret))
	return []resolve.SourceAdapter{appleMusic, deezer, spotify, tidal, soundCloud, youTubeMusic, amazonMusic, bandcamp}
}

func defaultTargetAdapters(client *http.Client, config Config) []resolve.TargetAdapter {
	appleMusic := newAppleMusicAdapter(client, config)
	bandcamp := bandcampadapter.New(client)
	deezer := deezeradapter.New(client)
	soundCloud := soundcloudadapter.New(client)
	youTubeMusic := youtubemusicadapter.New(client)
	targets := []resolve.TargetAdapter{appleMusic, bandcamp, deezer, soundCloud, youTubeMusic}
	if spotifyCredentialsConfigured(config) {
		targets = append(targets, spotifyadapter.New(client, spotifyadapter.WithCredentials(config.Spotify.ClientID, config.Spotify.ClientSecret)))
	}
	if tidalCredentialsConfigured(config) {
		targets = append(targets, tidaladapter.New(client, tidaladapter.WithCredentials(config.TIDAL.ClientID, config.TIDAL.ClientSecret)))
	}
	return targets
}
