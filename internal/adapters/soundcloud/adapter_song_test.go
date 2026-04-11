package soundcloud

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
)

func TestFetchSongFromHydration(t *testing.T) {
	fixture := newTestFixture(t)

	song, err := fixture.adapter.FetchSong(context.Background(), model.ParsedAlbumURL{
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

func TestIdentifierSongSearchIsUnsupported(t *testing.T) {
	fixture := newTestFixture(t)

	results, err := fixture.adapter.SearchSongByISRC(context.Background(), soundCloudTrackISRC)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestSearchSongByMetadataSkipsMalformedHits(t *testing.T) {
	const clientID = "22222222222222222222222222222222"

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			_, _ = fmt.Fprintf(w, `<html><body><script src="%s%s"></script></body></html>`, server.URL, soundCloudAssetPath)
		case soundCloudAssetPath:
			_, _ = w.Write([]byte(`window.__sc_config={client_id:"` + clientID + `"};`))
		case soundCloudSongSearch:
			_, _ = w.Write([]byte(`{"collection":[{"title":"Broken Track","permalink_url":"","user":{"username":"Artist"}},{"title":"Good Track","permalink_url":"` + server.URL + `/artist/good-track","user":{"username":"Artist"}}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := New(server.Client(), WithSiteBaseURL(server.URL), WithAPIBaseURL(server.URL))
	results, err := adapter.SearchSongByMetadata(context.Background(), model.CanonicalSong{Title: "Good Track", Artists: []string{"Artist"}})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, server.URL+"/artist/good-track", results[0].MatchURL)
}

func TestToCanonicalSongLeavesAlbumArtistsEmptyWithoutAlbumTitle(t *testing.T) {
	song := toCanonicalSong(soundTrack{
		Title:        "Loose Track",
		PermalinkURL: "https://soundcloud.com/example/loose-track",
		User:         soundUser{Username: "Example Artist"},
		PublisherMetadata: publisherMetadata{
			Artist:     "Example Artist",
			AlbumTitle: "",
		},
	})

	require.NotNil(t, song)
	assert.Empty(t, song.AlbumTitle)
	assert.Nil(t, song.AlbumArtists)
	assert.Nil(t, song.AlbumNormalizedArtists)
	assert.Equal(t, []string{"Example Artist"}, song.Artists)
}
