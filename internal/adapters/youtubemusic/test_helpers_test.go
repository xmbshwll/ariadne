package youtubemusic

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
)

const (
	youtubeMusicBrowsePath        = "/browse/MPREb_tQfaWH32ovE"
	youtubeMusicSearchPath        = "/search"
	youtubeMusicBrokenPageHTML    = `<html><head></head><body>broken</body></html>`
	youtubeMusicSourceFixturePath = "testdata/source-page.html"
	youtubeMusicSearchFixturePath = "testdata/search-page.html"
	youtubeMusicAbbeyRoadTitle    = "Abbey Road"
	youtubeMusicAbbeyRoadArtist   = "The Beatles"
)

type youTubeMusicSearchResult struct {
	Title    string
	BrowseID string
	Artist   string
}

func newYouTubeMusicTestServer(routes map[string][]byte) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := routes[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(body)
	}))
}

func newYouTubeMusicTestAdapter(server *httptest.Server) *Adapter {
	return New(server.Client(), WithBaseURL(server.URL))
}

func newYouTubeMusicAlbumSource(baseURL string) model.ParsedAlbumURL {
	return model.ParsedAlbumURL{
		Service:      model.ServiceYouTubeMusic,
		EntityType:   "album",
		ID:           "MPREb_tQfaWH32ovE",
		CanonicalURL: baseURL + youtubeMusicBrowsePath,
	}
}

func youTubeMusicAbbeyRoadAlbum() model.CanonicalAlbum {
	return model.CanonicalAlbum{
		Title:   youtubeMusicAbbeyRoadTitle,
		Artists: []string{youtubeMusicAbbeyRoadArtist},
	}
}

func mustReadYouTubeMusicSourcePage(t *testing.T) []byte {
	t.Helper()
	return mustReadYouTubeMusicFixture(t, youtubeMusicSourceFixturePath)
}

func mustReadYouTubeMusicSearchPage(t *testing.T) []byte {
	t.Helper()
	return mustReadYouTubeMusicFixture(t, youtubeMusicSearchFixturePath)
}

func youTubeMusicBrokenBrowsePage() []byte {
	return []byte(youtubeMusicBrokenPageHTML)
}

func youTubeMusicAlbumSearchPage(results ...youTubeMusicSearchResult) []byte {
	parts := make([]string, 0, len(results))
	for _, result := range results {
		parts = append(parts, youTubeMusicAlbumSearchResult(result))
	}
	return []byte(strings.Join(parts, " "))
}

func youTubeMusicAlbumSearchResult(result youTubeMusicSearchResult) string {
	return fmt.Sprintf(
		`title\x22:\x7b\x22runs\x22:\x5b\x7b\x22text\x22:\x22%s\x22,\x22navigationEndpoint\x22:\x7b anything browseId\x22:\x22%s\x22 anything pageType\x22:\x22MUSIC_PAGE_TYPE_ALBUM\x22 anything subtitle\x22:\x7b\x22runs\x22:\x5b\x7b\x22text\x22:\x22Album\x22\x7d,\x7b\x22text\x22:\x22 · \x22\x7d,\x7b\x22text\x22:\x22%s\x22`,
		result.Title,
		result.BrowseID,
		result.Artist,
	)
}

func mustReadYouTubeMusicFixture(t *testing.T, relativePath string) []byte {
	t.Helper()
	path := filepath.Clean(relativePath)
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return content
}
