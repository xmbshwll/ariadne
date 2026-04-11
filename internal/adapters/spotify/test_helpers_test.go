package spotify

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
)

func mustReadTestFile(t *testing.T, relativePath string) []byte {
	t.Helper()
	path := filepath.Clean(relativePath)
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return content
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	require.NoError(t, json.NewEncoder(w).Encode(payload))
}

func assertSingleAlbum(t *testing.T, candidates []model.CandidateAlbum, wantID string) {
	t.Helper()
	require.Len(t, candidates, 1)
	assert.Equal(t, wantID, candidates[0].CandidateID)
}

func assertSingleSong(t *testing.T, candidates []model.CandidateSong, wantID string) {
	t.Helper()
	require.Len(t, candidates, 1)
	assert.Equal(t, wantID, candidates[0].CandidateID)
}

func TestSearchAlbumByMetadataEmptyAlbumWithoutCredentialsReturnsEmptyResults(t *testing.T) {
	adapter := New(http.DefaultClient)

	results, err := adapter.SearchByMetadata(context.Background(), model.CanonicalAlbum{})
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestSearchSongByMetadataEmptySongWithoutCredentialsReturnsEmptyResults(t *testing.T) {
	adapter := New(http.DefaultClient)

	results, err := adapter.SearchSongByMetadata(context.Background(), model.CanonicalSong{})
	require.NoError(t, err)
	assert.Empty(t, results)
}
