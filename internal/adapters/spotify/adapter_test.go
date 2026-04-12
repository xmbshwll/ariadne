package spotify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
)

func TestFetchAlbumViaBootstrap(t *testing.T) {
	html := mustReadTestFile(t, "testdata/source-page.html")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/album/0ETFjACtuP2ADo6LFhL6HN" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(html)
	}))
	defer server.Close()

	adapter := New(server.Client(), WithWebBaseURL(server.URL))
	parsed := model.ParsedAlbumURL{
		Service:      model.ServiceSpotify,
		EntityType:   "album",
		ID:           "0ETFjACtuP2ADo6LFhL6HN",
		CanonicalURL: "https://open.spotify.com/album/0ETFjACtuP2ADo6LFhL6HN",
		RawURL:       "https://open.spotify.com/album/0ETFjACtuP2ADo6LFhL6HN",
	}

	album, err := adapter.FetchAlbum(context.Background(), parsed)
	require.NoError(t, err)
	require.NotNil(t, album)
	assert.Equal(t, "Abbey Road (Remastered)", album.Title)
	assert.NotEmpty(t, album.Label)
	assert.Equal(t, 17, album.TrackCount)
	require.Len(t, album.Tracks, 17)
	assert.Equal(t, "Come Together - Remastered 2009", album.Tracks[0].Title)
	assert.Equal(t, 259946, album.Tracks[0].DurationMS)
	assert.NotEmpty(t, album.ArtworkURL)
}
