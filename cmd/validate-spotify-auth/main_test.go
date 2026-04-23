package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFlags(t *testing.T) {
	t.Parallel()

	t.Run("parses explicit values", func(t *testing.T) {
		t.Parallel()

		opts, err := parseFlags([]string{
			"-url", "https://open.spotify.com/album/123",
			"-sample-url-file", "sample.txt",
			"-out-dir", "out",
			"-api-base-url", "https://api.spotify.test/v1",
			"-auth-base-url", "https://auth.spotify.test/api",
		})
		require.NoError(t, err)
		assert.Equal(t, "https://open.spotify.com/album/123", opts.sampleURL)
		assert.Equal(t, "sample.txt", opts.sampleURLPath)
		assert.Equal(t, "out", opts.outputDir)
		assert.Equal(t, "https://api.spotify.test/v1", opts.apiBaseURL)
		assert.Equal(t, "https://auth.spotify.test/api", opts.authBaseURL)
	})

	t.Run("rejects positional args", func(t *testing.T) {
		t.Parallel()

		_, err := parseFlags([]string{"https://open.spotify.com/album/123"})
		require.Error(t, err)
		assert.ErrorIs(t, err, errSpotifyValidateUsage)
	})
}

func TestMetadataQueryAndAlbumArtists(t *testing.T) {
	t.Parallel()

	album := spotifyAlbumPayload{
		Name:    " Fixture Album ",
		Artists: []spotifyArtist{{Name: " "}, {Name: "Fixture Artist"}, {Name: "Guest Artist"}},
	}

	assert.Equal(t, []string{"Fixture Artist", "Guest Artist"}, albumArtists(album))
	assert.Equal(t, "album:Fixture Album artist:Fixture Artist", metadataQuery(album))
	assert.Empty(t, metadataQuery(spotifyAlbumPayload{Name: "Fixture Album"}))
	assert.Empty(t, metadataQuery(spotifyAlbumPayload{Artists: []spotifyArtist{{Name: "Fixture Artist"}}}))
}

func TestNormalizeBaseURLAndAPIURL(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "https://api.spotify.test/v1", normalizeBaseURL("https://api.spotify.test/v1/"))
	assert.Equal(t, "https://api.spotify.test/v1/albums/123", apiURL("https://api.spotify.test/v1/", "/albums/123"))
}

func TestWriteValidationArtifacts(t *testing.T) {
	t.Parallel()

	outputDir := t.TempDir()
	err := writeValidationArtifacts(outputDir, validationArtifacts{
		albumBody:    []byte(`{"album":true}`),
		upcBody:      []byte(`{"upc":true}`),
		isrcBody:     []byte(`{"isrc":true}`),
		metadataBody: []byte(`{"metadata":true}`),
		summary:      map[string]any{"ok": true},
	})
	require.NoError(t, err)

	for _, name := range []string{
		"source-payload-api.json",
		"search-upc-results.json",
		"search-isrc-results.json",
		"search-metadata-results.json",
		"authenticated-summary.json",
	} {
		path := filepath.Join(outputDir, name)
		content, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		assert.NotEmpty(t, content)
		assert.Equal(t, byte('\n'), content[len(content)-1])
	}
}
