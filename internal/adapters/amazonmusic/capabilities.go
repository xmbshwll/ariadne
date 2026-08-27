package amazonmusic

import (
	"github.com/xmbshwll/ariadne/internal/adapters"
)

// Every provider satisfies the same Adapter interface; what differs is only
// Capabilities, and the methods this package leaves to base.Unsupported.
var _ adapters.Adapter = (*Adapter)(nil)

// Capabilities reports which Amazon Music Adapter methods Ariadne can use.
//
// Source Input parsing for both Entity Shapes; album and song fetching are Runtime Deferred and Amazon Music is not a Target Search service.
func Capabilities() adapters.Capabilities {
	return adapters.Capabilities{AlbumSource: true, SongSource: true}
}

// Capabilities reports which Amazon Music Adapter methods Ariadne can use.
func (a *Adapter) Capabilities() adapters.Capabilities {
	return Capabilities()
}
