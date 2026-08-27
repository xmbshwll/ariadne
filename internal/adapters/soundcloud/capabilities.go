package soundcloud

import (
	"github.com/xmbshwll/ariadne/internal/adapters"
)

// Every provider satisfies the same Adapter interface; what differs is only
// Capabilities, and the methods this package leaves to base.Unsupported.
var _ adapters.Adapter = (*Adapter)(nil)

// Capabilities reports which SoundCloud Adapter methods Ariadne can use.
//
// Source Input for both Entity Shapes and metadata Target Search; SoundCloud exposes no UPC or ISRC lookup.
func Capabilities() adapters.Capabilities {
	return adapters.Capabilities{AlbumSource: true, SongSource: true, AlbumMetadata: true, SongMetadata: true}
}

// Capabilities reports which SoundCloud Adapter methods Ariadne can use.
func (a *Adapter) Capabilities() adapters.Capabilities {
	return Capabilities()
}
