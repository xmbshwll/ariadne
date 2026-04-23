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
			"-url", "https://tidal.com/album/123",
			"-sample-url-file", "sample.txt",
			"-out-dir", "out",
			"-api-base-url", "https://openapi.tidal.test/v2",
			"-auth-base-url", "https://auth.tidal.test/v1",
			"-country-code", "gb",
		})
		require.NoError(t, err)
		assert.Equal(t, "https://tidal.com/album/123", opts.sampleURL)
		assert.Equal(t, "sample.txt", opts.sampleURLPath)
		assert.Equal(t, "out", opts.outputDir)
		assert.Equal(t, "https://openapi.tidal.test/v2", opts.apiBaseURL)
		assert.Equal(t, "https://auth.tidal.test/v1", opts.authBaseURL)
		assert.Equal(t, "gb", opts.countryCode)
	})

	t.Run("rejects positional args", func(t *testing.T) {
		t.Parallel()

		_, err := parseFlags([]string{"https://tidal.com/album/123"})
		require.Error(t, err)
		assert.ErrorIs(t, err, errTIDALValidateUsage)
	})
}

func TestBuildTIDALQuery(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Album Artist", buildTIDALQuery("Album", []string{"Artist", "Guest"}, "fallback-id"))
	assert.Equal(t, "Album", buildTIDALQuery("Album", nil, "fallback-id"))
	assert.Equal(t, "fallback-id", buildTIDALQuery("", nil, "fallback-id"))
}

func TestTIDALIncludedHelpers(t *testing.T) {
	t.Parallel()

	included := []tidalIncludedResource{
		{ID: "artist-1", Type: "artists", Attributes: tidalAttributes{Name: "Artist One"}},
		{ID: "artist-1-duplicate", Type: "artists", Attributes: tidalAttributes{Name: "Artist One"}},
		{ID: "artist-2", Type: "artists", Attributes: tidalAttributes{Title: "Artist Two"}},
		{ID: "artist-2", Type: "albums", Attributes: tidalAttributes{Title: "Wrong Collision", ISRC: "ISRC000"}},
		{ID: "track-1", Type: "tracks", Attributes: tidalAttributes{Title: "Track One", ISRC: "ISRC001"}},
		{ID: "track-2", Type: "tracks", Attributes: tidalAttributes{Name: "Track Two", ISRC: "ISRC001"}},
		{ID: "track-3", Type: "tracks", Attributes: tidalAttributes{Title: "Track Three", ISRC: "ISRC003"}},
	}

	relations := []tidalRelationshipData{{ID: "artist-2"}, {ID: "artist-1"}, {ID: "missing"}}

	assert.Equal(t, []string{"Artist One", "Artist Two"}, collectIncludedNames(included, "artists"))
	assert.Equal(t, []string{"Artist Two", "Artist One"}, collectRelationshipNames(relations, included))
	assert.Equal(t, []string{"Track One", "Track Two"}, collectIncludedTitles(included, "tracks", 2))
	assert.Equal(t, []string{"ISRC001", "ISRC003"}, collectIncludedValues(included, "tracks", 5, includedISRC))
	assert.Equal(t, "Artist One", firstArtist([]string{"Artist One", "Artist Two"}))
	assert.Empty(t, firstArtist(nil))
	assert.Equal(t, "value", firstNonEmpty(" ", "value", "other"))
	assert.Empty(t, firstNonEmpty("", "  "))
}

func TestWriteValidationArtifacts(t *testing.T) {
	t.Parallel()

	outputDir := t.TempDir()
	err := writeValidationArtifacts(outputDir, validationArtifacts{
		targets: map[string][]byte{
			"source-payload-official.json": []byte(`{"album":true}`),
			"search-albums-official.json":  []byte(`{"search":true}`),
		},
		summary: map[string]any{"ok": true},
	})
	require.NoError(t, err)

	for _, name := range []string{
		"source-payload-official.json",
		"search-albums-official.json",
		"official-summary.json",
	} {
		path := filepath.Join(outputDir, name)
		content, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		assert.NotEmpty(t, content)
		assert.Equal(t, byte('\n'), content[len(content)-1])
	}
}
