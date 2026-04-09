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

// ParsedURL is the normalized form of a parsed source URL.
type ParsedURL struct {
	// Service is the service that recognized the input URL.
	Service ServiceName
	// EntityType is the parsed entity kind, such as "album" or "song".
	EntityType string
	// ID is the service-specific entity identifier.
	ID string
	// CanonicalURL is the normalized URL form for the parsed entity.
	CanonicalURL string
	// RegionHint is the storefront or market implied by the URL when known.
	RegionHint string
	// RawURL is the original caller-provided URL.
	RawURL string
}

// ParsedAlbumURL is kept as an alias while the public API expands beyond album-only resolution.
type ParsedAlbumURL = ParsedURL

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

// CanonicalSong is the normalized song representation shared across services.
type CanonicalSong struct {
	// Service is the service that supplied this song.
	Service ServiceName
	// SourceID is the service-specific song identifier.
	SourceID string
	// SourceURL is the canonical service URL for the song.
	SourceURL string
	// RegionHint is the storefront or market implied by the source data when known.
	RegionHint string
	// Title is the service-provided song title.
	Title string
	// NormalizedTitle is the normalized title used for matching.
	NormalizedTitle string
	// Artists lists the credited song artist names.
	Artists []string
	// NormalizedArtists contains the normalized artist names used for matching.
	NormalizedArtists []string
	// DurationMS is the song duration in milliseconds when known.
	DurationMS int
	// ISRC is the song's International Standard Recording Code when known.
	ISRC string
	// Explicit reports whether the song is marked explicit.
	Explicit bool
	// DiscNumber is the 1-based disc index when known.
	DiscNumber int
	// TrackNumber is the 1-based track index within the disc when known.
	TrackNumber int
	// AlbumID is the service-specific album identifier when album context is known.
	AlbumID string
	// AlbumTitle is the containing release title when known.
	AlbumTitle string
	// AlbumNormalizedTitle is the normalized album title used for matching.
	AlbumNormalizedTitle string
	// AlbumArtists lists the credited release artist names when known.
	AlbumArtists []string
	// AlbumNormalizedArtists contains normalized release artist names when known.
	AlbumNormalizedArtists []string
	// ReleaseDate is the service-provided release date string when known.
	ReleaseDate string
	// ArtworkURL is the preferred artwork URL when known.
	ArtworkURL string
	// EditionHints contains normalized descriptors such as live, edit, or remaster.
	EditionHints []string
}

// CandidateAlbum is one service-specific search result mapped into canonical form.
type CandidateAlbum struct {
	CanonicalAlbum
	// CandidateID is the service-specific identifier for the search result.
	CandidateID string
	// MatchURL is the service URL that should be presented for this candidate.
	MatchURL string
}

// CandidateSong is one service-specific song search result mapped into canonical form.
type CandidateSong struct {
	CanonicalSong
	// CandidateID is the service-specific identifier for the search result.
	CandidateID string
	// MatchURL is the service URL that should be presented for this candidate.
	MatchURL string
}

// ScoredMatch is one ranked candidate returned by the album resolver.
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

