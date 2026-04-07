package ariadne

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

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
	"github.com/xmbshwll/ariadne/internal/score"
)

// ServiceName identifies a music service known to the library.
type ServiceName string

const (
	// ServiceSpotify identifies Spotify.
	ServiceSpotify ServiceName = "spotify"
	// ServiceAppleMusic identifies Apple Music.
	ServiceAppleMusic ServiceName = "appleMusic"
	// ServiceDeezer identifies Deezer.
	ServiceDeezer ServiceName = "deezer"
	// ServiceSoundCloud identifies SoundCloud.
	ServiceSoundCloud ServiceName = "soundcloud"
	// ServiceBandcamp identifies Bandcamp.
	ServiceBandcamp ServiceName = "bandcamp"
	// ServiceYouTubeMusic identifies YouTube Music.
	ServiceYouTubeMusic ServiceName = "youtubeMusic"
	// ServiceTIDAL identifies TIDAL.
	ServiceTIDAL ServiceName = "tidal"
	// ServiceAmazonMusic identifies Amazon Music.
	ServiceAmazonMusic ServiceName = "amazonMusic"
)

// MatchStrength buckets raw scores into user-facing confidence bands.
type MatchStrength string

const (
	// MatchStrengthVeryWeak indicates a low-confidence match.
	MatchStrengthVeryWeak MatchStrength = "very_weak"
	// MatchStrengthWeak indicates a weak match.
	MatchStrengthWeak MatchStrength = "weak"
	// MatchStrengthProbable indicates a probable match.
	MatchStrengthProbable MatchStrength = "probable"
	// MatchStrengthStrong indicates a strong match.
	MatchStrengthStrong MatchStrength = "strong"
)

// ParsedAlbumURL is the normalized form of a parsed source URL.
type ParsedAlbumURL struct {
	// Service is the service that recognized the input URL.
	Service ServiceName
	// EntityType is the parsed entity kind, usually "album".
	EntityType string
	// ID is the service-specific album identifier.
	ID string
	// CanonicalURL is the normalized URL form for the parsed album.
	CanonicalURL string
	// RegionHint is the storefront or market implied by the URL when known.
	RegionHint string
	// RawURL is the original caller-provided URL.
	RawURL string
}

// CanonicalTrack is the normalized track representation shared across services.
type CanonicalTrack struct {
	// DiscNumber is the 1-based disc index when known.
	DiscNumber int
	// TrackNumber is the 1-based track index within the disc when known.
	TrackNumber int
	// Title is the service-provided track title.
	Title string
	// NormalizedTitle is the normalized form used for matching.
	NormalizedTitle string
	// DurationMS is the track duration in milliseconds when known.
	DurationMS int
	// ISRC is the track's International Standard Recording Code when known.
	ISRC string
	// Artists lists the credited artist names for the track.
	Artists []string
}

// CanonicalAlbum is the normalized album representation shared across services.
type CanonicalAlbum struct {
	// Service is the service that supplied this album.
	Service ServiceName
	// SourceID is the service-specific album identifier.
	SourceID string
	// SourceURL is the canonical service URL for the album.
	SourceURL string
	// RegionHint is the storefront or market implied by the source data when known.
	RegionHint string
	// Title is the service-provided album title.
	Title string
	// NormalizedTitle is the normalized title used for matching.
	NormalizedTitle string
	// Artists lists the credited album artist names.
	Artists []string
	// NormalizedArtists contains the normalized artist names used for matching.
	NormalizedArtists []string
	// ReleaseDate is the service-provided release date string.
	ReleaseDate string
	// Label is the record label when known.
	Label string
	// UPC is the album's Universal Product Code when known.
	UPC string
	// TrackCount is the number of tracks when known.
	TrackCount int
	// TotalDurationMS is the summed track duration in milliseconds when known.
	TotalDurationMS int
	// ArtworkURL is the preferred cover-art URL when known.
	ArtworkURL string
	// Explicit reports whether the release is marked explicit.
	Explicit bool
	// EditionHints contains normalized descriptors such as remaster or deluxe.
	EditionHints []string
	// Tracks contains the normalized track listing when available.
	Tracks []CanonicalTrack
}

