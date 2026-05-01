package ariadne

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/resolve"
)

func TestConversionSliceContracts(t *testing.T) {
	identity := func(value string) string { return value }

	assert.Nil(t, cloneStrings(nil))
	assert.Nil(t, translateSlice[string, string](nil, identity))
	assert.Nil(t, translateNonEmptySlice[string, string]([]string{}, identity))

	clonedEmpty := cloneStrings([]string{})
	assert.NotNil(t, clonedEmpty)
	assert.Empty(t, clonedEmpty)

	translatedEmpty := translateSliceToEmpty[string, string](nil, identity)
	assert.NotNil(t, translatedEmpty)
	assert.Empty(t, translatedEmpty)

	values := []string{"one"}
	cloned := cloneStrings(values)
	values[0] = "mutated"
	assert.Equal(t, []string{"one"}, cloned)
}

func TestCanonicalAlbumTranslationDeepCopiesSlices(t *testing.T) {
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
	album.Artists[0] = "mutated artist"
	album.NormalizedArtists[0] = "mutated normalized artist"
	album.EditionHints[0] = "mutated edition"
	album.Tracks[0].Title = "Mutated Track"
	album.Tracks[0].Artists[0] = "Mutated Track Artist"

	assert.Equal(t, "Original Artist", internal.Artists[0])
	assert.Equal(t, "original artist", internal.NormalizedArtists[0])
	assert.Equal(t, "deluxe", internal.EditionHints[0])
	assert.Equal(t, "Original Track", internal.Tracks[0].Title)
	assert.Equal(t, "Track Artist", internal.Tracks[0].Artists[0])

	public := fromInternalCanonicalAlbum(internal)
	internal.Artists[0] = "mutated internal artist"
	internal.Tracks[0].Artists[0] = "mutated internal track artist"

	assert.Equal(t, "Original Artist", public.Artists[0])
	assert.Equal(t, "Track Artist", public.Tracks[0].Artists[0])
}

func TestCanonicalSongTranslationDeepCopiesSlices(t *testing.T) {
	song := CanonicalSong{
		Service:                ServiceSpotify,
		Artists:                []string{"Song Artist"},
		NormalizedArtists:      []string{"song artist"},
		AlbumArtists:           []string{"Album Artist"},
		AlbumNormalizedArtists: []string{"album artist"},
		EditionHints:           []string{"single"},
	}

	internal := toInternalCanonicalSong(song)
	song.Artists[0] = "mutated song artist"
	song.AlbumArtists[0] = "mutated album artist"
	song.EditionHints[0] = "mutated edition"

	assert.Equal(t, "Song Artist", internal.Artists[0])
	assert.Equal(t, "Album Artist", internal.AlbumArtists[0])
	assert.Equal(t, "single", internal.EditionHints[0])

	public := fromInternalCanonicalSong(internal)
	internal.Artists[0] = "mutated internal song artist"
	internal.AlbumArtists[0] = "mutated internal album artist"

	assert.Equal(t, "Song Artist", public.Artists[0])
	assert.Equal(t, "Album Artist", public.AlbumArtists[0])
}

func TestCandidateBatchTranslationPreservesEmptyAsNil(t *testing.T) {
	assert.Nil(t, toInternalCandidateAlbums(nil))
	assert.Nil(t, toInternalCandidateAlbums([]CandidateAlbum{}))
	assert.Nil(t, toInternalCandidateSongs(nil))
	assert.Nil(t, toInternalCandidateSongs([]CandidateSong{}))
}

func TestResultTranslationUsesEmptyOutputContainers(t *testing.T) {
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
