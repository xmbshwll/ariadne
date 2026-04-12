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
	youtubeMusicBrowsePath      = "/browse/MPREb_tQfaWH32ovE"
	youtubeMusicSearchPath      = "/search"
	youtubeMusicBrokenPageHTML  = `<html><head></head><body>broken</body></html>`
	youtubeMusicAbbeyRoadTitle  = "Abbey Road"
	youtubeMusicAbbeyRoadArtist = "The Beatles"
)

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

func youTubeMusicAbbeyRoadAlbum() model.CanonicalAlbum {
	return model.CanonicalAlbum{
		Title:   youtubeMusicAbbeyRoadTitle,
		Artists: []string{youtubeMusicAbbeyRoadArtist},
	}
}

func youTubeMusicSearchPage(results ...string) []byte {
	return []byte(strings.Join(results, " "))
}

func youTubeMusicAlbumSearchResult(title string, browseID string, artist string) string {
	return fmt.Sprintf(
		`title\x22:\x7b\x22runs\x22:\x5b\x7b\x22text\x22:\x22%s\x22,\x22navigationEndpoint\x22:\x7b anything browseId\x22:\x22%s\x22 anything pageType\x22:\x22MUSIC_PAGE_TYPE_ALBUM\x22 anything subtitle\x22:\x7b\x22runs\x22:\x5b\x7b\x22text\x22:\x22Album\x22\x7d,\x7b\x22text\x22:\x22 · \x22\x7d,\x7b\x22text\x22:\x22%s\x22`,
		title,
		browseID,
		artist,
	)
}

func mustReadYouTubeMusicFixture(t *testing.T, relativePath string) []byte {
	t.Helper()
	path := filepath.Clean(relativePath)
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return content
}
