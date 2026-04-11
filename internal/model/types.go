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

// ParsedURL is the normalized output of a service-specific entity URL parser.
type ParsedURL struct {
	Service      ServiceName
	EntityType   string
	ID           string
	CanonicalURL string
	RegionHint   string
	RawURL       string
}

// ParsedAlbumURL keeps album-specific APIs readable while sharing the common parsed URL shape.
type ParsedAlbumURL = ParsedURL

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

// CanonicalSong is the normalized song representation shared across services.
type CanonicalSong struct {
	Service                ServiceName
	SourceID               string
	SourceURL              string
	RegionHint             string
	Title                  string
	NormalizedTitle        string
	Artists                []string
	NormalizedArtists      []string
	DurationMS             int
	ISRC                   string
	Explicit               bool
	DiscNumber             int
	TrackNumber            int
	AlbumID                string
	AlbumTitle             string
	AlbumNormalizedTitle   string
	AlbumArtists           []string
	AlbumNormalizedArtists []string
	ReleaseDate            string
	ArtworkURL             string
	EditionHints           []string
}

// CandidateAlbum is a service-specific search result converted into canonical form.
type CandidateAlbum struct {
	CanonicalAlbum
	CandidateID string
	MatchURL    string
}

// CandidateSong is a service-specific song search result converted into canonical form.
type CandidateSong struct {
	CanonicalSong
	CandidateID string
	MatchURL    string
}