// CandidateAlbum is one service-specific search result mapped into canonical form.
type CandidateAlbum struct {
	CanonicalAlbum
	// CandidateID is the service-specific identifier for the search result.
	CandidateID string
	// MatchURL is the service URL that should be presented for this candidate.
	MatchURL string
}

// ScoredMatch is one ranked candidate returned by the resolver.
type ScoredMatch struct {
	// URL is the best presentation URL for the candidate.
	URL string
	// Score is the aggregate matching score.
	Score int
	// Reasons lists the major signals that contributed to the score.
	Reasons []string
	// Candidate is the underlying canonicalized candidate payload.
	Candidate CandidateAlbum
}

// MatchResult is the ranked output for one target service.
type MatchResult struct {
	// Service is the target service that was searched.
	Service ServiceName
	// Best is the highest-ranked candidate, or nil when nothing matched.
	Best *ScoredMatch
	// Alternates contains lower-ranked candidates after Best.
	Alternates []ScoredMatch
}

// Resolution is the full output of resolving one input album URL.
type Resolution struct {
	// InputURL is the original URL passed to ResolveAlbum.
	InputURL string
	// Parsed is the normalized parsed form of the source URL.
	Parsed ParsedAlbumURL
	// Source is the canonical album fetched from the source service.
	Source CanonicalAlbum
	// Matches contains ranked target-service matches keyed by service name.
	Matches map[ServiceName]MatchResult
}

// SourceAdapter fetches canonical album metadata from a parsed source URL.
type SourceAdapter interface {
	Service() ServiceName
	ParseAlbumURL(raw string) (*ParsedAlbumURL, error)
	FetchAlbum(ctx context.Context, parsed ParsedAlbumURL) (*CanonicalAlbum, error)
}

// TargetAdapter searches a target service for matching albums.
type TargetAdapter interface {
	Service() ServiceName
	SearchByUPC(ctx context.Context, upc string) ([]CandidateAlbum, error)
	SearchByISRC(ctx context.Context, isrcs []string) ([]CandidateAlbum, error)
	SearchByMetadata(ctx context.Context, album CanonicalAlbum) ([]CandidateAlbum, error)
}

var (
	// ErrUnsupportedURL indicates that no registered source adapter recognized the input URL.
	ErrUnsupportedURL = resolve.ErrUnsupportedURL
	// ErrNoSourceAdapters indicates that a resolver was created without source adapters.
	ErrNoSourceAdapters = resolve.ErrNoSourceAdapters
	// ErrAmazonMusicDeferred indicates that Amazon Music URLs are recognized, but runtime resolution remains intentionally deferred.
	ErrAmazonMusicDeferred = amazonmusicadapter.ErrDeferredRuntimeAdapter
	// ErrAppleMusicCredentialsNotConfigured indicates that an Apple Music official API operation requires developer token credentials.
	ErrAppleMusicCredentialsNotConfigured = applemusicadapter.ErrCredentialsNotConfigured
	// ErrSpotifyCredentialsNotConfigured indicates that a Spotify Web API operation requires app credentials.
	ErrSpotifyCredentialsNotConfigured = spotifyadapter.ErrCredentialsNotConfigured
	// ErrTIDALCredentialsNotConfigured indicates that a TIDAL operation requires app credentials that were not configured.
	ErrTIDALCredentialsNotConfigured = tidaladapter.ErrCredentialsNotConfigured

	errSourceAdapterReturnedNilParsed = errors.New("source adapter returned nil parsed album url")
	errSourceAdapterReturnedNilAlbum  = errors.New("source adapter returned nil album")
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
	// ScoreWeights controls how the ranking algorithm weights matching signals.
	ScoreWeights ScoreWeights
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
	return c.Spotify.ClientID != "" && c.Spotify.ClientSecret != ""
}

// TIDALEnabled reports whether TIDAL credential-gated features are available.
func (c Config) TIDALEnabled() bool {
	return c.TIDAL.ClientID != "" && c.TIDAL.ClientSecret != ""
}

// Resolver wraps the internal resolver with a public library-facing API.
type Resolver struct {
	inner *resolve.Resolver
}

