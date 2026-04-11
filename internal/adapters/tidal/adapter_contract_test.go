package tidal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchByISRCKeepsEarlierResultsWhenLaterQueriesFail(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth2/token", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(tokenResponse{AccessToken: "token-123", TokenType: "Bearer", ExpiresIn: 3600})
	})
	mux.HandleFunc("/tracks", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("filter[isrc]") {
		case "GOODISRC":
			writeJSON(w, apiDocument{Data: []apiResource{{ID: "track-good", Type: "tracks", Relationships: resourceRelationships{Albums: relationship{Data: []relationshipData{{ID: "album-good", Type: "albums"}}}}}}})
		case "BADISRC":
			http.Error(w, "temporary tidal failure", http.StatusBadGateway)
		default:
			http.NotFound(w, r)
		}
	})
	mux.HandleFunc("/albums/album-good", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, apiDocument{Data: apiResource{ID: "album-good", Type: "albums", Attributes: resourceAttributes{Title: "Album", ReleaseDate: "2024-01-01", NumberOfItems: 1}, Relationships: resourceRelationships{Artists: relationship{Data: []relationshipData{{ID: "artist-1", Type: "artists"}}}, Items: relationship{Data: []relationshipData{{ID: "track-good", Type: "tracks", Meta: relationshipMeta{TrackNumber: 1, VolumeNumber: 1}}}}}}, Included: []apiResource{{ID: "artist-1", Type: "artists", Attributes: resourceAttributes{Name: "Artist"}}, {ID: "track-good", Type: "tracks", Attributes: resourceAttributes{Title: "Song", ISRC: "GOODISRC", Duration: "PT3M"}, Relationships: resourceRelationships{Artists: relationship{Data: []relationshipData{{ID: "artist-1", Type: "artists"}}}}}}})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	adapter := New(server.Client(), WithCredentials("tidal-client", "tidal-secret"), WithAPIBaseURL(server.URL), WithAuthBaseURL(server.URL))
	results, err := adapter.SearchByISRC(context.Background(), []string{"GOODISRC", "BADISRC"})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "album-good", results[0].CandidateID)
}
