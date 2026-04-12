package deezer

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
)

const (
	deezerAlbumPath                = "/album/12047952"
	deezerAlbumTracksPath          = "/album/12047952/tracks"
	deezerComeTogetherISRC         = "GBAYE0601690"
	deezerTrackSearchPayload       = `{"data":[{"id":116348128,"title":"Come Together (Remastered 2009)"},{"id":999999,"title":"Come Together"}]}`
	deezerComeTogetherTrackPayload = `{"id":116348128,"title":"Come Together (Remastered 2009)","link":"https://www.deezer.com/track/116348128","isrc":"GBAYE0601690","album":{"id":12047952,"title":"Abbey Road (Remastered)","link":"https://www.deezer.com/album/12047952","cover_xl":"https://e-cdns-images.dzcdn.net/images/cover/test/1000x1000.jpg","release_date":"1969-09-26"},"artist":{"id":1,"name":"The Beatles"},"duration":258,"track_position":1,"disk_number":1,"explicit_lyrics":false}`
	deezerLiveTrackPayload         = `{"id":999999,"title":"Come Together","link":"https://www.deezer.com/track/999999","isrc":"OTHER0001","album":{"id":555,"title":"Abbey Road Live","link":"https://www.deezer.com/album/555","release_date":"2020-01-01"},"artist":{"id":2,"name":"Tribute Band"},"duration":200,"track_position":8,"disk_number":1,"explicit_lyrics":false}`
	deezerSomethingTrackPayload    = `{"id":116348454,"title":"Something (Remastered 2009)","link":"https://www.deezer.com/track/116348454","isrc":"GBAYE0601691","album":{"id":12047952,"title":"Abbey Road (Remastered)","link":"https://www.deezer.com/album/12047952","release_date":"1969-09-26"},"artist":{"id":1,"name":"The Beatles"},"duration":182,"track_position":2,"disk_number":1,"explicit_lyrics":false}`
)

func newTestServer(t *testing.T, albumBytes, trackBytes, searchBytes []byte) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case deezerAlbumPath:
			_, _ = w.Write(albumBytes)
		case "/album/upc:602547670342":
			_, _ = w.Write(albumBytes)
		case deezerAlbumTracksPath:
			_, _ = w.Write(trackBytes)
		case "/search/album":
			_, _ = w.Write(searchBytes)
		case "/search/track":
			_, _ = w.Write([]byte(deezerTrackSearchPayload))
		case "/track/116348128":
			_, _ = w.Write([]byte(deezerComeTogetherTrackPayload))
		case "/track/999999":
			_, _ = w.Write([]byte(deezerLiveTrackPayload))
		case "/track/isrc:" + deezerComeTogetherISRC:
			_, _ = w.Write([]byte(deezerComeTogetherTrackPayload))
		case "/track/isrc:GBAYE0601691":
			_, _ = w.Write([]byte(deezerSomethingTrackPayload))
		default:
			http.NotFound(w, r)
		}
	}))
}

func newTestAdapter(server *httptest.Server) *Adapter {
	adapter := New(server.Client())
	adapter.baseURL = server.URL
	return adapter
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