// DefaultScoreWeights returns the built-in ranking weights.
func DefaultScoreWeights() ScoreWeights {
	return fromInternalScoreWeights(score.DefaultWeights())
}

// MatchStrengthForScore maps a raw score into a confidence band.
func MatchStrengthForScore(score int) MatchStrength {
	switch {
	case score >= 100:
		return MatchStrengthStrong
	case score >= 70:
		return MatchStrengthProbable
	case score >= 50:
		return MatchStrengthWeak
	default:
		return MatchStrengthVeryWeak
	}
}

// DefaultConfig returns the library defaults without reading the environment.
func DefaultConfig() Config {
	return Config{AppleMusicStorefront: "us", HTTPTimeout: httpx.DefaultTimeout(), ScoreWeights: DefaultScoreWeights()}
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
	config = normalizedConfig(config)
	return NewWithClient(httpx.NewClient(config.HTTPTimeout), config)
}

// NewWithClient builds a Resolver with the default adapter set and a caller-provided HTTP client.
func NewWithClient(client *http.Client, config Config) *Resolver {
	if client == nil {
		client = http.DefaultClient
	}
	config = normalizedConfig(config)
	return &Resolver{inner: resolve.New(defaultSourceAdapters(client, config), defaultTargetAdapters(client, config), toInternalScoreWeights(config.ScoreWeights))}
}

// NewWithAdapters builds a Resolver from caller-provided source and target adapters using the default ranking weights.
func NewWithAdapters(sources []SourceAdapter, targets []TargetAdapter) *Resolver {
	return NewWithAdaptersAndWeights(sources, targets, DefaultScoreWeights())
}

// NewWithAdaptersAndWeights builds a Resolver from caller-provided source and target adapters and explicit ranking weights.
func NewWithAdaptersAndWeights(sources []SourceAdapter, targets []TargetAdapter, weights ScoreWeights) *Resolver {
	return &Resolver{inner: resolve.New(wrapSourceAdapters(sources), wrapTargetAdapters(targets), toInternalScoreWeights(weights))}
}

// ResolveAlbum resolves one input album URL into a canonical source album plus per-service matches.
//
// Callers should use errors.Is on the returned error when branching on public
// resolver failure modes. The stable exported sentinels are:
//   - ErrUnsupportedURL when no registered source adapter recognizes inputURL
//   - ErrNoSourceAdapters when the resolver was built without any source adapters
//   - ErrAmazonMusicDeferred when an Amazon Music URL is recognized but runtime
//     resolution is intentionally deferred
//   - ErrAppleMusicCredentialsNotConfigured when an Apple Music official API
//     operation requires developer token credentials
//   - ErrSpotifyCredentialsNotConfigured when a Spotify Web API operation
//     requires app credentials
//   - ErrTIDALCredentialsNotConfigured when a TIDAL source or target operation
//     requires credentials that are not configured
func (r *Resolver) ResolveAlbum(ctx context.Context, inputURL string) (*Resolution, error) {
	resolution, err := r.inner.ResolveAlbum(ctx, inputURL)
	if err != nil {
		//nolint:wrapcheck // Preserve the underlying resolver error for callers and CLI output.
		return nil, err
	}
	public := fromInternalResolution(*resolution)
	return &public, nil
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
	if config.SpotifyEnabled() {
		targets = append(targets, spotifyadapter.New(client, spotifyadapter.WithCredentials(config.Spotify.ClientID, config.Spotify.ClientSecret)))
	}
	if config.TIDALEnabled() {
		targets = append(targets, tidaladapter.New(client, tidaladapter.WithCredentials(config.TIDAL.ClientID, config.TIDAL.ClientSecret)))
	}
	return filterTargetAdapters(targets, config.TargetServices)
}

func filterTargetAdapters(targets []resolve.TargetAdapter, services []ServiceName) []resolve.TargetAdapter {
	if len(services) == 0 {
		return targets
	}

	allowed := make(map[ServiceName]struct{}, len(services))
	for _, service := range services {
		allowed[service] = struct{}{}
	}

	filtered := make([]resolve.TargetAdapter, 0, len(targets))
	for _, target := range targets {
		if _, ok := allowed[fromInternalServiceName(target.Service())]; ok {
			filtered = append(filtered, target)
		}
	}
	return filtered
}

