package ariadne

import "context"

// SourceAdapter fetches canonical album metadata from a parsed source URL.
//
// Implementations must either return a parsed value or a non-nil error from ParseAlbumURL,
// and either a canonical album or a non-nil error from FetchAlbum. Returning nil with a nil
// error violates the adapter contract and is normalized to an exported ErrSourceAdapterReturnedNil* sentinel.
type SourceAdapter interface {
	Service() ServiceName
	ParseAlbumURL(raw string) (*ParsedAlbumURL, error)
	FetchAlbum(ctx context.Context, parsed ParsedAlbumURL) (*CanonicalAlbum, error)
}

// SongSourceAdapter fetches canonical song metadata from a parsed source URL.
//
// Implementations must either return a parsed value or a non-nil error from ParseSongURL,
// and either a canonical song or a non-nil error from FetchSong. Returning nil with a nil
// error violates the adapter contract and is normalized to an exported ErrSourceAdapterReturnedNil* sentinel.
type SongSourceAdapter interface {
	Service() ServiceName
	ParseSongURL(raw string) (*ParsedSongURL, error)
	FetchSong(ctx context.Context, parsed ParsedSongURL) (*CanonicalSong, error)
}

// TargetAdapter searches a target service for matching albums.
//
// Ariadne preserves adapter-returned errors under the resolver's context wrappers, so callers
// can still use errors.Is against adapter-defined sentinels.
type TargetAdapter interface {
	Service() ServiceName
	SearchByUPC(ctx context.Context, upc string) ([]CandidateAlbum, error)
	SearchByISRC(ctx context.Context, isrcs []string) ([]CandidateAlbum, error)
	SearchByMetadata(ctx context.Context, album CanonicalAlbum) ([]CandidateAlbum, error)
}

// SongTargetAdapter searches a target service for matching songs.
//
// Ariadne preserves adapter-returned errors under the resolver's context wrappers, so callers
// can still use errors.Is against adapter-defined sentinels.
type SongTargetAdapter interface {
	Service() ServiceName
	SearchSongByISRC(ctx context.Context, isrc string) ([]CandidateSong, error)
	SearchSongByMetadata(ctx context.Context, song CanonicalSong) ([]CandidateSong, error)
}
