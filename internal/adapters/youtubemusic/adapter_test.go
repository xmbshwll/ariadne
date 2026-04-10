package youtubemusic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
)

func TestAdapter(t *testing.T) {
	sourcePage := mustReadYouTubeMusicFixture(t, filepath.Join("testdata", "source-page.html"))
	searchPage := mustReadYouTubeMusicFixture(t, filepath.Join("testdata", "search-page.html"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/browse/MPREb_tQfaWH32ovE":
			_, _ = w.Write(sourcePage)
		case "/playlist":
			_, _ = w.Write(sourcePage)
		case "/search":
			_, _ = w.Write(searchPage)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := New(server.Client(), WithBaseURL(server.URL))

	t.Run("fetch album from browse page", func(t *testing.T) {
		album, err := adapter.FetchAlbum(context.Background(), model.ParsedAlbumURL{
			Service:      model.ServiceYouTubeMusic,
			EntityType:   "album",
			ID:           "MPREb_tQfaWH32ovE",
			CanonicalURL: server.URL + "/browse/MPREb_tQfaWH32ovE",
		})
		require.NoError(t, err)
		require.NotNil(t, album)
		assert.Equal(t, "Abbey Road (Super Deluxe Edition)", album.Title)
		assert.Equal(t, "https://music.youtube.com/playlist?list=OLAK5uy_lqcFZTOPHGwcnP0nYMzNuY0IES0fl7Fe4", album.SourceURL)
		assert.Equal(t, "OLAK5uy_lqcFZTOPHGwcnP0nYMzNuY0IES0fl7Fe4", album.SourceID)
		assert.Equal(t, []string{"The Beatles"}, album.Artists)
		assert.NotZero(t, album.TrackCount)
		require.NotEmpty(t, album.Tracks)
		assert.Equal(t, "Come Together (2019 Mix)", album.Tracks[0].Title)
		assert.NotEmpty(t, album.ArtworkURL)
	})

	t.Run("search by metadata hydrates browse result", func(t *testing.T) {
		results, err := adapter.SearchByMetadata(context.Background(), model.CanonicalAlbum{
			Title:   "Abbey Road",
			Artists: []string{"The Beatles"},
		})
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, "OLAK5uy_lqcFZTOPHGwcnP0nYMzNuY0IES0fl7Fe4", results[0].CandidateID)
		assert.Equal(t, "Abbey Road (Super Deluxe Edition)", results[0].Title)
		assert.NotEmpty(t, results[0].Tracks)
	})

	t.Run("identifier search unsupported", func(t *testing.T) {
		upcResults, err := adapter.SearchByUPC(context.Background(), "123")
		require.NoError(t, err)
		assert.Empty(t, upcResults)
		isrcResults, err := adapter.SearchByISRC(context.Background(), []string{"ABC"})
		require.NoError(t, err)
		assert.Empty(t, isrcResults)
	})
}

func mustReadYouTubeMusicFixture(t *testing.T, relativePath string) []byte {
	t.Helper()
	path := filepath.Clean(relativePath)
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return content
}