func wrapSourceAdapters(sources []SourceAdapter) []resolve.SourceAdapter {
	wrapped := make([]resolve.SourceAdapter, 0, len(sources))
	for _, source := range sources {
		wrapped = append(wrapped, sourceAdapterBridge{source: source})
	}
	return wrapped
}

func wrapTargetAdapters(targets []TargetAdapter) []resolve.TargetAdapter {
	wrapped := make([]resolve.TargetAdapter, 0, len(targets))
	for _, target := range targets {
		wrapped = append(wrapped, targetAdapterBridge{target: target})
	}
	return wrapped
}

type sourceAdapterBridge struct {
	source SourceAdapter
}

func (b sourceAdapterBridge) Service() model.ServiceName {
	return toInternalServiceName(b.source.Service())
}

func (b sourceAdapterBridge) ParseAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	parsed, err := b.source.ParseAlbumURL(raw)
	if err != nil || parsed == nil {
		if err != nil {
			//nolint:wrapcheck // Preserve adapter parse errors without adding another wrapper layer.
			return nil, err
		}
		return nil, errSourceAdapterReturnedNilParsed
	}
	internal := toInternalParsedAlbumURL(*parsed)
	return &internal, nil
}

func (b sourceAdapterBridge) FetchAlbum(ctx context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalAlbum, error) {
	album, err := b.source.FetchAlbum(ctx, fromInternalParsedAlbumURL(parsed))
	if err != nil || album == nil {
		if err != nil {
			//nolint:wrapcheck // Preserve adapter fetch errors without adding another wrapper layer.
			return nil, err
		}
		return nil, errSourceAdapterReturnedNilAlbum
	}
	internal := toInternalCanonicalAlbum(*album)
	return &internal, nil
}

type targetAdapterBridge struct {
	target TargetAdapter
}

func (b targetAdapterBridge) Service() model.ServiceName {
	return toInternalServiceName(b.target.Service())
}

func (b targetAdapterBridge) SearchByUPC(ctx context.Context, upc string) ([]model.CandidateAlbum, error) {
	albums, err := b.target.SearchByUPC(ctx, upc)
	if err != nil {
		//nolint:wrapcheck // Preserve target adapter errors without adding another wrapper layer.
		return nil, err
	}
	return toInternalCandidateAlbums(albums), nil
}

func (b targetAdapterBridge) SearchByISRC(ctx context.Context, isrcs []string) ([]model.CandidateAlbum, error) {
	albums, err := b.target.SearchByISRC(ctx, append([]string(nil), isrcs...))
	if err != nil {
		//nolint:wrapcheck // Preserve target adapter errors without adding another wrapper layer.
		return nil, err
	}
	return toInternalCandidateAlbums(albums), nil
}

func (b targetAdapterBridge) SearchByMetadata(ctx context.Context, album model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
	albums, err := b.target.SearchByMetadata(ctx, fromInternalCanonicalAlbum(album))
	if err != nil {
		//nolint:wrapcheck // Preserve target adapter errors without adding another wrapper layer.
		return nil, err
	}
	return toInternalCandidateAlbums(albums), nil
}

func toInternalServiceName(service ServiceName) model.ServiceName {
	return model.ServiceName(service)
}

func fromInternalServiceName(service model.ServiceName) ServiceName {
	return ServiceName(service)
}

