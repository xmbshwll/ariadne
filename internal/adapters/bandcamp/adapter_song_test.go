package bandcamp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
)

func TestSearchSongByMetadataReturnsFirstHydrationErrorWhenNothingRecovers(t *testing.T) {
	server := newBandcampTestServer(func(baseURL string) map[string][]byte {
		return map[string][]byte{
			bandcampSearchPath: mustRenderBandcampFixture(t, "testdata/search-song-broken.html", baseURL),
			"/track/broken":    brokenBandcampSchemaBody(t),
		}
	})
	defer server.Close()

	adapter := newBandcampTestAdapter(server)
	_, err := adapter.SearchSongByMetadata(context.Background(), model.CanonicalSong{Title: "Come Together", Artists: []string{"COMRADIATION"}})
	require.Error(t, err)
	assert.ErrorIs(t, err, errMalformedBandcampJSONLD)
}

func TestSongAdapter(t *testing.T) {
	trackPage := mustBandcampTrackPage(t, "Come Together", "COMRADIATION", lonAbatyAbbeyRoad, "2021-12-02", 251000)
	liveTrackPage := mustBandcampTrackPage(t, "Come Together (Live)", "Tribute Band", "Abbey Road Live", "2020-01-01", 300000)

	server := newBandcampTestServer(func(baseURL string) map[string][]byte {
		return map[string][]byte{
			"/track/come-together":      trackPage,
			bandcampSearchPath:          mustRenderBandcampFixture(t, "testdata/search-song-basic.html", baseURL),
			"/track/come-together-live": liveTrackPage,
		}
	})
	defer server.Close()

	adapter := newBandcampTestAdapter(server)
	parsed := newBandcampSongSource(server.URL, "come-together")

	t.Run("fetch song", func(t *testing.T) {
		song, err := adapter.FetchSong(context.Background(), parsed)
		require.NoError(t, err)
		assert.Equal(t, "Come Together", song.Title)
		assert.Equal(t, lonAbatyAbbeyRoad, song.AlbumTitle)
		assert.Equal(t, 251000, song.DurationMS)
	})

	t.Run("search song by metadata", func(t *testing.T) {
		results, err := adapter.SearchSongByMetadata(context.Background(), model.CanonicalSong{
			Title:      "Come Together",
			Artists:    []string{"COMRADIATION"},
			DurationMS: 251000,
			AlbumTitle: lonAbatyAbbeyRoad,
		})
		require.NoError(t, err)
		require.Len(t, results, 2)
		assert.Equal(t, "come-together", results[0].CandidateID)
		assert.Equal(t, lonAbatyAbbeyRoad, results[0].AlbumTitle)
		assert.Equal(t, "come-together-live", results[1].CandidateID)
	})
}
