package ariadne

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

// CandidateSong is one service-specific song search result mapped into canonical form.
type CandidateSong struct {
	CanonicalSong
	// CandidateID is the service-specific identifier for the search result.
	CandidateID string
	// MatchURL is the service URL that should be presented for this candidate.
	MatchURL string
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
