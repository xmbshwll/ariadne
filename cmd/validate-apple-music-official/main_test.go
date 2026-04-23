package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
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

func TestFetchAppleMusicISRCSearchAggregatesAllQueries(t *testing.T) {
	t.Parallel()

	queried := make([]string, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queried = append(queried, r.URL.Query().Get("filter[isrc]"))
		_, _ = fmt.Fprintf(w, `{"data":[{"id":%q}]}`, r.URL.Query().Get("filter[isrc]"))
	}))
	defer server.Close()

	inputs := validationInputs{
		opts:           options{apiBaseURL: server.URL},
		developerToken: "token",
		storefront:     "us",
	}

	body, err := fetchAppleMusicISRCSearch(context.Background(), server.Client(), inputs, []string{"ISRC001", "ISRC002"})
	require.NoError(t, err)
	assert.Equal(t, []string{"ISRC001", "ISRC002"}, queried)
	assert.JSONEq(t, `{"data":[{"id":"ISRC001"},{"id":"ISRC002"}]}`, string(body))
}

func TestBuildValidationSummaryUsesWrittenOptionalArtifacts(t *testing.T) {
	t.Parallel()

	summary := buildValidationSummary(validationInputs{
		rawURL:     "https://music.apple.com/us/album/example/123",
		outputDir:  "/tmp/ariadne-apple-music-validation",
		parsed:     &model.ParsedAlbumURL{ID: "123", CanonicalURL: "https://music.apple.com/us/album/example/123"},
		storefront: "us",
	}, validationArtifacts{
		isrcBody: []byte(`{"data":[]}`),
	}, "Example Album", "Example Artist", "2024-01-02", "Example Label", "00602567713449", []string{"ISRC001"})

	artifactPaths, ok := summary["artifacts"].(map[string]string)
	require.True(t, ok)
	assert.NotContains(t, artifactPaths, "search_upc_official")
	assert.Contains(t, artifactPaths["search_isrc_official"], appleMusicSearchISRCFile)
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
		require.NotEmpty(t, content)
		assert.Equal(t, byte('\n'), content[len(content)-1])
	}
}
