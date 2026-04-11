package soundcloud

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
)

func TestFetchSongFromHydration(t *testing.T) {
	fixture := newTestFixture(t)

	song, err := fixture.adapter.FetchSong(context.Background(), model.ParsedURL{
		Service:      model.ServiceSoundCloud,
		EntityType:   "song",
		ID:           "evidence-official/the-liner-notes-feat-aloe-1",
		CanonicalURL: fixture.server.URL + "/track",
		RawURL:       fixture.server.URL + "/track",
	})
	require.NoError(t, err)
	assert.Equal(t, "The Liner Notes (feat. Aloe Blacc)", song.Title)
	assert.Equal(t, soundCloudCatsAndDogs, song.AlbumTitle)
	assert.Equal(t, soundCloudTrackISRC, song.ISRC)
	assert.Equal(t, "https://i1.sndcdn.com/artworks-track-large.jpg", song.ArtworkURL)
}

func TestExtractTrackHydrationRequiresExactURLMatch(t *testing.T) {
	fixture := newTestFixture(t)

	body := fmt.Appendf(
		nil,
		`<html><body><script>window.__sc_hydration = [{"hydratable":"sound","data":%s}];</script></body></html>`,
		fixture.trackPayload,
	)
	track, err := extractTrackHydration(body, fixture.server.URL+"/missing-track")
	require.Error(t, err)
	assert.Nil(t, track)
	assert.ErrorIs(t, err, errSoundCloudTrackNotFound)
}

func TestSearchSongByMetadata(t *testing.T) {
	fixture := newTestFixture(t)

	results, err := fixture.adapter.SearchSongByMetadata(context.Background(), model.CanonicalSong{
		Title:      "The Liner Notes",
		Artists:    []string{"Evidence"},
		AlbumTitle: soundCloudCatsAndDogs,
		DurationMS: 268706,
	})
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, "evidence-official/the-liner-notes-feat-aloe-1", results[0].CandidateID)
	assert.Equal(t, soundCloudCatsAndDogs, results[0].AlbumTitle)
}
