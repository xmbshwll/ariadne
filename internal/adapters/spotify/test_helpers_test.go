package spotify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func newSpotifyAPIAdapter(t *testing.T, registerHandlers func(*http.ServeMux)) *Adapter {
	t.Helper()
	mux := http.NewServeMux()
	registerSpotifyTokenEndpoint(t, mux)
	registerHandlers(mux)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return New(server.Client(), WithCredentials("client-id", "client-secret"), WithAPIBaseURL(server.URL), WithAuthBaseURL(server.URL))
}

func registerSpotifyTokenEndpoint(t *testing.T, mux *http.ServeMux) {
	t.Helper()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		requireSpotifyTokenRequest(t, r)
		_ = json.NewEncoder(w).Encode(tokenResponse{AccessToken: "token-123", TokenType: "Bearer", ExpiresIn: 3600})
	})
}

func requireSpotifyTokenRequest(t *testing.T, r *http.Request) {
	t.Helper()
	assert.Equal(t, http.MethodPost, r.Method)
	assert.NoError(t, r.ParseForm())
	assert.Equal(t, "client_credentials", r.Form.Get("grant_type"))
	assert.True(t, strings.HasPrefix(r.Header.Get("Authorization"), "Basic "))
}

func requireSpotifyBearerAuth(t *testing.T, r *http.Request) {
	t.Helper()
	assert.Equal(t, "Bearer token-123", r.Header.Get("Authorization"))
}

func writeSpotifyTrackBatchJSON(t *testing.T, w http.ResponseWriter, r *http.Request, tracksByID map[string]apiTrack) {
	t.Helper()
	ids := strings.Split(r.URL.Query().Get("ids"), ",")
	tracks := make([]*apiTrack, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		track, ok := tracksByID[id]
		if !ok {
			tracks = append(tracks, nil)
			continue
		}
		trackCopy := track
		tracks = append(tracks, &trackCopy)
	}
	writeJSON(t, w, apiTrackBatchResponse{Tracks: tracks})
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
