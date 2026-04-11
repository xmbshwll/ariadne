package tidal

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
)

func assertSingleAlbum(t *testing.T, candidates []model.CandidateAlbum, wantID string) {
	t.Helper()
	require.Len(t, candidates, 1)
	assert.Equal(t, wantID, candidates[0].CandidateID)
	assert.Contains(t, candidates[0].MatchURL, wantID)
}

func assertSingleSong(t *testing.T, candidates []model.CandidateSong, wantID string) {
	t.Helper()
	require.Len(t, candidates, 1)
	assert.Equal(t, wantID, candidates[0].CandidateID)
	assert.Contains(t, candidates[0].MatchURL, wantID)
}

func newTIDALAPIAdapter(t *testing.T, registerHandlers func(*http.ServeMux)) *Adapter {
	t.Helper()
	mux := http.NewServeMux()
	registerTIDALTokenEndpoint(mux)
	registerHandlers(mux)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return New(server.Client(), WithCredentials("tidal-client", "tidal-secret"), WithAPIBaseURL(server.URL), WithAuthBaseURL(server.URL))
}

func registerTIDALTokenEndpoint(mux *http.ServeMux) {
	mux.HandleFunc("/oauth2/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, tokenResponse{AccessToken: "token-123", TokenType: "Bearer", ExpiresIn: 3600})
	})
}

func writeJSON(w http.ResponseWriter, payload any) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(payload); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(buf.Bytes())
}
