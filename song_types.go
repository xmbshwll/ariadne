package ariadne

import (
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/resolve"
)

// CanonicalSong is the normalized song representation shared across services.
type CanonicalSong = model.CanonicalSong

// CandidateSong is one service-specific song search result mapped into canonical form.
type CandidateSong = model.CandidateSong

// SongScoredMatch is one ranked song candidate returned by the song resolver.
type SongScoredMatch = resolve.SongScoredMatch
