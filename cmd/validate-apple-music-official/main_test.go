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
			"-url", "https://music.apple.com/us/album/example/123",
			"-sample-url-file", "sample.txt",
			"-out-dir", "out",
			"-api-base-url", "https://api.music.test/v1",
			"-storefront", "gb",
		})
		require.NoError(t, err)
		assert.Equal(t, "https://music.apple.com/us/album/example/123", opts.sampleURL)
		assert.Equal(t, "sample.txt", opts.sampleURLPath)
		assert.Equal(t, "out", opts.outputDir)
		assert.Equal(t, "https://api.music.test/v1", opts.apiBaseURL)
		assert.Equal(t, "gb", opts.storefront)
	})

	t.Run("rejects positional args", func(t *testing.T) {
		t.Parallel()

		_, err := parseFlags([]string{"https://music.apple.com/us/album/example/123"})
		require.Error(t, err)
		assert.ErrorIs(t, err, errAppleMusicValidateUsage)
	})
}

func TestResolveStorefront(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		flagValue    string
		parsedRegion string
		configured   string
		want         string
	}{
		{name: "uses flag first", flagValue: "GB", parsedRegion: "us", configured: "ca", want: "gb"},
		{name: "falls back to parsed region", parsedRegion: "JP", configured: "ca", want: "jp"},
		{name: "falls back to config", configured: "DE", want: "de"},
		{name: "falls back to us", want: "us"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, resolveStorefront(tt.flagValue, tt.parsedRegion, tt.configured))
		})
	}
}

func TestAlbumISRCsAndNonEmptyStrings(t *testing.T) {
	t.Parallel()

	album := appleMusicAlbumResource{}
	album.Relationships.Tracks.Data = []appleMusicSongResource{
		{Attributes: appleMusicSongAttributes{ISRC: "ISRC001"}},
		{Attributes: appleMusicSongAttributes{ISRC: " ISRC001 "}},
		{Attributes: appleMusicSongAttributes{ISRC: "ISRC002"}},
		{Attributes: appleMusicSongAttributes{ISRC: "ISRC003"}},
		{Attributes: appleMusicSongAttributes{ISRC: "ISRC004"}},
		{Attributes: appleMusicSongAttributes{ISRC: "ISRC005"}},
		{Attributes: appleMusicSongAttributes{ISRC: "ISRC006"}},
		{Attributes: appleMusicSongAttributes{ISRC: ""}},
	}

	assert.Equal(t, []string{"ISRC001", "ISRC002", "ISRC003", "ISRC004", "ISRC005"}, albumISRCs(album))
	assert.Equal(t, []string{"Artist", "Guest"}, nonEmptyStrings(" ", "Artist", "", "Guest"))
}

func TestWriteValidationArtifacts(t *testing.T) {
	t.Parallel()

	outputDir := t.TempDir()
	err := writeValidationArtifacts(outputDir, validationArtifacts{
		albumBody:    []byte(`{"album":true}`),
		metadataBody: []byte(`{"metadata":true}`),
		upcBody:      []byte(`{"upc":true}`),
		isrcBody:     []byte(`{"isrc":true}`),
		summary:      map[string]any{"ok": true},
	})
	require.NoError(t, err)

	for _, name := range []string{
		"source-payload-official.json",
		"search-metadata-official.json",
		"search-upc-official.json",
		"search-isrc-official.json",
		"official-summary.json",
	} {
		path := filepath.Join(outputDir, name)
		content, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		assert.NotEmpty(t, content)
		assert.Equal(t, byte('\n'), content[len(content)-1])
	}
}
