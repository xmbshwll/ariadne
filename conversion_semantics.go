package ariadne

import (
	"github.com/xmbshwll/ariadne/internal/model"
)

// Conversion Semantics Module.
//
// This file is the Interface for the public/internal conversion seam.
// conversions_album.go, conversions_song.go, and conversions.go use only the
// helpers declared here so every public value produced by the package obeys
// the same shape rules. Field mappings live next to their types; shape rules
// live here.
//
// Three invariants govern every translation helper:
//
//  1. Nil preservation. When a caller passes a nil slice to a helper that
//     translates "owned" caller state (CanonicalAlbum.Artists, request
//     batches, and so on), the output is also nil. This keeps round-trips
//     through the public API shape-preserving: nil goes in, nil comes out.
//     Owned helpers: copySlice, translateOwnedSlice, translateCompactSlice.
//
//  2. Stable output container. When the package owns the output shape
//     (resolver Matches maps, MatchResult.Alternates), the helper always
//     returns a non-nil container even when the input was nil or empty.
//     Callers of ResolveAlbum and ResolveSong can therefore range over
//     Matches and Alternates without nil checks.
//     Stable helpers: translateStableSlice, translateStableServiceMap.
//
//  3. Deep-copy ownership. Every slice crossing the seam is copied, so
//     mutations on one side cannot be observed on the other. Nested slices
//     (CanonicalTrack.Artists inside CanonicalAlbum.Tracks) are copied too
//     because the per-element translator itself runs the copy.
//     Enforced by: copySlice (directly) and the translate* helpers that
//     invoke element translators which in turn invoke copySlice.
//
// Compact helpers exist for request batches where an empty list and a
// missing list are semantically identical and we want the internal model to
// see one canonical form (nil).
// Compact helpers: translateCompactSlice.

// copySlice deep-copies a slice while preserving nil. It is the single place
// where ownership-transfer copies happen; all other helpers compose on top.
func copySlice[T any](values []T) []T {
	if values == nil {
		return nil
	}
	copied := make([]T, len(values))
	copy(copied, values)
	return copied
}

// copyStrings is a typed alias for the common []string case. It exists so
// field mappings read as copyStrings(artists) instead of copySlice[string].
func copyStrings(values []string) []string {
	return copySlice(values)
}

// translateOwnedSlice preserves nil inputs (invariant 1) and deep-copies via
// the element translator (invariant 3). Use for slices the caller owns.
func translateOwnedSlice[From any, To any](values []From, translate func(From) To) []To {
	if values == nil {
		return nil
	}
	return translateStableSlice(values, translate)
}

// translateStableSlice always returns a non-nil slice (invariant 2). Use when
// the package owns the output container and callers should be able to range
// over it without nil checks.
func translateStableSlice[From any, To any](values []From, translate func(From) To) []To {
	translated := make([]To, len(values))
	for i, value := range values {
		translated[i] = translate(value)
	}
	return translated
}

// translateCompactSlice collapses nil and empty inputs to a nil output. Use
// for request batches where "no items" has one canonical form on the
// internal side.
func translateCompactSlice[From any, To any](values []From, translate func(From) To) []To {
	if len(values) == 0 {
		return nil
	}
	return translateStableSlice(values, translate)
}

// translateStableServiceMap always returns a non-nil map (invariant 2) keyed
// by the public ServiceName. Used for resolver result containers so
// Resolution.Matches and SongResolution.Matches are always safe to range.
func translateStableServiceMap[From any, To any](values map[model.ServiceName]From, translate func(From) To) map[ServiceName]To {
	translated := make(map[ServiceName]To, len(values))
	for service, value := range values {
		translated[fromInternalServiceName(service)] = translate(value)
	}
	return translated
}
