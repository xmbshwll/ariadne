package ariadne

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/resolve"
)

// The tests in this file are the executable contract for
// conversion_semantics.go. They exercise each invariant against every helper
// that claims to enforce it.

func TestConversionSemantics_NilPreservation(t *testing.T) {
	// Invariant 1: owned helpers return nil for nil input.
	identity := func(value string) string { return value }

	assert.Nil(t, copySlice[string](nil))
	assert.Nil(t, copyStrings(nil))
	assert.Nil(t, translateOwnedSlice[string, string](nil, identity))
	assert.Nil(t, translateCompactSlice[string, string](nil, identity))
	// Compact also collapses empty to nil by design.
	assert.Nil(t, translateCompactSlice[string, string]([]string{}, identity))
}

func TestConversionSemantics_StableOutputContainer(t *testing.T) {
	// Invariant 2: stable helpers return non-nil, even for nil/empty input.
	identity := func(value string) string { return value }

	fromNil := translateStableSlice[string, string](nil, identity)
	assert.NotNil(t, fromNil)
	assert.Empty(t, fromNil)

	fromEmpty := translateStableSlice([]string{}, identity)
	assert.NotNil(t, fromEmpty)
	assert.Empty(t, fromEmpty)

	mapFromNil := translateStableServiceMap[int, int](nil, func(v int) int { return v })
	assert.NotNil(t, mapFromNil)
	assert.Empty(t, mapFromNil)

	// copyStrings preserves nil but must return a non-nil empty slice when
	// the caller explicitly passes one, so round-trips do not lose the
	// "empty but present" signal.
	copiedEmpty := copyStrings([]string{})
	assert.NotNil(t, copiedEmpty)
	assert.Empty(t, copiedEmpty)
}

func TestConversionSemantics_DeepCopyOwnership(t *testing.T) {
	// Invariant 3: mutations on one side of the seam are not observable on
	// the other.
	const mutatedMarker = "mutated"
	const mutatedInternalMarker = "mutated internal"

	values := []string{"one"}
	copied := copyStrings(values)
	values[0] = mutatedMarker
	assert.Equal(t, []string{"one"}, copied)

	album := CanonicalAlbum{
		Service:           ServiceSpotify,
		Artists:           []string{"Original Artist"},
		NormalizedArtists: []string{"original artist"},
		EditionHints:      []string{"deluxe"},
		Tracks: []CanonicalTrack{{
			Title:   "Original Track",
			Artists: []string{"Track Artist"},
		}},
	}
	internal := toInternalCanonicalAlbum(album)
	album.Artists[0] = mutatedMarker
	album.NormalizedArtists[0] = mutatedMarker
	album.EditionHints[0] = mutatedMarker
	album.Tracks[0].Title = mutatedMarker
	album.Tracks[0].Artists[0] = mutatedMarker

	assert.Equal(t, "Original Artist", internal.Artists[0])
	assert.Equal(t, "original artist", internal.NormalizedArtists[0])
	assert.Equal(t, "deluxe", internal.EditionHints[0])
	assert.Equal(t, "Original Track", internal.Tracks[0].Title)
	assert.Equal(t, "Track Artist", internal.Tracks[0].Artists[0])

	public := fromInternalCanonicalAlbum(internal)
	internal.Artists[0] = mutatedInternalMarker
	internal.Tracks[0].Artists[0] = mutatedInternalMarker
	assert.Equal(t, "Original Artist", public.Artists[0])
	assert.Equal(t, "Track Artist", public.Tracks[0].Artists[0])

	song := CanonicalSong{
		Service:                ServiceSpotify,
		Artists:                []string{"Song Artist"},
		NormalizedArtists:      []string{"song artist"},
		AlbumArtists:           []string{"Album Artist"},
		AlbumNormalizedArtists: []string{"album artist"},
		EditionHints:           []string{"single"},
	}
	internalSong := toInternalCanonicalSong(song)
	song.Artists[0] = mutatedMarker
	song.AlbumArtists[0] = mutatedMarker
	song.EditionHints[0] = mutatedMarker

	assert.Equal(t, "Song Artist", internalSong.Artists[0])
	assert.Equal(t, "Album Artist", internalSong.AlbumArtists[0])
	assert.Equal(t, "single", internalSong.EditionHints[0])

	publicSong := fromInternalCanonicalSong(internalSong)
	internalSong.Artists[0] = mutatedInternalMarker
	internalSong.AlbumArtists[0] = mutatedInternalMarker
	assert.Equal(t, "Song Artist", publicSong.Artists[0])
	assert.Equal(t, "Album Artist", publicSong.AlbumArtists[0])
}

func TestConversionSemantics_CompactCollapsesBatches(t *testing.T) {
	// Compact helper use site: request batches. Nil and empty map to nil so
	// the internal model sees one canonical "no items" form.
	assert.Nil(t, toInternalCandidateAlbums(nil))
	assert.Nil(t, toInternalCandidateAlbums([]CandidateAlbum{}))
	assert.Nil(t, toInternalCandidateSongs(nil))
	assert.Nil(t, toInternalCandidateSongs([]CandidateSong{}))
}

func TestConversionSemantics_ResolverResultsAreStable(t *testing.T) {
	// Stable helper use sites: Resolution.Matches, SongResolution.Matches,
	// MatchResult.Alternates, SongMatchResult.Alternates. Callers must be
	// able to range over these without nil checks regardless of whether the
	// underlying resolver produced anything.
	album := fromInternalResolution(resolve.Resolution{
		Matches: map[model.ServiceName]resolve.MatchResult{
			model.ServiceSpotify: {Service: model.ServiceSpotify},
		},
	})
	require.Contains(t, album.Matches, ServiceSpotify)
	assert.NotNil(t, album.Matches)
	assert.NotNil(t, album.Matches[ServiceSpotify].Alternates)
	assert.Empty(t, album.Matches[ServiceSpotify].Alternates)

	emptyAlbum := fromInternalResolution(resolve.Resolution{})
	assert.NotNil(t, emptyAlbum.Matches)
	assert.Empty(t, emptyAlbum.Matches)

	song := fromInternalSongResolution(resolve.SongResolution{
		Matches: map[model.ServiceName]resolve.SongMatchResult{
			model.ServiceAppleMusic: {Service: model.ServiceAppleMusic},
		},
	})
	require.Contains(t, song.Matches, ServiceAppleMusic)
	assert.NotNil(t, song.Matches)
	assert.NotNil(t, song.Matches[ServiceAppleMusic].Alternates)
	assert.Empty(t, song.Matches[ServiceAppleMusic].Alternates)

	emptySong := fromInternalSongResolution(resolve.SongResolution{})
	assert.NotNil(t, emptySong.Matches)
	assert.Empty(t, emptySong.Matches)
}