func toInternalScoreWeights(weights ScoreWeights) score.Weights {
	return score.Weights{
		UPCExact:             weights.UPCExact,
		ISRCStrongOverlap:    weights.ISRCStrongOverlap,
		ISRCPartialScale:     weights.ISRCPartialScale,
		TrackTitleStrong:     weights.TrackTitleStrong,
		TrackTitlePartial:    weights.TrackTitlePartial,
		TitleExact:           weights.TitleExact,
		CoreTitleExact:       weights.CoreTitleExact,
		PrimaryArtistExact:   weights.PrimaryArtistExact,
		ArtistOverlap:        weights.ArtistOverlap,
		TrackCountExact:      weights.TrackCountExact,
		TrackCountNear:       weights.TrackCountNear,
		TrackCountMismatch:   weights.TrackCountMismatch,
		ReleaseDateExact:     weights.ReleaseDateExact,
		ReleaseYearExact:     weights.ReleaseYearExact,
		DurationNear:         weights.DurationNear,
		LabelExact:           weights.LabelExact,
		ExplicitMismatch:     weights.ExplicitMismatch,
		EditionMismatch:      weights.EditionMismatch,
		EditionMarkerPenalty: weights.EditionMarkerPenalty,
	}
}

func fromInternalScoreWeights(weights score.Weights) ScoreWeights {
	return ScoreWeights{
		UPCExact:             weights.UPCExact,
		ISRCStrongOverlap:    weights.ISRCStrongOverlap,
		ISRCPartialScale:     weights.ISRCPartialScale,
		TrackTitleStrong:     weights.TrackTitleStrong,
		TrackTitlePartial:    weights.TrackTitlePartial,
		TitleExact:           weights.TitleExact,
		CoreTitleExact:       weights.CoreTitleExact,
		PrimaryArtistExact:   weights.PrimaryArtistExact,
		ArtistOverlap:        weights.ArtistOverlap,
		TrackCountExact:      weights.TrackCountExact,
		TrackCountNear:       weights.TrackCountNear,
		TrackCountMismatch:   weights.TrackCountMismatch,
		ReleaseDateExact:     weights.ReleaseDateExact,
		ReleaseYearExact:     weights.ReleaseYearExact,
		DurationNear:         weights.DurationNear,
		LabelExact:           weights.LabelExact,
		ExplicitMismatch:     weights.ExplicitMismatch,
		EditionMismatch:      weights.EditionMismatch,
		EditionMarkerPenalty: weights.EditionMarkerPenalty,
	}
}

func toInternalParsedAlbumURL(parsed ParsedAlbumURL) model.ParsedAlbumURL {
	return model.ParsedAlbumURL{
		Service:      toInternalServiceName(parsed.Service),
		EntityType:   parsed.EntityType,
		ID:           parsed.ID,
		CanonicalURL: parsed.CanonicalURL,
		RegionHint:   parsed.RegionHint,
		RawURL:       parsed.RawURL,
	}
}

func fromInternalParsedAlbumURL(parsed model.ParsedAlbumURL) ParsedAlbumURL {
	return ParsedAlbumURL{
		Service:      fromInternalServiceName(parsed.Service),
		EntityType:   parsed.EntityType,
		ID:           parsed.ID,
		CanonicalURL: parsed.CanonicalURL,
		RegionHint:   parsed.RegionHint,
		RawURL:       parsed.RawURL,
	}
}

func toInternalCanonicalTrack(track CanonicalTrack) model.CanonicalTrack {
	return model.CanonicalTrack{
		DiscNumber:      track.DiscNumber,
		TrackNumber:     track.TrackNumber,
		Title:           track.Title,
		NormalizedTitle: track.NormalizedTitle,
		DurationMS:      track.DurationMS,
		ISRC:            track.ISRC,
		Artists:         append([]string(nil), track.Artists...),
	}
}

func fromInternalCanonicalTrack(track model.CanonicalTrack) CanonicalTrack {
	return CanonicalTrack{
		DiscNumber:      track.DiscNumber,
		TrackNumber:     track.TrackNumber,
		Title:           track.Title,
		NormalizedTitle: track.NormalizedTitle,
		DurationMS:      track.DurationMS,
		ISRC:            track.ISRC,
		Artists:         append([]string(nil), track.Artists...),
	}
}

