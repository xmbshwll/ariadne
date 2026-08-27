package applemusic

import (
	"github.com/xmbshwll/ariadne/internal/adapters"
)

// Every provider satisfies the same Adapter interface; what differs is only
// Capabilities, and the methods this package leaves to base.Unsupported.
var _ adapters.Adapter = (*Adapter)(nil)

// Capabilities reports which Apple Music Adapter methods Ariadne can use.
//
// Every capability: Source Input for both Entity Shapes, plus Target Search by UPC, ISRC and metadata.
func Capabilities() adapters.Capabilities {
	return adapters.Capabilities{AlbumSource: true, SongSource: true, AlbumUPC: true, AlbumISRC: true, AlbumMetadata: true, SongISRC: true, SongMetadata: true}
}

// Capabilities reports which Apple Music Adapter methods Ariadne can use.
func (a *Adapter) Capabilities() adapters.Capabilities {
	return Capabilities()
}
