package applemusic

import (
	"context"
	"net/http"
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
	found := false
	for _, track := range results[0].Tracks {
		if track.ISRC == comeTogetherISRC {
			found = true
			break
		}
	}
	assert.True(t, found)
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
	adapter := newOfficialTestAdapter(t, func(mux *http.ServeMux) {
		mux.HandleFunc("/catalog/gb/songs", func(w http.ResponseWriter, r *http.Request) {
			if !requireOfficialAuth(w, r) {
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
	})

	results, err := adapter.SearchByISRC(context.Background(), []string{"BADISRC", comeTogetherISRC})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "1441164426", results[0].CandidateID)
}

func TestSearchSongByISRCReturnsMalformedOfficialResponseError(t *testing.T) {
	adapter := newOfficialTestAdapter(t, func(mux *http.ServeMux) {
		mux.HandleFunc("/catalog/gb/songs", func(w http.ResponseWriter, r *http.Request) {
			if !requireOfficialAuth(w, r) {
				return
			}
			_, _ = w.Write([]byte("{"))
		})
	})

	_, err := adapter.SearchSongByISRC(context.Background(), comeTogetherISRC)
	require.Error(t, err)
	assert.ErrorIs(t, err, errMalformedAppleMusicOfficialResponse)
}
