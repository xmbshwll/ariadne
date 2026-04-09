package ariadne

import (
	"context"
	"errors"

	amazonmusicadapter "github.com/xmbshwll/ariadne/internal/adapters/amazonmusic"
	applemusicadapter "github.com/xmbshwll/ariadne/internal/adapters/applemusic"
	spotifyadapter "github.com/xmbshwll/ariadne/internal/adapters/spotify"
	tidaladapter "github.com/xmbshwll/ariadne/internal/adapters/tidal"
	"github.com/xmbshwll/ariadne/internal/resolve"
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
