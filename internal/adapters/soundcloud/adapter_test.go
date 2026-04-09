package soundcloud

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
)

func TestAdapter(t *testing.T) {
	const (
		soundCloudCatsAndDogs = "Cats & Dogs"
		soundCloudTrackISRC   = "USBWK1100093"
	)

	sourcePayload := mustReadSoundCloudFixture(t, filepath.Join("testdata", "source-payload.json"))
	searchPayload := mustReadSoundCloudFixture(t, filepath.Join("testdata", "search-results.json"))
	trackPayload := mustReadSoundCloudFixture(t, filepath.Join("testdata", "track-payload.json"))
	trackSearchPayload := mustReadSoundCloudFixture(t, filepath.Join("testdata", "track-search-results.json"))
	clientID := "qNxp6KCjufkNWMIclTv0O4ycYGY0eFFX"

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			_, _ = fmt.Fprintf(w, `<html><body><script src="%s/assets/app.js"></script></body></html>`, server.URL)
		case "/assets/app.js":
			_, _ = w.Write([]byte(`window.__sc_config={client_id:"` + clientID + `"};`))
		case "/album":
			_, _ = fmt.Fprintf(w, `<html><body><script>window.__sc_hydration = [{"hydratable":"playlist","data":%s}];</script></body></html>`, sourcePayload)
		case "/track":
			_, _ = fmt.Fprintf(w, `<html><body><script>window.__sc_hydration = [{"hydratable":"sound","data":%s}];</script></body></html>`, trackPayload)
		case "/search/playlists":
			if r.URL.Query().Get("client_id") != clientID {
				http.Error(w, "missing client id", http.StatusUnauthorized)
				return
			}
			_, _ = w.Write(searchPayload)
		case "/search/tracks":
			if r.URL.Query().Get("client_id") != clientID {
				http.Error(w, "missing client id", http.StatusUnauthorized)
				return
			}
			_, _ = w.Write(trackSearchPayload)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	trackPayload = []byte(strings.ReplaceAll(
		string(trackPayload),
		"https://soundcloud.com/evidence-official/the-liner-notes-feat-aloe-1",
		server.URL+"/track",
	))

	adapter := New(server.Client(), WithSiteBaseURL(server.URL), WithAPIBaseURL(server.URL))
	parsed := model.ParsedAlbumURL{
		Service:      model.ServiceSoundCloud,
		EntityType:   "set",
		ID:           "evidence-official/sets/cats-dogs-6",
		CanonicalURL: server.URL + "/album",
		RawURL:       server.URL + "/album",
	}

	t.Run("fetch album from hydration", func(t *testing.T) {
		album, err := adapter.FetchAlbum(context.Background(), parsed)
		require.NoError(t, err)
		assert.Equal(t, soundCloudCatsAndDogs, album.Title)
		assert.Equal(t, "https://soundcloud.com/evidence-official/sets/cats-dogs-6", album.SourceURL)
		assert.Equal(t, 17, album.TrackCount)
		assert.Equal(t, "826257014467", album.UPC)
		require.NotEmpty(t, album.Tracks)
		assert.Equal(t, soundCloudTrackISRC, album.Tracks[0].ISRC)
		assert.Equal(t, "Rhymesayers", album.Label)
	})

	t.Run("search by metadata via api v2", func(t *testing.T) {
		results, err := adapter.SearchByMetadata(context.Background(), model.CanonicalAlbum{
			Title:   "Cats & Dogs",
			Artists: []string{"Evidence"},
		})
		require.NoError(t, err)
		require.Len(t, results, 5)
		assert.Equal(t, "evidence-official/sets/cats-dogs-3", results[0].CandidateID)
		assert.Equal(t, "evidence-official/sets/cats-dogs-6", results[1].CandidateID)
		assert.Equal(t, "USBWK1100093", results[1].Tracks[0].ISRC)
	})

	t.Run("fetch song from hydration", func(t *testing.T) {
		song, err := adapter.FetchSong(context.Background(), model.ParsedAlbumURL{
			Service:      model.ServiceSoundCloud,
			EntityType:   "song",
			ID:           "evidence-official/the-liner-notes-feat-aloe-1",
			CanonicalURL: server.URL + "/track",
			RawURL:       server.URL + "/track",
		})
		require.NoError(t, err)
		assert.Equal(t, "The Liner Notes (feat. Aloe Blacc)", song.Title)
		assert.Equal(t, soundCloudCatsAndDogs, song.AlbumTitle)
		assert.Equal(t, soundCloudTrackISRC, song.ISRC)
		assert.Equal(t, "https://i1.sndcdn.com/artworks-track-large.jpg", song.ArtworkURL)
	})

	t.Run("extract track hydration requires exact url match", func(t *testing.T) {
		body := fmt.Appendf(
			nil,
			`<html><body><script>window.__sc_hydration = [{"hydratable":"sound","data":%s}];</script></body></html>`,
			trackPayload,
		)
		track, err := extractTrackHydration(body, server.URL+"/missing-track")
		require.Error(t, err)
		assert.Nil(t, track)
		assert.ErrorIs(t, err, errSoundCloudTrackNotFound)
	})

	t.Run("search song by metadata via api v2", func(t *testing.T) {
		results, err := adapter.SearchSongByMetadata(context.Background(), model.CanonicalSong{
			Title:      "The Liner Notes",
			Artists:    []string{"Evidence"},
			AlbumTitle: soundCloudCatsAndDogs,
			DurationMS: 268706,
		})
		require.NoError(t, err)
		require.Len(t, results, 2)
		assert.Equal(t, "evidence-official/the-liner-notes-feat-aloe-1", results[0].CandidateID)
		assert.Equal(t, soundCloudCatsAndDogs, results[0].AlbumTitle)
	})

	t.Run("identifier search unsupported", func(t *testing.T) {
		upcResults, err := adapter.SearchByUPC(context.Background(), "826257014467")
		require.NoError(t, err)
		assert.Empty(t, upcResults)
		isrcResults, err := adapter.SearchByISRC(context.Background(), []string{soundCloudTrackISRC})
		require.NoError(t, err)
		assert.Empty(t, isrcResults)
		songISRCResults, err := adapter.SearchSongByISRC(context.Background(), soundCloudTrackISRC)
		require.NoError(t, err)
		assert.Empty(t, songISRCResults)
	})
}

func mustReadSoundCloudFixture(t *testing.T, relativePath string) []byte {
	t.Helper()
	path := filepath.Clean(relativePath)
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return content
}
