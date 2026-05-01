package ariadne

import (
	"github.com/xmbshwll/ariadne/internal/resolve"
)

// MatchResult is the ranked output for one target service.
type MatchResult = resolve.MatchResult

// SongMatchResult is the ranked song output for one target service.
type SongMatchResult = resolve.SongMatchResult

// Resolution is the full output of resolving one input album URL.
type Resolution = resolve.Resolution

// SongResolution is the full output of resolving one input song URL.
type SongResolution = resolve.SongResolution

// EntityResolution is the generic output of resolving one input URL.
type EntityResolution struct {
	// Parsed is the normalized parsed form of the source URL.
	Parsed ParsedURL
	// Album is set when the input resolved as an album.
	Album *Resolution
	// Song is set when the input resolved as a song.
	Song *SongResolution
}
