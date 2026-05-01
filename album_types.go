package ariadne

import (
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/resolve"
)

// ParsedURL is the normalized form of a parsed source URL.
type ParsedURL = model.ParsedURL

// ParsedAlbumURL keeps album-specific APIs readable while sharing the common parsed URL shape.
type ParsedAlbumURL = model.ParsedAlbumURL

// ParsedSongURL keeps song-specific APIs readable while sharing the common parsed URL shape.
type ParsedSongURL = model.ParsedURL

// CanonicalTrack is the normalized track representation shared across services.
type CanonicalTrack = model.CanonicalTrack

// CanonicalAlbum is the normalized album representation shared across services.
type CanonicalAlbum = model.CanonicalAlbum

// CandidateAlbum is one service-specific search result mapped into canonical form.
type CandidateAlbum = model.CandidateAlbum

// ScoredMatch is one ranked candidate returned by the album resolver.
type ScoredMatch = resolve.ScoredMatch
