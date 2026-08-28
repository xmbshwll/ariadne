package deezer_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	deezer "github.com/xmbshwll/ariadne/internal/adapters/deezer"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xmbshwll/ariadne/internal/model"
)

const (
	deezerAlbumPath        = "/album/12047952"
	deezerAlbumTracksPath  = "/album/12047952/tracks"
	deezerAlbumSearchPath  = "/search/album"
	deezerTrackSearchPath  = "/search/track"
	deezerComeTogetherISRC = "GBAYE0601690"
)

// Deezer's fixture payloads live in testdata so the JSON is real, not Go.
var (
	deezerTrackSearchPayload       = deezerFixture("track-search.json")
	deezerComeTogetherTrackPayload = deezerFixture("track-come-together.json")
	deezerLiveTrackPayload         = deezerFixture("track-live.json")
	deezerSomethingTrackPayload    = deezerFixture("track-something.json")
)

func deezerFixture(name string) string {
	content, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		panic(err)
	}
	return string(content)
}

type jsonRoute struct {
	status int
	body   []byte
}

func newTestServer(t *testing.T, albumBytes, trackBytes, searchBytes []byte) *httptest.Server {
	t.Helper()

	srv := newJSONRouteServer(map[string]jsonRoute{
		deezerAlbumPath:                         jsonOK(albumBytes),
		"/album/upc:602547670342":               jsonOK(albumBytes),
		deezerAlbumTracksPath:                   jsonOK(trackBytes),
		deezerAlbumSearchPath:                   jsonOK(searchBytes),
		deezerTrackSearchPath:                   jsonOK([]byte(deezerTrackSearchPayload)),
		"/track/116348128":                      jsonOK([]byte(deezerComeTogetherTrackPayload)),
		"/track/999999":                         jsonOK([]byte(deezerLiveTrackPayload)),
		"/track/isrc:" + deezerComeTogetherISRC: jsonOK([]byte(deezerComeTogetherTrackPayload)),
		"/track/isrc:GBAYE0601691":              jsonOK([]byte(deezerSomethingTrackPayload)),
	})
	t.Cleanup(srv.Close)
	return srv
}

func newJSONTestServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		handler(w, r)
	}))
}

func newJSONRouteServer(routes map[string]jsonRoute) *httptest.Server {
	return newJSONTestServer(func(w http.ResponseWriter, r *http.Request) {
		route, ok := routes[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		if route.status != 0 && route.status != http.StatusOK {
			w.WriteHeader(route.status)
		}
		_, _ = w.Write(route.body)
	})
}

func jsonOK(body []byte) jsonRoute {
	return jsonRoute{status: http.StatusOK, body: body}
}

func jsonError(status int, body string) jsonRoute {
	return jsonRoute{status: status, body: []byte(body)}
}

func newTestAdapter(server *httptest.Server) *deezer.Adapter {
	return deezer.NewAdapter(server.Client(), server.URL)
}

func mustReadDeezerAlbumFixtures(t *testing.T) ([]byte, []byte) {
	t.Helper()
	return mustReadTestFile(t, "testdata/source-payload.json"), mustReadTestFile(t, "testdata/tracks.json")
}

func mustReadDeezerAlbumSearchFixture(t *testing.T) []byte {
	t.Helper()
	return mustReadTestFile(t, "testdata/search-album-single.json")
}

func assertSingleCandidate(t *testing.T, results []model.CandidateAlbum) {
	t.Helper()
	require.Len(t, results, 1)
	assert.Equal(t, "12047952", results[0].CandidateID)
	assert.Equal(t, "https://www.deezer.com/album/12047952", results[0].MatchURL)
	assert.Equal(t, "602547670342", results[0].UPC)
}

func assertSingleSongCandidate(t *testing.T, results []model.CandidateSong) {
	t.Helper()
	require.Len(t, results, 1)
	assert.Equal(t, "116348128", results[0].CandidateID)
	assert.Equal(t, "https://www.deezer.com/track/116348128", results[0].MatchURL)
	assert.Equal(t, deezerComeTogetherISRC, results[0].ISRC)
}

func mustReadTestFile(t *testing.T, relativePath string) []byte {
	t.Helper()
	path := filepath.Clean(relativePath)
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return content
}
