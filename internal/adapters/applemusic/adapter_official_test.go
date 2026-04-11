package applemusic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchByUPCWithOfficialAuth(t *testing.T) {
	fixture := newTestFixture(t, buildTestPayloads(t))

	results, err := fixture.authAdapter.SearchByUPC(context.Background(), "00602567713449")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "1441164426", results[0].CandidateID)
	assert.Equal(t, "https://music.apple.com/gb/album/abbey-road-remastered/1441164426", results[0].MatchURL)
	assert.Equal(t, "gb", results[0].RegionHint)
	assert.Equal(t, "00602567713449", results[0].UPC)
}

func TestSearchByISRCWithOfficialAuth(t *testing.T) {
	fixture := newTestFixture(t, buildTestPayloads(t))

	results, err := fixture.authAdapter.SearchByISRC(context.Background(), []string{comeTogetherISRC})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "1441164426", results[0].CandidateID)
	require.Len(t, results[0].Tracks, 2)
	assert.Equal(t, comeTogetherISRC, results[0].Tracks[0].ISRC)
}

func TestSearchSongByISRCWithOfficialAuth(t *testing.T) {
	fixture := newTestFixture(t, buildTestPayloads(t))

	results, err := fixture.authAdapter.SearchSongByISRC(context.Background(), comeTogetherISRC)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "1441164430", results[0].CandidateID)
	assert.Equal(t, comeTogetherTitle, results[0].Title)
}

func TestSearchByISRCWithOfficialAuthKeepsGoingAfterEarlierQueryFailure(t *testing.T) {
	payloads := buildTestPayloads(t)
	keyPath := writeTestPrivateKey(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/catalog/gb/songs", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			http.Error(w, "missing auth", http.StatusUnauthorized)
			return
		}
		switch r.URL.Query().Get("filter[isrc]") {
		case "BADISRC":
			http.Error(w, "temporary failure", http.StatusBadGateway)
		case comeTogetherISRC:
			_, _ = w.Write(payloads.officialISRC)
		default:
			http.NotFound(w, r)
		}
	})
	mux.HandleFunc("/catalog/gb/albums/1441164426", officialAlbumHandler(payloads))

	server := httptest.NewServer(mux)
	defer server.Close()

	adapter := New(
		server.Client(),
		WithAPIBaseURL(server.URL),
		WithDefaultStorefront("gb"),
		WithDeveloperTokenAuth("TEST12345", "TEAM123456", keyPath),
	)

	results, err := adapter.SearchByISRC(context.Background(), []string{"BADISRC", comeTogetherISRC})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "1441164426", results[0].CandidateID)
}
