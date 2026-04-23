package ariadne

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
	Parsed ParsedSongURL
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
