package youtubemusic

import (
	"github.com/xmbshwll/ariadne/internal/adapters"
)

// Every provider satisfies the same Adapter interface; what differs is only
// Capabilities, and the methods this package leaves to base.Unsupported.
var _ adapters.Adapter = (*Adapter)(nil)

// Capabilities reports which YouTube Music Adapter methods Ariadne can use.
//
// Source Input for both Entity Shapes and album metadata Target Search; song Source fetching stays Runtime Deferred and there is no song Target Search.
func Capabilities() adapters.Capabilities {
	return adapters.Capabilities{AlbumSource: true, SongSource: true, AlbumMetadata: true}
}

// Capabilities reports which YouTube Music Adapter methods Ariadne can use.
func (a *Adapter) Capabilities() adapters.Capabilities {
	return Capabilities()
}
