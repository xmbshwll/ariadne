package soundcloud_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	soundcloud "github.com/xmbshwll/ariadne/internal/adapters/soundcloud"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xmbshwll/ariadne/internal/adapters"
	"github.com/xmbshwll/ariadne/internal/model"
)

func TestIdentifierSongSearchIsUnsupported(t *testing.T) {
	fixture := newTestFixture(t)

	_, err := fixture.adapter.SearchSongByISRC(context.Background(), "GBAYE0601690")
	assert.ErrorIs(t, err, adapters.ErrUnsupported)
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
			require.Equal(t, clientID, r.URL.Query().Get("client_id"))
			_, _ = w.Write([]byte(`{"collection":[{"title":"Broken Track","permalink_url":"","user":{"username":"Artist"}},{"title":"Good Track","permalink_url":"` + server.URL + `/artist/good-track","user":{"username":"Artist"}}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := soundcloud.New(server.Client(), soundcloud.WithSiteBaseURL(server.URL), soundcloud.WithAPIBaseURL(server.URL))
	results, err := adapter.SearchSongByMetadata(context.Background(), model.CanonicalSong{Title: "Good Track", Artists: []string{"Artist"}})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, server.URL+"/artist/good-track", results[0].MatchURL)
}

func TestSearchSongByMetadataFindsClientIDInLaterScriptAsset(t *testing.T) {
	const clientID = "33333333333333333333333333333333"
	const assetCount = 11
	const clientIDAssetPath = "/assets/11.js"

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/":
			var html strings.Builder
			html.WriteString("<html><body>")
			for i := 1; i <= assetCount; i++ {
				_, _ = fmt.Fprintf(&html, `<script src="%s/assets/%d.js"></script>`, server.URL, i)
			}
			html.WriteString("</body></html>")
			_, _ = w.Write([]byte(html.String()))
		case r.URL.Path == clientIDAssetPath:
			_, _ = w.Write([]byte(`window.__sc_config={client_id:"` + clientID + `"};`))
		case strings.HasPrefix(r.URL.Path, "/assets/"):
			_, _ = w.Write([]byte(`console.log("noop")`))
		case r.URL.Path == soundCloudSongSearch:
			require.Equal(t, clientID, r.URL.Query().Get("client_id"))
			_, _ = w.Write([]byte(`{"collection":[{"title":"Good Track","permalink_url":"` + server.URL + `/artist/good-track","user":{"username":"Artist"}}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := soundcloud.New(server.Client(), soundcloud.WithSiteBaseURL(server.URL), soundcloud.WithAPIBaseURL(server.URL))
	results, err := adapter.SearchSongByMetadata(context.Background(), model.CanonicalSong{Title: "Good Track", Artists: []string{"Artist"}})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, server.URL+"/artist/good-track", results[0].MatchURL)
}

func TestSearchSongByMetadataReturnsNoResultsWhenClientIDUnavailable(t *testing.T) {
	server := newSoundCloudServerWithoutClientID()
	defer server.Close()

	adapter := soundcloud.New(server.Client(), soundcloud.WithSiteBaseURL(server.URL), soundcloud.WithAPIBaseURL(server.URL))
	results, err := adapter.SearchSongByMetadata(context.Background(), model.CanonicalSong{Title: "FENIAN", Artists: []string{"KNEECAP"}})

	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestToCanonicalSongLeavesAlbumArtistsEmptyWithoutAlbumTitle(t *testing.T) {
	song := soundcloud.ToCanonicalSong(soundcloud.SoundTrack{
		Title:        "Loose Track",
		PermalinkURL: "https://soundcloud.com/example/loose-track",
		User:         soundcloud.SoundUser{Username: "Example Artist"},
		PublisherMetadata: soundcloud.PublisherMetadata{
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
