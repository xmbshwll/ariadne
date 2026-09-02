package main

// White-box: the tool's own orchestration (album document, identifier and
// metadata searches) is its product; the shared seams it sits on are tested in
// cmd/internal/validation.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	applemusicadapter "github.com/xmbshwll/ariadne/internal/adapters/applemusic"
)

const appleMusicAlbumFixture = `{
	"data": [{
		"attributes": {
			"name": "Abbey Road",
			"artistName": "The Beatles",
			"releaseDate": "1969-09-26",
			"recordLabel": "Apple Records",
			"upc": "00602567713449"
		},
		"relationships": {
			"tracks": {"data": [
				{"attributes": {"isrc": "GBAYE0601690"}},
				{"attributes": {"isrc": "GBAYE0601691"}}
			]}
		}
	}]
}`

func appleMusicFixtureServer(t *testing.T, failStage string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/catalog/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/catalog/us/albums" && r.URL.Query().Get("filter[upc]") != "":
			if failStage == "upc" {
				http.Error(w, "upc boom", http.StatusInternalServerError)
				return
			}
			_, _ = w.Write([]byte(`{"data":[]}`))
		case r.URL.Path == "/catalog/us/songs":
			if failStage == "isrc" {
				http.Error(w, "isrc boom", http.StatusInternalServerError)
				return
			}
			_, _ = w.Write([]byte(`{"data":[]}`))
		case r.URL.Path == "/catalog/us/search":
			if failStage == "metadata" {
				http.Error(w, "metadata boom", http.StatusInternalServerError)
				return
			}
			_, _ = w.Write([]byte(`{"results":{}}`))
		default:
			if failStage == "album" {
				http.Error(w, "album boom", http.StatusInternalServerError)
				return
			}
			_, _ = w.Write([]byte(appleMusicAlbumFixture))
		}
	})
	return httptest.NewServer(mux)
}

func appleMusicValidationInputs(t *testing.T, serverURL string) validationInputs {
	t.Helper()
	parsed, err := applemusicadapter.ParseAlbumURL("https://music.apple.com/us/album/abbey-road/1441164426")
	require.NoError(t, err)
	return validationInputs{
		opts:           options{apiBaseURL: serverURL},
		rawURL:         "https://music.apple.com/us/album/abbey-road/1441164426",
		outputDir:      t.TempDir(),
		parsed:         parsed,
		storefront:     "us",
		developerToken: "token-1",
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
			wantErrText: "unexpected apple music api status",
		},
		{
			name:        "a metadata search failure aborts collection",
			failStage:   "metadata",
			wantErrText: "search official apple music metadata",
		},
		{
			name:        "a upc search failure aborts collection",
			failStage:   "upc",
			wantErrText: "search official apple music by upc",
		},
		{
			name:        "an isrc search failure aborts collection",
			failStage:   "isrc",
			wantErrText: "search official apple music by isrc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := appleMusicFixtureServer(t, tt.failStage)
			defer server.Close()

			artifacts, err := collectValidationArtifacts(context.Background(), appleMusicValidationInputs(t, server.URL))

			if tt.wantErrText != "" {
				require.ErrorContains(t, err, tt.wantErrText, tt.name)
				return
			}
			require.NoError(t, err, tt.name)

			assert.Equal(t, tt.wantUPC, artifacts.summary["upc"], tt.name)
			assert.NotEmpty(t, artifacts.albumBody, tt.name)
			assert.NotEmpty(t, artifacts.metadataBody, tt.name)
		})
	}
}