func toInternalCanonicalAlbum(album CanonicalAlbum) model.CanonicalAlbum {
	tracks := make([]model.CanonicalTrack, 0, len(album.Tracks))
	for _, track := range album.Tracks {
		tracks = append(tracks, toInternalCanonicalTrack(track))
	}
	return model.CanonicalAlbum{
		Service:           toInternalServiceName(album.Service),
		SourceID:          album.SourceID,
		SourceURL:         album.SourceURL,
		RegionHint:        album.RegionHint,
		Title:             album.Title,
		NormalizedTitle:   album.NormalizedTitle,
		Artists:           append([]string(nil), album.Artists...),
		NormalizedArtists: append([]string(nil), album.NormalizedArtists...),
		ReleaseDate:       album.ReleaseDate,
		Label:             album.Label,
		UPC:               album.UPC,
		TrackCount:        album.TrackCount,
		TotalDurationMS:   album.TotalDurationMS,
		ArtworkURL:        album.ArtworkURL,
		Explicit:          album.Explicit,
		EditionHints:      append([]string(nil), album.EditionHints...),
		Tracks:            tracks,
	}
}

func fromInternalCanonicalAlbum(album model.CanonicalAlbum) CanonicalAlbum {
	tracks := make([]CanonicalTrack, 0, len(album.Tracks))
	for _, track := range album.Tracks {
		tracks = append(tracks, fromInternalCanonicalTrack(track))
	}
	return CanonicalAlbum{
		Service:           fromInternalServiceName(album.Service),
		SourceID:          album.SourceID,
		SourceURL:         album.SourceURL,
		RegionHint:        album.RegionHint,
		Title:             album.Title,
		NormalizedTitle:   album.NormalizedTitle,
		Artists:           append([]string(nil), album.Artists...),
		NormalizedArtists: append([]string(nil), album.NormalizedArtists...),
		ReleaseDate:       album.ReleaseDate,
		Label:             album.Label,
		UPC:               album.UPC,
		TrackCount:        album.TrackCount,
		TotalDurationMS:   album.TotalDurationMS,
		ArtworkURL:        album.ArtworkURL,
		Explicit:          album.Explicit,
		EditionHints:      append([]string(nil), album.EditionHints...),
		Tracks:            tracks,
	}
}

func toInternalCandidateAlbum(album CandidateAlbum) model.CandidateAlbum {
	return model.CandidateAlbum{
		CanonicalAlbum: toInternalCanonicalAlbum(album.CanonicalAlbum),
		CandidateID:    album.CandidateID,
		MatchURL:       album.MatchURL,
	}
}

func fromInternalCandidateAlbum(album model.CandidateAlbum) CandidateAlbum {
	return CandidateAlbum{
		CanonicalAlbum: fromInternalCanonicalAlbum(album.CanonicalAlbum),
		CandidateID:    album.CandidateID,
		MatchURL:       album.MatchURL,
	}
}

func toInternalCandidateAlbums(albums []CandidateAlbum) []model.CandidateAlbum {
	if len(albums) == 0 {
		return nil
	}
	internal := make([]model.CandidateAlbum, 0, len(albums))
	for _, album := range albums {
		internal = append(internal, toInternalCandidateAlbum(album))
	}
	return internal
}

func fromInternalScoredMatch(match resolve.ScoredMatch) ScoredMatch {
	return ScoredMatch{
		URL:       match.URL,
		Score:     match.Score,
		Reasons:   append([]string(nil), match.Reasons...),
		Candidate: fromInternalCandidateAlbum(match.Candidate),
	}
}

func fromInternalMatchResult(result resolve.MatchResult) MatchResult {
	public := MatchResult{
		Service:    fromInternalServiceName(result.Service),
		Alternates: make([]ScoredMatch, 0, len(result.Alternates)),
	}
	if result.Best != nil {
		best := fromInternalScoredMatch(*result.Best)
		public.Best = &best
	}
	for _, alternate := range result.Alternates {
		public.Alternates = append(public.Alternates, fromInternalScoredMatch(alternate))
	}
	return public
}

func fromInternalResolution(resolution resolve.Resolution) Resolution {
	matches := make(map[ServiceName]MatchResult, len(resolution.Matches))
	for service, match := range resolution.Matches {
		matches[fromInternalServiceName(service)] = fromInternalMatchResult(match)
	}
	return Resolution{
		InputURL: resolution.InputURL,
		Parsed:   fromInternalParsedAlbumURL(resolution.Parsed),
		Source:   fromInternalCanonicalAlbum(resolution.Source),
		Matches:  matches,
	}
}