// SongScoredMatch is one ranked song candidate returned by the song resolver.
type SongScoredMatch struct {
	// URL is the best presentation URL for the candidate.
	URL string
	// Score is the aggregate matching score.
	Score int
	// Reasons lists the major signals that contributed to the score.
	Reasons []string
	// Candidate is the underlying canonicalized song payload.
	Candidate CandidateSong
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

// SongMatchResult is the ranked song output for one target service.
type SongMatchResult struct {
	// Service is the target service that was searched.
	Service ServiceName
	// Best is the highest-ranked candidate, or nil when nothing matched.
	Best *SongScoredMatch
	// Alternates contains lower-ranked candidates after Best.
	Alternates []SongScoredMatch
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

// SongResolution is the full output of resolving one input song URL.
type SongResolution struct {
	// InputURL is the original URL passed to ResolveSong.
	InputURL string
	// Parsed is the normalized parsed form of the source URL.
	Parsed ParsedURL
	// Source is the canonical song fetched from the source service.
	Source CanonicalSong
	// Matches contains ranked target-service matches keyed by service name.
	Matches map[ServiceName]SongMatchResult
}

// EntityResolution is the generic output of resolving one input URL.
type EntityResolution struct {
	// Parsed is the normalized parsed form of the source URL.
	Parsed ParsedURL
	// Album is set when the input resolved as an album.
	Album *Resolution
	// Song is set when the input resolved as a song.
	Song *SongResolution
}

// SourceAdapter fetches canonical album metadata from a parsed source URL.
type SourceAdapter interface {
	Service() ServiceName
	ParseAlbumURL(raw string) (*ParsedAlbumURL, error)
	FetchAlbum(ctx context.Context, parsed ParsedAlbumURL) (*CanonicalAlbum, error)
}

// SongSourceAdapter fetches canonical song metadata from a parsed source URL.
type SongSourceAdapter interface {
	Service() ServiceName
	ParseSongURL(raw string) (*ParsedURL, error)
	FetchSong(ctx context.Context, parsed ParsedURL) (*CanonicalSong, error)
}

// TargetAdapter searches a target service for matching albums.
type TargetAdapter interface {
	Service() ServiceName
	SearchByUPC(ctx context.Context, upc string) ([]CandidateAlbum, error)
	SearchByISRC(ctx context.Context, isrcs []string) ([]CandidateAlbum, error)
	SearchByMetadata(ctx context.Context, album CanonicalAlbum) ([]CandidateAlbum, error)
}

// SongTargetAdapter searches a target service for matching songs.
type SongTargetAdapter interface {
	Service() ServiceName
	SearchSongByISRC(ctx context.Context, isrc string) ([]CandidateSong, error)
	SearchSongByMetadata(ctx context.Context, song CanonicalSong) ([]CandidateSong, error)
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

	errSourceAdapterReturnedNilParsed = errors.New("source adapter returned nil parsed url")
	errSourceAdapterReturnedNilAlbum  = errors.New("source adapter returned nil album")
	errSourceAdapterReturnedNilSong   = errors.New("source adapter returned nil song")
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

// Resolver wraps the internal resolvers with a public library-facing API.
type Resolver struct {
	inner     *resolve.Resolver
	songInner *resolve.SongResolver
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
	return &Resolver{
		inner:     resolve.New(defaultSourceAdapters(client, config), defaultTargetAdapters(client, config), toInternalScoreWeights(config.ScoreWeights)),
		songInner: resolve.NewSongs(defaultSongSourceAdapters(client, config), defaultSongTargetAdapters(client, config), toInternalSongScoreWeights(DefaultSongScoreWeights())),
	}
}

// NewWithAdapters builds a Resolver from caller-provided album source and target adapters using the default ranking weights.
func NewWithAdapters(sources []SourceAdapter, targets []TargetAdapter) *Resolver {
	return NewWithAdaptersAndWeights(sources, targets, DefaultScoreWeights())
}

// NewWithAdaptersAndWeights builds a Resolver from caller-provided album source and target adapters and explicit ranking weights.
func NewWithAdaptersAndWeights(sources []SourceAdapter, targets []TargetAdapter, weights ScoreWeights) *Resolver {
	return NewWithEntityAdaptersAndWeights(sources, targets, nil, nil, weights, DefaultSongScoreWeights())
}

// NewWithEntityAdapters builds a Resolver from caller-provided album and song adapters using default ranking weights.
func NewWithEntityAdapters(albumSources []SourceAdapter, albumTargets []TargetAdapter, songSources []SongSourceAdapter, songTargets []SongTargetAdapter) *Resolver {
	return NewWithEntityAdaptersAndWeights(albumSources, albumTargets, songSources, songTargets, DefaultScoreWeights(), DefaultSongScoreWeights())
}

// NewWithEntityAdaptersAndWeights builds a Resolver from caller-provided album and song adapters and explicit ranking weights.
func NewWithEntityAdaptersAndWeights(albumSources []SourceAdapter, albumTargets []TargetAdapter, songSources []SongSourceAdapter, songTargets []SongTargetAdapter, albumWeights ScoreWeights, songWeights SongScoreWeights) *Resolver {
	return &Resolver{
		inner:     resolve.New(wrapSourceAdapters(albumSources), wrapTargetAdapters(albumTargets), toInternalScoreWeights(albumWeights)),
		songInner: resolve.NewSongs(wrapSongSourceAdapters(songSources), wrapSongTargetAdapters(songTargets), toInternalSongScoreWeights(songWeights)),
	}
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

// ResolveSong resolves one input song URL into a canonical source song plus per-service matches.
func (r *Resolver) ResolveSong(ctx context.Context, inputURL string) (*SongResolution, error) {
	resolution, err := r.songInner.ResolveSong(ctx, inputURL)
	if err != nil {
		//nolint:wrapcheck // Preserve the underlying resolver error for callers and CLI output.
		return nil, err
	}
	public := fromInternalSongResolution(*resolution)
	return &public, nil
}

// Resolve parses an input URL and dispatches to album or song resolution.
func (r *Resolver) Resolve(ctx context.Context, inputURL string) (*EntityResolution, error) {
	songResolution, err := r.ResolveSong(ctx, inputURL)
	if err == nil {
		return &EntityResolution{Parsed: songResolution.Parsed, Song: songResolution}, nil
	}
	if !errors.Is(err, ErrUnsupportedURL) && !errors.Is(err, ErrNoSourceAdapters) {
		return nil, err
	}

	albumResolution, albumErr := r.ResolveAlbum(ctx, inputURL)
	if albumErr != nil {
		//nolint:wrapcheck // Preserve the underlying resolver error for callers and CLI output.
		return nil, albumErr
	}
	return &EntityResolution{Parsed: ParsedURL(albumResolution.Parsed), Album: albumResolution}, nil
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

func defaultSongSourceAdapters(client *http.Client, config Config) []resolve.SongSourceAdapter {
	appleMusic := newAppleMusicAdapter(client, config)
	deezer := deezeradapter.New(client)
	spotify := spotifyadapter.New(client, spotifyadapter.WithCredentials(config.Spotify.ClientID, config.Spotify.ClientSecret))
	tidal := tidaladapter.New(client, tidaladapter.WithCredentials(config.TIDAL.ClientID, config.TIDAL.ClientSecret))

	candidates := []any{appleMusic, deezer, spotify, tidal}
	sources := make([]resolve.SongSourceAdapter, 0, len(candidates))
	for _, candidate := range candidates {
		adapter, ok := candidate.(resolve.SongSourceAdapter)
		if !ok {
			continue
		}
		sources = append(sources, adapter)
	}
	return sources
}

func defaultSongTargetAdapters(client *http.Client, config Config) []resolve.SongTargetAdapter {
	appleMusic := newAppleMusicAdapter(client, config)
	deezer := deezeradapter.New(client)
	candidates := []any{appleMusic, deezer}
	if config.SpotifyEnabled() {
		candidates = append(candidates, spotifyadapter.New(client, spotifyadapter.WithCredentials(config.Spotify.ClientID, config.Spotify.ClientSecret)))
	}
	if config.TIDALEnabled() {
		candidates = append(candidates, tidaladapter.New(client, tidaladapter.WithCredentials(config.TIDAL.ClientID, config.TIDAL.ClientSecret)))
	}

	allowed := map[ServiceName]struct{}{}
	if len(config.TargetServices) > 0 {
		allowed = make(map[ServiceName]struct{}, len(config.TargetServices))
		for _, service := range config.TargetServices {
			allowed[service] = struct{}{}
		}
	}

	targets := make([]resolve.SongTargetAdapter, 0, len(candidates))
	for _, candidate := range candidates {
		adapter, ok := candidate.(resolve.SongTargetAdapter)
		if !ok {
			continue
		}
		if len(allowed) > 0 {
			if _, ok := allowed[fromInternalServiceName(adapter.Service())]; !ok {
				continue
			}
		}
		targets = append(targets, adapter)
	}
	return targets
}

func wrapSourceAdapters(sources []SourceAdapter) []resolve.SourceAdapter {
	wrapped := make([]resolve.SourceAdapter, 0, len(sources))
	for _, source := range sources {
		wrapped = append(wrapped, sourceAdapterBridge{source: source})
	}
	return wrapped
}

func wrapSongSourceAdapters(sources []SongSourceAdapter) []resolve.SongSourceAdapter {
	wrapped := make([]resolve.SongSourceAdapter, 0, len(sources))
	for _, source := range sources {
		wrapped = append(wrapped, songSourceAdapterBridge{source: source})
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

func wrapSongTargetAdapters(targets []SongTargetAdapter) []resolve.SongTargetAdapter {
	wrapped := make([]resolve.SongTargetAdapter, 0, len(targets))
	for _, target := range targets {
		wrapped = append(wrapped, songTargetAdapterBridge{target: target})
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

type songSourceAdapterBridge struct {
	source SongSourceAdapter
}

func (b songSourceAdapterBridge) Service() model.ServiceName {
	return toInternalServiceName(b.source.Service())
}

func (b songSourceAdapterBridge) ParseSongURL(raw string) (*model.ParsedAlbumURL, error) {
	parsed, err := b.source.ParseSongURL(raw)
	if err != nil || parsed == nil {
		if err != nil {
			//nolint:wrapcheck // Preserve adapter parse errors without adding another wrapper layer.
			return nil, err
		}
		return nil, errSourceAdapterReturnedNilParsed
	}
	internal := toInternalParsedAlbumURL(ParsedAlbumURL(*parsed))
	return &internal, nil
}

func (b songSourceAdapterBridge) FetchSong(ctx context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalSong, error) {
	song, err := b.source.FetchSong(ctx, ParsedURL(fromInternalParsedAlbumURL(parsed)))
	if err != nil || song == nil {
		if err != nil {
			//nolint:wrapcheck // Preserve adapter fetch errors without adding another wrapper layer.
			return nil, err
		}
		return nil, errSourceAdapterReturnedNilSong
	}
	internal := toInternalCanonicalSong(*song)
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

type songTargetAdapterBridge struct {
	target SongTargetAdapter
}

func (b songTargetAdapterBridge) Service() model.ServiceName {
	return toInternalServiceName(b.target.Service())
}

func (b songTargetAdapterBridge) SearchSongByISRC(ctx context.Context, isrc string) ([]model.CandidateSong, error) {
	songs, err := b.target.SearchSongByISRC(ctx, isrc)
	if err != nil {
		//nolint:wrapcheck // Preserve target adapter errors without adding another wrapper layer.
		return nil, err
	}
	return toInternalCandidateSongs(songs), nil
}

func (b songTargetAdapterBridge) SearchSongByMetadata(ctx context.Context, song model.CanonicalSong) ([]model.CandidateSong, error) {
	songs, err := b.target.SearchSongByMetadata(ctx, fromInternalCanonicalSong(song))
	if err != nil {
		//nolint:wrapcheck // Preserve target adapter errors without adding another wrapper layer.
		return nil, err
	}
	return toInternalCandidateSongs(songs), nil
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

func toInternalSongScoreWeights(weights SongScoreWeights) score.SongWeights {
	return score.SongWeights{
		ISRCExact:            weights.ISRCExact,
		TitleExact:           weights.TitleExact,
		CoreTitleExact:       weights.CoreTitleExact,
		PrimaryArtistExact:   weights.PrimaryArtistExact,
		ArtistOverlap:        weights.ArtistOverlap,
		DurationNear:         weights.DurationNear,
		AlbumTitleExact:      weights.AlbumTitleExact,
		ReleaseDateExact:     weights.ReleaseDateExact,
		ReleaseYearExact:     weights.ReleaseYearExact,
		TrackNumberExact:     weights.TrackNumberExact,
		ExplicitMismatch:     weights.ExplicitMismatch,
		EditionMismatch:      weights.EditionMismatch,
		EditionMarkerPenalty: weights.EditionMarkerPenalty,
	}
}

func fromInternalSongScoreWeights(weights score.SongWeights) SongScoreWeights {
	return SongScoreWeights{
		ISRCExact:            weights.ISRCExact,
		TitleExact:           weights.TitleExact,
		CoreTitleExact:       weights.CoreTitleExact,
		PrimaryArtistExact:   weights.PrimaryArtistExact,
		ArtistOverlap:        weights.ArtistOverlap,
		DurationNear:         weights.DurationNear,
		AlbumTitleExact:      weights.AlbumTitleExact,
		ReleaseDateExact:     weights.ReleaseDateExact,
		ReleaseYearExact:     weights.ReleaseYearExact,
		TrackNumberExact:     weights.TrackNumberExact,
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

func toInternalCanonicalSong(song CanonicalSong) model.CanonicalSong {
	return model.CanonicalSong{
		Service:                toInternalServiceName(song.Service),
		SourceID:               song.SourceID,
		SourceURL:              song.SourceURL,
		RegionHint:             song.RegionHint,
		Title:                  song.Title,
		NormalizedTitle:        song.NormalizedTitle,
		Artists:                append([]string(nil), song.Artists...),
		NormalizedArtists:      append([]string(nil), song.NormalizedArtists...),
		DurationMS:             song.DurationMS,
		ISRC:                   song.ISRC,
		Explicit:               song.Explicit,
		DiscNumber:             song.DiscNumber,
		TrackNumber:            song.TrackNumber,
		AlbumID:                song.AlbumID,
		AlbumTitle:             song.AlbumTitle,
		AlbumNormalizedTitle:   song.AlbumNormalizedTitle,
		AlbumArtists:           append([]string(nil), song.AlbumArtists...),
		AlbumNormalizedArtists: append([]string(nil), song.AlbumNormalizedArtists...),
		ReleaseDate:            song.ReleaseDate,
		ArtworkURL:             song.ArtworkURL,
		EditionHints:           append([]string(nil), song.EditionHints...),
	}
}

func fromInternalCanonicalSong(song model.CanonicalSong) CanonicalSong {
	return CanonicalSong{
		Service:                fromInternalServiceName(song.Service),
		SourceID:               song.SourceID,
		SourceURL:              song.SourceURL,
		RegionHint:             song.RegionHint,
		Title:                  song.Title,
		NormalizedTitle:        song.NormalizedTitle,
		Artists:                append([]string(nil), song.Artists...),
		NormalizedArtists:      append([]string(nil), song.NormalizedArtists...),
		DurationMS:             song.DurationMS,
		ISRC:                   song.ISRC,
		Explicit:               song.Explicit,
		DiscNumber:             song.DiscNumber,
		TrackNumber:            song.TrackNumber,
		AlbumID:                song.AlbumID,
		AlbumTitle:             song.AlbumTitle,
		AlbumNormalizedTitle:   song.AlbumNormalizedTitle,
		AlbumArtists:           append([]string(nil), song.AlbumArtists...),
		AlbumNormalizedArtists: append([]string(nil), song.AlbumNormalizedArtists...),
		ReleaseDate:            song.ReleaseDate,
		ArtworkURL:             song.ArtworkURL,
		EditionHints:           append([]string(nil), song.EditionHints...),
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

func toInternalCandidateSong(song CandidateSong) model.CandidateSong {
	return model.CandidateSong{
		CanonicalSong: toInternalCanonicalSong(song.CanonicalSong),
		CandidateID:   song.CandidateID,
		MatchURL:      song.MatchURL,
	}
}

func fromInternalCandidateSong(song model.CandidateSong) CandidateSong {
	return CandidateSong{
		CanonicalSong: fromInternalCanonicalSong(song.CanonicalSong),
		CandidateID:   song.CandidateID,
		MatchURL:      song.MatchURL,
	}
}

func toInternalCandidateSongs(songs []CandidateSong) []model.CandidateSong {
	if len(songs) == 0 {
		return nil
	}
	internal := make([]model.CandidateSong, 0, len(songs))
	for _, song := range songs {
		internal = append(internal, toInternalCandidateSong(song))
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

func fromInternalSongScoredMatch(match resolve.SongScoredMatch) SongScoredMatch {
	return SongScoredMatch{
		URL:       match.URL,
		Score:     match.Score,
		Reasons:   append([]string(nil), match.Reasons...),
		Candidate: fromInternalCandidateSong(match.Candidate),
	}
}

func fromInternalSongMatchResult(result resolve.SongMatchResult) SongMatchResult {
	public := SongMatchResult{
		Service:    fromInternalServiceName(result.Service),
		Alternates: make([]SongScoredMatch, 0, len(result.Alternates)),
	}
	if result.Best != nil {
		best := fromInternalSongScoredMatch(*result.Best)
		public.Best = &best
	}
	for _, alternate := range result.Alternates {
		public.Alternates = append(public.Alternates, fromInternalSongScoredMatch(alternate))
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

func fromInternalSongResolution(resolution resolve.SongResolution) SongResolution {
	matches := make(map[ServiceName]SongMatchResult, len(resolution.Matches))
	for service, match := range resolution.Matches {
		matches[fromInternalServiceName(service)] = fromInternalSongMatchResult(match)
	}
	return SongResolution{
		InputURL: resolution.InputURL,
		Parsed:   ParsedURL(fromInternalParsedAlbumURL(resolution.Parsed)),
		Source:   fromInternalCanonicalSong(resolution.Source),
		Matches:  matches,
	}
}
