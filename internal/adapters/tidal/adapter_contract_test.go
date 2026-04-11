package tidal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

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

func TestSearchByISRCKeepsEarlierResultsWhenLaterHydrationFails(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth2/token", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(tokenResponse{AccessToken: "token-123", TokenType: "Bearer", ExpiresIn: 3600})
	})
	mux.HandleFunc("/tracks", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("filter[isrc]") {
		case "GOODISRC":
			writeJSON(w, apiDocument{Data: []apiResource{{ID: "track-good", Type: "tracks", Relationships: resourceRelationships{Albums: relationship{Data: []relationshipData{{ID: "album-good", Type: "albums"}}}}}}})
		case "MISSINGALBUM":
			writeJSON(w, apiDocument{Data: []apiResource{{ID: "track-missing", Type: "tracks", Relationships: resourceRelationships{Albums: relationship{Data: []relationshipData{{ID: "album-missing", Type: "albums"}}}}}}})
		default:
			http.NotFound(w, r)
		}
	})
	mux.HandleFunc("/albums/album-good", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, apiDocument{Data: apiResource{ID: "album-good", Type: "albums", Attributes: resourceAttributes{Title: "Album", ReleaseDate: "2024-01-01", NumberOfItems: 1}, Relationships: resourceRelationships{Artists: relationship{Data: []relationshipData{{ID: "artist-1", Type: "artists"}}}, Items: relationship{Data: []relationshipData{{ID: "track-good", Type: "tracks", Meta: relationshipMeta{TrackNumber: 1, VolumeNumber: 1}}}}}}, Included: []apiResource{{ID: "artist-1", Type: "artists", Attributes: resourceAttributes{Name: "Artist"}}, {ID: "track-good", Type: "tracks", Attributes: resourceAttributes{Title: "Song", ISRC: "GOODISRC", Duration: "PT3M"}, Relationships: resourceRelationships{Artists: relationship{Data: []relationshipData{{ID: "artist-1", Type: "artists"}}}}}}})
	})
	mux.HandleFunc("/albums/album-missing", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, apiDocument{})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	adapter := New(server.Client(), WithCredentials("tidal-client", "tidal-secret"), WithAPIBaseURL(server.URL), WithAuthBaseURL(server.URL))
	results, err := adapter.SearchByISRC(context.Background(), []string{"GOODISRC", "MISSINGALBUM"})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "album-good", results[0].CandidateID)
}

func TestAccessTokenSerializesConcurrentRefresh(t *testing.T) {
	var tokenRequests atomic.Int32
	started := make(chan struct{}, 8)
	allowResponse := make(chan struct{})

	mux := http.NewServeMux()
	mux.HandleFunc("/oauth2/token", func(w http.ResponseWriter, r *http.Request) {
		tokenRequests.Add(1)
		started <- struct{}{}
		<-allowResponse
		_ = json.NewEncoder(w).Encode(tokenResponse{AccessToken: "token-123", TokenType: "Bearer", ExpiresIn: 3600})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	adapter := New(server.Client(), WithCredentials("tidal-client", "tidal-secret"), WithAuthBaseURL(server.URL))
	errCh := make(chan error, 8)
	for range 8 {
		go func() {
			_, err := adapter.accessToken(context.Background())
			errCh <- err
		}()
	}

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		require.FailNow(t, "timed out waiting for token refresh")
	}

	select {
	case <-started:
		require.FailNow(t, "saw concurrent token refresh")
	case <-time.After(100 * time.Millisecond):
	}
	close(allowResponse)

	for range 8 {
		require.NoError(t, <-errCh)
	}
	assert.EqualValues(t, 1, tokenRequests.Load())
}
