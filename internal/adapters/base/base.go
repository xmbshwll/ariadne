// Package base supplies what adapter implementations share. Unsupported answers
// the whole Adapter interface with ErrUnsupported, so a provider embeds it and
// implements only the methods its API really offers: the methods a provider
// writes are then exactly the capabilities it reports.
package base

import (
	"context"

	"github.com/xmbshwll/ariadne/internal/adapters"
	"github.com/xmbshwll/ariadne/internal/model"
)

// Unsupported is the zero behavior of an Adapter: every capability answers that
// the service does not support it, tagged with the service name for error text.
// Embed it and override the methods the service supports.
type Unsupported struct {
	ServiceName model.ServiceName
}

// Compile-time proof that Unsupported alone satisfies the full Adapter contract
// apart from Service and Capabilities, which every provider states itself.
var _ interface {
	ParseAlbumURL(string) (*model.ParsedAlbumURL, error)
	FetchAlbum(context.Context, model.ParsedAlbumURL) (*model.CanonicalAlbum, error)
	ParseSongURL(string) (*model.ParsedURL, error)
	FetchSong(context.Context, model.ParsedURL) (*model.CanonicalSong, error)
	SearchAlbumByUPC(context.Context, string) ([]model.CandidateAlbum, error)
	SearchAlbumByISRC(context.Context, []string) ([]model.CandidateAlbum, error)
	SearchAlbumByMetadata(context.Context, model.CanonicalAlbum) ([]model.CandidateAlbum, error)
	SearchSongByISRC(context.Context, string) ([]model.CandidateSong, error)
	SearchSongByMetadata(context.Context, model.CanonicalSong) ([]model.CandidateSong, error)
} = Unsupported{}

// ParseAlbumURL reports that the service has no album Source Input.
func (u Unsupported) ParseAlbumURL(string) (*model.ParsedAlbumURL, error) {
	return nil, adapters.Unsupported(u.ServiceName, "album source")
}

// FetchAlbum reports that the service has no album Source Input.
func (u Unsupported) FetchAlbum(context.Context, model.ParsedAlbumURL) (*model.CanonicalAlbum, error) {
	return nil, adapters.Unsupported(u.ServiceName, "album source")
}

// ParseSongURL reports that the service has no song Source Input.
func (u Unsupported) ParseSongURL(string) (*model.ParsedURL, error) {
	return nil, adapters.Unsupported(u.ServiceName, "song source")
}

// FetchSong reports that the service has no song Source Input.
func (u Unsupported) FetchSong(context.Context, model.ParsedURL) (*model.CanonicalSong, error) {
	return nil, adapters.Unsupported(u.ServiceName, "song source")
}

// SearchAlbumByUPC reports that the service cannot search albums by UPC.
func (u Unsupported) SearchAlbumByUPC(context.Context, string) ([]model.CandidateAlbum, error) {
	return nil, adapters.Unsupported(u.ServiceName, "album UPC search")
}

// SearchAlbumByISRC reports that the service cannot search albums by ISRC.
func (u Unsupported) SearchAlbumByISRC(context.Context, []string) ([]model.CandidateAlbum, error) {
	return nil, adapters.Unsupported(u.ServiceName, "album ISRC search")
}

// SearchAlbumByMetadata reports that the service cannot search albums by metadata.
func (u Unsupported) SearchAlbumByMetadata(context.Context, model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
	return nil, adapters.Unsupported(u.ServiceName, "album metadata search")
}

// SearchSongByISRC reports that the service cannot search songs by ISRC.
func (u Unsupported) SearchSongByISRC(context.Context, string) ([]model.CandidateSong, error) {
	return nil, adapters.Unsupported(u.ServiceName, "song ISRC search")
}

// SearchSongByMetadata reports that the service cannot search songs by metadata.
func (u Unsupported) SearchSongByMetadata(context.Context, model.CanonicalSong) ([]model.CandidateSong, error) {
	return nil, adapters.Unsupported(u.ServiceName, "song metadata search")
}
