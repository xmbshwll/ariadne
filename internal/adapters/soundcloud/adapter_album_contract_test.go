package soundcloud_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	soundcloud "github.com/xmbshwll/ariadne/internal/adapters/soundcloud"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xmbshwll/ariadne/internal/adapters"
	"github.com/xmbshwll/ariadne/internal/model"
)

func TestIdentifierAlbumSearchIsUnsupported(t *testing.T) {
	fixture := newTestFixture(t)

	_, err := fixture.adapter.SearchAlbumByUPC(context.Background(), "00602537184945")
	assert.ErrorIs(t, err, adapters.ErrUnsupported)
	_, err = fixture.adapter.SearchAlbumByISRC(context.Background(), []string{"GBAYE0601690"})
	assert.ErrorIs(t, err, adapters.ErrUnsupported)
}

func TestSearchAlbumByMetadataSkipsMalformedHits(t *testing.T) {
	const clientID = "22222222222222222222222222222222"

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			_, _ = fmt.Fprintf(w, `<html><body><script src="%s%s"></script></body></html>`, server.URL, soundCloudAssetPath)
		case soundCloudAssetPath:
			_, _ = w.Write([]byte(`window.__sc_config={client_id:"` + clientID + `"};`))
		case soundCloudAlbumSearch:
			if r.URL.Query().Get("client_id") != clientID {
				http.Error(w, "invalid client_id", http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(`{"collection":[{"kind":"playlist","title":"Broken Playlist","permalink_url":"","user":{"username":"Artist"}},{"kind":"playlist","title":"Good Playlist","permalink_url":"` + server.URL + `/artist/sets/good-playlist","user":{"username":"Artist"}}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := soundcloud.New(server.Client(), soundcloud.WithSiteBaseURL(server.URL), soundcloud.WithAPIBaseURL(server.URL))
	results, err := adapter.SearchAlbumByMetadata(context.Background(), model.CanonicalAlbum{Title: "Good Playlist", Artists: []string{"Artist"}})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, server.URL+"/artist/sets/good-playlist", results[0].MatchURL)
}

func TestSearchAlbumByMetadataReturnsNoResultsWhenClientIDUnavailable(t *testing.T) {
	server := newSoundCloudServerWithoutClientID()
	defer server.Close()

	adapter := soundcloud.New(server.Client(), soundcloud.WithSiteBaseURL(server.URL), soundcloud.WithAPIBaseURL(server.URL))
	results, err := adapter.SearchAlbumByMetadata(context.Background(), model.CanonicalAlbum{Title: "FENIAN", Artists: []string{"KNEECAP"}})

	require.NoError(t, err)
	assert.Empty(t, results)
}

func newSoundCloudServerWithoutClientID() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			_, _ = w.Write([]byte(`<html><body><script src="/assets/app.js"></script></body></html>`))
		case "/assets/app.js":
			_, _ = w.Write([]byte(`console.log("no public client id")`))
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestSearchAlbumByMetadataRefreshesRejectedClientID(t *testing.T) {
	searchPayload := mustReadSoundCloudFixture(t, "testdata/search-results.json")
	const staleClientID = "11111111111111111111111111111111"
	const freshClientID = "22222222222222222222222222222222"

	assetRequests := 0
	searchRequests := 0

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			_, _ = fmt.Fprintf(w, `<html><body><script src="%s%s"></script></body></html>`, server.URL, soundCloudAssetPath)
		case soundCloudAssetPath:
			assetRequests++
			clientID := staleClientID
			if assetRequests > 1 {
				clientID = freshClientID
			}
			_, _ = w.Write([]byte(`window.__sc_config={client_id:"` + clientID + `"};`))
		case soundCloudAlbumSearch:
			searchRequests++
			if r.URL.Query().Get("client_id") != freshClientID {
				http.Error(w, "invalid client_id", http.StatusUnauthorized)
				return
			}
			_, _ = w.Write(searchPayload)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := soundcloud.New(server.Client(), soundcloud.WithSiteBaseURL(server.URL), soundcloud.WithAPIBaseURL(server.URL))
	results, err := adapter.SearchAlbumByMetadata(context.Background(), model.CanonicalAlbum{
		Title:   soundCloudCatsAndDogs,
		Artists: []string{"Evidence"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, results)
	assert.Equal(t, 2, assetRequests)
	assert.Equal(t, 2, searchRequests)
}
