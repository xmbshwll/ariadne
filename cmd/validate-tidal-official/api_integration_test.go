package main

// White-box: the tool's own orchestration (token, album document, identifier
// searches) is its product; the shared seams it sits on are tested in
// cmd/internal/validation.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tidaladapter "github.com/xmbshwll/ariadne/internal/adapters/tidal"
)

const tidalAlbumFixture = `{
	"data": {
		"id": "12047952",
		"attributes": {"title": "Abbey Road", "barcodeId": "00602567713449", "releaseDate": "1969-09-26"},
		"relationships": {"artists": {"data": [{"id": "a1"}]}}
	},
	"included": [
		{"id": "a1", "type": "artists", "attributes": {"name": "The Beatles"}},
		{"id": "t1", "type": "tracks", "attributes": {"title": "Come Together", "isrc": "GBAYE0601690"}}
	]
}`

func tidalFixtureServer(t *testing.T, failStage string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth2/token", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"access_token":"token-1"}`))
	})
	mux.HandleFunc("/v2/albums/", func(w http.ResponseWriter, _ *http.Request) {
		if failStage == "album" {
			http.Error(w, "album boom", http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(tidalAlbumFixture))
	})
	mux.HandleFunc("/v2/searchResults", func(w http.ResponseWriter, _ *http.Request) {
		if failStage == "search" {
			http.Error(w, "search boom", http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"data":[]}`))
	})
	mux.HandleFunc("/v2/tracks", func(w http.ResponseWriter, _ *http.Request) {
		if failStage == "isrc" {
			http.Error(w, "isrc boom", http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"data":[]}`))
	})
	return httptest.NewServer(mux)
}

func tidalValidationInputs(t *testing.T, serverURL string) validationInputs {
	t.Helper()
	parsed, err := tidaladapter.ParseAlbumURL("https://tidal.com/album/12047952")
	require.NoError(t, err)
	return validationInputs{
		opts:        options{apiBaseURL: serverURL + "/v2", authBaseURL: serverURL},
		rawURL:      "https://tidal.com/album/12047952",
		outputDir:   t.TempDir(),
		parsed:      parsed,
		countryCode: "US",
	}
}

func TestCollectValidationArtifacts(t *testing.T) {
	tests := []struct {
		name        string
		failStage   string
		wantErrText string
		wantTitle   string
	}{
		{
			name:      "collects the album payload and identifier searches",
			wantTitle: "Abbey Road",
		},
		{
			name:        "an album fetch failure aborts collection",
			failStage:   "album",
			wantErrText: "unexpected tidal api status",
		},
		{
			name:        "a search failure aborts collection",
			failStage:   "search",
			wantErrText: "search tidal albums",
		},
		{
			name:        "an isrc search failure aborts collection",
			failStage:   "isrc",
			wantErrText: "search tidal tracks by isrc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tidalFixtureServer(t, tt.failStage)
			defer server.Close()

			artifacts, err := collectValidationArtifacts(context.Background(), tidalValidationInputs(t, server.URL))

			if tt.wantErrText != "" {
				require.ErrorContains(t, err, tt.wantErrText, tt.name)
				return
			}
			require.NoError(t, err, tt.name)

			assert.Equal(t, tt.wantTitle, artifacts.summary["title"], tt.name)
			assert.Equal(t, "00602567713449", artifacts.summary["upc"], tt.name)
			assert.NotEmpty(t, artifacts.targets["source-payload-official.json"], tt.name)
			assert.NotEmpty(t, artifacts.targets["search-albums-official.json"], tt.name)
		})
	}
}

// TestBuildTIDALQueryPinsTheMetadataFallback keeps the album-id fallback when
// the document carries no usable title.
func TestBuildTIDALQueryPinsTheMetadataFallback(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		artists []string
		albumID string
		want    string
	}{
		{name: "title and artist join into the query", title: "Abbey Road", artists: []string{"The Beatles"}, albumID: "12047952", want: "Abbey Road The Beatles"},
		{name: "an empty document falls back to the album id", title: "", artists: nil, albumID: "12047952", want: "12047952"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, buildTIDALQuery(tt.title, tt.artists, tt.albumID), tt.name)
		})
	}
}
