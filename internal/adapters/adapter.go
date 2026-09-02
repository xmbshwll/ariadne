// Package adapters defines the one Module every Music Service adapter
// satisfies. Every built-in adapter implements the whole Adapter interface, so
// callers never ask whether a service happens to implement an extra interface:
// they read Capabilities to decide what to call, and any method a service does
// not support returns an error matching ErrUnsupported.
package adapters

import (
	"context"
	"errors"
	"fmt"

	"github.com/xmbshwll/ariadne/internal/model"
)

// ErrUnsupported identifies an Adapter method a Music Service does not support.
// Adapters return errors that match it, so callers can distinguish "this service
// cannot do this" from a transport or API failure.
var ErrUnsupported = errors.New("capability unsupported by service")

// Unsupported builds an ErrUnsupported error that names the service and the
// capability, for the case where the error reaches a user.
func Unsupported(service model.ServiceName, capability string) error {
	return fmt.Errorf("%s: %s: %w", service, capability, ErrUnsupported)
}

// Capabilities states which parts of Adapter a Music Service really implements.
// It is the data Target Search and the Provider Catalog act on, so capability
// support is declared once per service instead of inferred from Go interfaces.
type Capabilities struct {
	// AlbumSource reports that the service can parse and fetch album source URLs.
	AlbumSource bool
	// SongSource reports that the service can parse and fetch song source URLs.
	SongSource bool
	// AlbumUPC reports that the service can find albums by UPC.
	AlbumUPC bool
	// AlbumISRC reports that the service can find albums from track ISRCs.
	AlbumISRC bool
	// AlbumMetadata reports that the service can find albums by title and artist metadata.
	AlbumMetadata bool
	// SongISRC reports that the service can find songs by ISRC.
	SongISRC bool
	// SongMetadata reports that the service can find songs by title and artist metadata.
	SongMetadata bool
}

// SupportsAlbumTarget reports whether the service can act as an album Target.
func (c Capabilities) SupportsAlbumTarget() bool {
	return c.AlbumUPC || c.AlbumISRC || c.AlbumMetadata
}

// SupportsSongTarget reports whether the service can act as a song Target.
func (c Capabilities) SupportsSongTarget() bool {
	return c.SongISRC || c.SongMetadata
}

// SupportsAnyTarget reports whether the service can act as a Target at all.
func (c Capabilities) SupportsAnyTarget() bool {
	return c.SupportsAlbumTarget() || c.SupportsSongTarget()
}

// Adapter is the complete interface a Music Service adapter satisfies: Source
// Input parsing and fetching for both Entity Shapes, and Target Search by every
// identifier Ariadne can use. Methods a service does not support return errors
// matching ErrUnsupported and say so in Capabilities.
type Adapter interface {
	// Service names the Music Service this adapter talks to.
	Service() model.ServiceName
	// Capabilities reports which Adapter methods this service supports.
	Capabilities() Capabilities

	// ParseAlbumURL recognizes an album Source URL for this service.
	ParseAlbumURL(raw string) (*model.ParsedAlbumURL, error)
	// FetchAlbum reads the canonical album behind a parsed album Source URL.
	FetchAlbum(ctx context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalAlbum, error)

	// ParseSongURL recognizes a song Source URL for this service.
	ParseSongURL(raw string) (*model.ParsedURL, error)
	// FetchSong reads the canonical song behind a parsed song Source URL.
	FetchSong(ctx context.Context, parsed model.ParsedURL) (*model.CanonicalSong, error)

	// SearchAlbumByUPC finds albums by Universal Product Code.
	SearchAlbumByUPC(ctx context.Context, upc string) ([]model.CandidateAlbum, error)
	// SearchAlbumByISRC finds albums that contain any of the given track ISRCs.
	SearchAlbumByISRC(ctx context.Context, isrcs []string) ([]model.CandidateAlbum, error)
	// SearchAlbumByMetadata finds albums by title and artist metadata.
	SearchAlbumByMetadata(ctx context.Context, album model.CanonicalAlbum) ([]model.CandidateAlbum, error)

	// SearchSongByISRC finds songs by International Standard Recording Code.
	SearchSongByISRC(ctx context.Context, isrc string) ([]model.CandidateSong, error)
	// SearchSongByMetadata finds songs by title and artist metadata.
	SearchSongByMetadata(ctx context.Context, song model.CanonicalSong) ([]model.CandidateSong, error)
}
