package soundcloud

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	soundCloudCatsAndDogs = "Cats & Dogs"
	soundCloudTrackISRC   = "USBWK1100093"
	soundCloudAssetPath   = "/assets/app.js"
	soundCloudAlbumSearch = "/search/playlists"
	soundCloudSongSearch  = "/search/tracks"
)

type testFixture struct {
	adapter       *Adapter
	server        *httptest.Server
	sourcePayload []byte
	trackPayload  []byte
}

func newTestFixture(t *testing.T) testFixture {
	t.Helper()

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
		case soundCloudAssetPath:
			_, _ = w.Write([]byte(`window.__sc_config={client_id:"` + clientID + `"};`))
		case "/album":
			_, _ = fmt.Fprintf(w, `<html><body><script>window.__sc_hydration = [{"hydratable":"playlist","data":%s}];</script></body></html>`, sourcePayload)
		case "/track":
			_, _ = fmt.Fprintf(w, `<html><body><script>window.__sc_hydration = [{"hydratable":"sound","data":%s}];</script></body></html>`, trackPayload)
		case soundCloudAlbumSearch:
			if r.URL.Query().Get("client_id") != clientID {
				http.Error(w, "missing client id", http.StatusUnauthorized)
				return
			}
			_, _ = w.Write(searchPayload)
		case soundCloudSongSearch:
			if r.URL.Query().Get("client_id") != clientID {
				http.Error(w, "missing client id", http.StatusUnauthorized)
				return
			}
			_, _ = w.Write(trackSearchPayload)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	sourcePayload = []byte(strings.ReplaceAll(
		string(sourcePayload),
		"https://soundcloud.com/evidence-official/sets/cats-dogs-6",
		server.URL+"/album",
	))
	trackPayload = []byte(strings.ReplaceAll(
		string(trackPayload),
		"https://soundcloud.com/evidence-official/the-liner-notes-feat-aloe-1",
		server.URL+"/track",
	))

	return testFixture{
		adapter:       New(server.Client(), WithSiteBaseURL(server.URL), WithAPIBaseURL(server.URL)),
		server:        server,
		sourcePayload: sourcePayload,
		trackPayload:  trackPayload,
	}
}

func mustReadSoundCloudFixture(t *testing.T, relativePath string) []byte {
	t.Helper()
	path := filepath.Clean(relativePath)
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return content
}
