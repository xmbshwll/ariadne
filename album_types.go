package ariadne

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

// ParsedAlbumURL keeps album-specific APIs readable while sharing the common parsed URL shape.
type ParsedAlbumURL = ParsedURL

// ParsedSongURL keeps song-specific APIs readable while sharing the common parsed URL shape.
type ParsedSongURL = ParsedURL

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
