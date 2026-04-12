package bandcamp

import (
	"encoding/json"
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
	bandcampSearchPath          = "/search"
	bandcampSourceFixture       = "testdata/source-page.html"
	bandcampBrokenSchemaFixture = "testdata/broken-schema-page.html"
	lonAbatyAbbeyRoad           = "Lôn Abaty / Abbey Road"
	bandcampBaseURLPlaceholder  = "{{BASE_URL}}"
)

func newBandcampTestServer(buildRoutes func(baseURL string) map[string][]byte) *httptest.Server {
	var routes map[string][]byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := routes[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(body)
	}))
	routes = buildRoutes(server.URL)
	return server
}

func newBandcampTestAdapter(server *httptest.Server) *Adapter {
	return New(server.Client(), WithSearchBaseURL(server.URL))
}

func newBandcampAlbumSource(baseURL, slug string) model.ParsedAlbumURL {
	path := "/album/" + slug
	return model.ParsedAlbumURL{
		Service:      model.ServiceBandcamp,
		EntityType:   "album",
		ID:           slug,
		CanonicalURL: baseURL + path,
		RawURL:       baseURL + path,
	}
}

func newBandcampSongSource(baseURL, slug string) model.ParsedURL {
	path := "/track/" + slug
	return model.ParsedURL{
		Service:      model.ServiceBandcamp,
		EntityType:   "song",
		ID:           slug,
		CanonicalURL: baseURL + path,
		RawURL:       baseURL + path,
	}
}

func mustBandcampAlbumPage(t *testing.T, title string, artist string, releaseDate string, tracks []string) []byte {
	t.Helper()
	items := make([]map[string]any, 0, len(tracks))
	for i, track := range tracks {
		items = append(items, map[string]any{
			"@type":    "ListItem",
			"position": i + 1,
			"item": map[string]any{
				"@type":    "MusicRecording",
				"name":     track,
				"duration": "P00H03M00S",
			},
		})
	}
	return mustBandcampSchemaPage(t, map[string]any{
		"@context":      "https://schema.org",
		"@type":         "MusicAlbum",
		"@id":           "https://example.bandcamp.com/album/test",
		"name":          title,
		"datePublished": releaseDate + " 00:00:00 GMT",
		"image":         "https://f4.bcbits.com/img/example.jpg",
		"byArtist": map[string]any{
			"@type": "MusicGroup",
			"name":  artist,
		},
		"publisher": map[string]any{
			"@type": "MusicGroup",
			"name":  artist,
		},
		"track": map[string]any{
			"@type":           "ItemList",
			"numberOfItems":   len(tracks),
			"itemListElement": items,
		},
	})
}

func mustBandcampTrackPage(t *testing.T, title string, artist string, albumTitle string, releaseDate string, durationMS int) []byte {
	t.Helper()
	return mustBandcampSchemaPage(t, map[string]any{
		"@context":      "https://schema.org",
		"@type":         "MusicRecording",
		"@id":           "https://example.bandcamp.com/track/test",
		"name":          title,
		"datePublished": releaseDate + " 00:00:00 GMT",
		"duration":      fmt.Sprintf("PT%dM%dS", durationMS/60000, (durationMS/1000)%60),
		"image":         "https://f4.bcbits.com/img/example-track.jpg",
		"byArtist": map[string]any{
			"@type": "MusicGroup",
			"name":  artist,
		},
		"publisher": map[string]any{
			"@type": "MusicGroup",
			"name":  artist,
		},
		"inAlbum": map[string]any{
			"@type": "MusicAlbum",
			"@id":   "https://example.bandcamp.com/album/example-album",
			"name":  albumTitle,
		},
	})
}

func mustBandcampSchemaPage(t *testing.T, payload map[string]any) []byte {
	t.Helper()
	content, err := json.Marshal(payload)
	require.NoError(t, err)
	page := fmt.Sprintf("<html><body><script type=\"application/ld+json\">%s</script></body></html>", content)
	return []byte(page)
}

func mustReadBandcampSourcePage(t *testing.T) []byte {
	t.Helper()
	return mustReadTestFile(t, bandcampSourceFixture)
}

func mustRenderBandcampFixture(t *testing.T, relativePath, baseURL string) []byte {
	t.Helper()
	content := string(mustReadTestFile(t, relativePath))
	content = strings.ReplaceAll(content, bandcampBaseURLPlaceholder, baseURL)
	return []byte(content)
}

func brokenBandcampSchemaBody(t *testing.T) []byte {
	t.Helper()
	return mustReadTestFile(t, bandcampBrokenSchemaFixture)
}

func mustReadTestFile(t *testing.T, relativePath string) []byte {
	t.Helper()
	path := filepath.Clean(relativePath)
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return content
}
