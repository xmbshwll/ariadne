package tidal_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	tidal "github.com/xmbshwll/ariadne/internal/adapters/tidal"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xmbshwll/ariadne/internal/model"
)

func TestSearchByISRCKeepsEarlierResultsWhenLaterQueriesFail(t *testing.T) {
	adapter := newTIDALAPIAdapter(t, func(mux *http.ServeMux) {
		mux.HandleFunc("/tracks", func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Query().Get("filter[isrc]") {
			case "GOODISRC":
				writeJSON(w, tidal.APIDocument{Data: []tidal.APIResource{{ID: "track-good", Type: "tracks", Relationships: tidal.ResourceRelationships{Albums: tidal.Relationship{Data: []tidal.RelationshipData{{ID: "album-good", Type: "albums"}}}}}}})
			case "BADISRC":
				http.Error(w, "temporary tidal failure", http.StatusBadGateway)
			default:
				http.NotFound(w, r)
			}
		})
		mux.HandleFunc("/albums/album-good", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, tidal.APIDocument{Data: tidal.APIResource{ID: "album-good", Type: "albums", Attributes: tidal.ResourceAttributes{Title: "Album", ReleaseDate: "2024-01-01", NumberOfItems: 1}, Relationships: tidal.ResourceRelationships{Artists: tidal.Relationship{Data: []tidal.RelationshipData{{ID: "artist-1", Type: "artists"}}}, Items: tidal.Relationship{Data: []tidal.RelationshipData{{ID: "track-good", Type: "tracks", Meta: tidal.RelationshipMeta{TrackNumber: 1, VolumeNumber: 1}}}}}}, Included: []tidal.APIResource{{ID: "artist-1", Type: "artists", Attributes: tidal.ResourceAttributes{Name: "Artist"}}, {ID: "track-good", Type: "tracks", Attributes: tidal.ResourceAttributes{Title: "Song", ISRC: "GOODISRC", Duration: "PT3M"}, Relationships: tidal.ResourceRelationships{Artists: tidal.Relationship{Data: []tidal.RelationshipData{{ID: "artist-1", Type: "artists"}}}}}}})
		})
	})

	results, err := adapter.SearchByISRC(context.Background(), []string{"GOODISRC", "BADISRC"})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "album-good", results[0].CandidateID)
}

func TestSearchByISRCKeepsEarlierResultsWhenLaterHydrationFails(t *testing.T) {
	adapter := newTIDALAPIAdapter(t, func(mux *http.ServeMux) {
		mux.HandleFunc("/tracks", func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Query().Get("filter[isrc]") {
			case "GOODISRC":
				writeJSON(w, tidal.APIDocument{Data: []tidal.APIResource{{ID: "track-good", Type: "tracks", Relationships: tidal.ResourceRelationships{Albums: tidal.Relationship{Data: []tidal.RelationshipData{{ID: "album-good", Type: "albums"}}}}}}})
			case "MISSINGALBUM":
				writeJSON(w, tidal.APIDocument{Data: []tidal.APIResource{{ID: "track-missing", Type: "tracks", Relationships: tidal.ResourceRelationships{Albums: tidal.Relationship{Data: []tidal.RelationshipData{{ID: "album-missing", Type: "albums"}}}}}}})
			default:
				http.NotFound(w, r)
			}
		})
		mux.HandleFunc("/albums/album-good", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, tidal.APIDocument{Data: tidal.APIResource{ID: "album-good", Type: "albums", Attributes: tidal.ResourceAttributes{Title: "Album", ReleaseDate: "2024-01-01", NumberOfItems: 1}, Relationships: tidal.ResourceRelationships{Artists: tidal.Relationship{Data: []tidal.RelationshipData{{ID: "artist-1", Type: "artists"}}}, Items: tidal.Relationship{Data: []tidal.RelationshipData{{ID: "track-good", Type: "tracks", Meta: tidal.RelationshipMeta{TrackNumber: 1, VolumeNumber: 1}}}}}}, Included: []tidal.APIResource{{ID: "artist-1", Type: "artists", Attributes: tidal.ResourceAttributes{Name: "Artist"}}, {ID: "track-good", Type: "tracks", Attributes: tidal.ResourceAttributes{Title: "Song", ISRC: "GOODISRC", Duration: "PT3M"}, Relationships: tidal.ResourceRelationships{Artists: tidal.Relationship{Data: []tidal.RelationshipData{{ID: "artist-1", Type: "artists"}}}}}}})
		})
		mux.HandleFunc("/albums/album-missing", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, tidal.APIDocument{})
		})
	})

	results, err := adapter.SearchByISRC(context.Background(), []string{"GOODISRC", "MISSINGALBUM"})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "album-good", results[0].CandidateID)
}

func TestSearchByMetadataReturnsMalformedResponseError(t *testing.T) {
	adapter := newTIDALAPIAdapter(t, func(mux *http.ServeMux) {
		mux.HandleFunc("/searchResults", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("filter[query]") != "Album Artist" {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write([]byte("{"))
		})
	})

	_, err := adapter.SearchByMetadata(context.Background(), model.CanonicalAlbum{Title: "Album", Artists: []string{"Artist"}})
	require.Error(t, err)
	assert.ErrorIs(t, err, tidal.ErrMalformedTIDALAPIResponse)
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
		_ = json.NewEncoder(w).Encode(tidal.TokenResponse{AccessToken: "token-123", TokenType: "Bearer", ExpiresIn: 3600})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	adapter := tidal.New(server.Client(), tidal.WithCredentials("tidal-client", "tidal-secret"), tidal.WithAuthBaseURL(server.URL))
	errCh := make(chan error, 8)
	for range 8 {
		go func() {
			_, err := adapter.AccessToken(context.Background())
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
