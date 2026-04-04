package model

// ServiceName identifies a music service supported by the resolver.
type ServiceName string

const (
	ServiceSpotify      ServiceName = "spotify"
	ServiceAppleMusic   ServiceName = "appleMusic"
	ServiceDeezer       ServiceName = "deezer"
	ServiceSoundCloud   ServiceName = "soundcloud"
	ServiceBandcamp     ServiceName = "bandcamp"
	ServiceYouTubeMusic ServiceName = "youtubeMusic"
	ServiceTIDAL        ServiceName = "tidal"
	ServiceAmazonMusic  ServiceName = "amazonMusic"
)

// ParsedAlbumURL is the normalized output of a service-specific album URL parser.
type ParsedAlbumURL struct {
	Service      ServiceName
	EntityType   string
	ID           string
	CanonicalURL string
	RegionHint   string
	RawURL       string
}

// CanonicalTrack is the normalized track representation shared across services.
type CanonicalTrack struct {
	DiscNumber      int
	TrackNumber     int
	Title           string
	NormalizedTitle string
	DurationMS      int
	ISRC            string
	Artists         []string
}

// CanonicalAlbum is the normalized album representation shared across services.
type CanonicalAlbum struct {
	Service           ServiceName
	SourceID          string
	SourceURL         string
	RegionHint        string
	Title             string
	NormalizedTitle   string
	Artists           []string
	NormalizedArtists []string
	ReleaseDate       string
	Label             string
	UPC               string
	TrackCount        int
	TotalDurationMS   int
	ArtworkURL        string
	Explicit          bool
	EditionHints      []string
	Tracks            []CanonicalTrack
}

// CandidateAlbum is a service-specific search result converted into canonical form.
type CandidateAlbum struct {
	CanonicalAlbum
	CandidateID string
	MatchURL    string
}
