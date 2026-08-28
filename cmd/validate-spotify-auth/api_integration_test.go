package main

// White-box: the tool's own orchestration (token, album fetch, identifier
// collection, three searches) is its product; the shared seams it sits on are
// tested in cmd/internal/validation.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	spotifyadapter "github.com/xmbshwll/ariadne/internal/adapters/spotify"
	"github.com/xmbshwll/ariadne/internal/model"
)

// spotifyFixtureServer serves the token endpoint plus every endpoint the tool
// hits, so collectValidationArtifacts runs end to end against canned payloads.
func spotifyFixtureServer(t *testing.T, albumBody string, failStage string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"access_token":"token-1"}`))
	})
	mux.HandleFunc("/v1/albums/", func(w http.ResponseWriter, _ *http.Request) {
		if failStage == "album" {
			http.Error(w, "album boom", http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(albumBody))
	})
	mux.HandleFunc("/v1/tracks/", func(w http.ResponseWriter, r *http.Request) {
		trackID := r.URL.Path[len("/v1/tracks/"):]
		_, _ = w.Write([]byte(`{"external_ids":{"isrc":"GBAYE060169` + trackID + `"}}`))
	})
	mux.HandleFunc("/v1/search", func(w http.ResponseWriter, _ *http.Request) {
		if failStage == "search" {
			http.Error(w, "search boom", http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"albums":{"items":[]}}`))
	})
	return httptest.NewServer(mux)
}

const spotifyAlbumFixture = `{
	"name":"Abbey Road",
	"release_date":"1969-09-26",
	"label":"Apple Records",
	"external_ids":{"upc":"00602567713449"},
	"artists":[{"name":"The Beatles"}],
	"tracks":{"items":[{"id":"1"},{"id":"2"}]}
}`

func mustParseSpotifyAlbum(t *testing.T, raw string) *model.ParsedAlbumURL {
	t.Helper()
	parsed, err := spotifyadapter.ParseAlbumURL(raw)
	require.NoError(t, err)
	return parsed
}

func spotifyValidationInputs(t *testing.T, serverURL string, outputDir string) validationInputs {
	t.Helper()
	opts := options{apiBaseURL: serverURL + "/v1", authBaseURL: serverURL + ""}
	return validationInputs{
		opts:      opts,
		rawURL:    "https://open.spotify.com/album/1DFixLWuPkv3KT3TnV35m3",
		outputDir: outputDir,
		parsed:    mustParseSpotifyAlbum(t, "https://open.spotify.com/album/1DFixLWuPkv3KT3TnV35m3"),
	}
}

func TestCollectValidationArtifacts(t *testing.T) {
	tests := []struct {
		name        string
		failStage   string
		wantErrText string
		wantUPC     string
	}{
		{
			name:    "collects the album payload and identifier searches",
			wantUPC: "00602567713449",
		},
		{
			name:        "an album fetch failure aborts collection",
			failStage:   "album",
			wantErrText: "unexpected spotify api status",
		},
		{
			name:        "a search failure aborts collection",
			failStage:   "search",
			wantErrText: "search spotify by upc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := spotifyFixtureServer(t, spotifyAlbumFixture, tt.failStage)
			defer server.Close()

			inputs := spotifyValidationInputs(t, server.URL, t.TempDir())
			artifacts, err := collectValidationArtifacts(context.Background(), inputs)

			if tt.wantErrText != "" {
				require.ErrorContains(t, err, tt.wantErrText, tt.name)
				return
			}
			require.NoError(t, err, tt.name)

			assert.Equal(t, tt.wantUPC, artifacts.summary["upc"], tt.name)
			var album spotifyAlbumPayload
			require.NoError(t, json.Unmarshal(artifacts.albumBody, &album))
			assert.Equal(t, "Abbey Road", album.Name, tt.name)
			assert.NotEmpty(t, artifacts.upcBody, tt.name)
			assert.NotEmpty(t, artifacts.isrcBody, tt.name)
			assert.NotEmpty(t, artifacts.metadataBody, tt.name)
		})
	}
}
