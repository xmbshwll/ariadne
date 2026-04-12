package bandcamp

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
)

func TestAlbumAdapter(t *testing.T) {
	sourcePage := mustReadBandcampSourcePage(t)

	server := newBandcampTestServer(func(baseURL string) map[string][]byte {
		return map[string][]byte{
			"/album/l-n-abaty-abbey-road": sourcePage,
			bandcampSearchPath:            mustRenderBandcampFixture(t, "testdata/search-album-basic.html", baseURL),
		}
	})
	defer server.Close()

	adapter := newBandcampTestAdapter(server)
	parsed := newBandcampAlbumSource(server.URL, "l-n-abaty-abbey-road")

	t.Run("fetch album", func(t *testing.T) {
		album, err := adapter.FetchAlbum(context.Background(), parsed)
		require.NoError(t, err)
		assert.Equal(t, lonAbatyAbbeyRoad, album.Title)
		assert.Equal(t, "l-n-abaty-abbey-road", album.SourceID)
		assert.Equal(t, parsed.CanonicalURL, album.SourceURL)
		assert.Equal(t, 14, album.TrackCount)
		require.Len(t, album.Tracks, 14)
		assert.NotEmpty(t, album.Tracks[0].Title)
		assert.Positive(t, album.TotalDurationMS)
		assert.NotEmpty(t, album.ArtworkURL)
		assert.Equal(t, "2021-12-02", album.ReleaseDate)
	})

	t.Run("search by metadata", func(t *testing.T) {
		results, err := adapter.SearchByMetadata(context.Background(), model.CanonicalAlbum{
			Title:   "Abbey Road",
			Artists: []string{"COMRADIATION"},
		})
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, "l-n-abaty-abbey-road", results[0].CandidateID)
		assert.Contains(t, results[0].MatchURL, "/album/l-n-abaty-abbey-road")
	})

	t.Run("search by upc unsupported", func(t *testing.T) {
		results, err := adapter.SearchByUPC(context.Background(), "123")
		require.NoError(t, err)
		assert.Empty(t, results)
	})
}

func TestSearchByMetadataReranksHydratedCandidates(t *testing.T) {
	source := model.CanonicalAlbum{
		Title:      "Live at KEXP",
		Artists:    []string{"Sea Lemon"},
		TrackCount: 4,
		Tracks: []model.CanonicalTrack{
			{Title: "Stay", NormalizedTitle: "stay"},
			{Title: "Cellar", NormalizedTitle: "cellar"},
			{Title: "Vaporized", NormalizedTitle: "vaporized"},
			{Title: "Give In", NormalizedTitle: "give in"},
		},
	}

	lowOverlapPage := mustBandcampAlbumPage(t, "Live at KEXP", "Sea Lemon", "2024-01-10", []string{"Stay", "Blue Moon", "Drive", "Night Swim"})
	highOverlapPage := mustBandcampAlbumPage(t, "Live at KEXP", "Sea Lemon", "2024-01-10", []string{"Stay", "Cellar", "Vaporized", "Give In"})

	server := newBandcampTestServer(func(baseURL string) map[string][]byte {
		return map[string][]byte{
			bandcampSearchPath:         mustRenderBandcampFixture(t, "testdata/search-album-rerank.html", baseURL),
			"/album/live-at-kexp-low":  lowOverlapPage,
			"/album/live-at-kexp-high": highOverlapPage,
		}
	})
	defer server.Close()

	adapter := newBandcampTestAdapter(server)
	results, err := adapter.SearchByMetadata(context.Background(), source)
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, "live-at-kexp-high", results[0].CandidateID)
	assert.Equal(t, "live-at-kexp-low", results[1].CandidateID)
}

func TestSearchByMetadataReturnsFirstHydrationErrorWhenNothingRecovers(t *testing.T) {
	server := newBandcampTestServer(func(baseURL string) map[string][]byte {
		return map[string][]byte{
			bandcampSearchPath: mustRenderBandcampFixture(t, "testdata/search-album-broken.html", baseURL),
			"/album/broken":    brokenBandcampSchemaBody(t),
		}
	})
	defer server.Close()

	adapter := newBandcampTestAdapter(server)
	_, err := adapter.SearchByMetadata(context.Background(), model.CanonicalAlbum{Title: "Abbey Road", Artists: []string{"COMRADIATION"}})
	require.Error(t, err)
	assert.ErrorIs(t, err, errMalformedBandcampJSONLD)
}

func TestRealSavedPages(t *testing.T) {
	tests := []struct {
		name        string
		fixture     string
		path        string
		wantTitle   string
		wantArtist  string
		wantTracks  int
		wantDate    string
		wantArtwork bool
	}{
		{
			name:        "after abbey road",
			fixture:     "testdata/real-after-abbey-road.html",
			path:        "/album/after-abbey-road",
			wantTitle:   "After Abbey Road",
			wantArtist:  "Mike Westbrook",
			wantTracks:  17,
			wantDate:    "2019-09-27",
			wantArtwork: true,
		},
		{
			name:        "morningrise abbey road remaster",
			fixture:     "testdata/real-morningrise-abbey-road-remaster.html",
			path:        "/album/morningrise-abbey-road-remaster",
			wantTitle:   "Morningrise (Abbey Road Remaster)",
			wantArtist:  "Opeth",
			wantTracks:  5,
			wantDate:    "2023-06-02",
			wantArtwork: true,
		},
		{
			name:        "for those that wish to exist at abbey road",
			fixture:     "testdata/real-for-those-that-wish-to-exist-at-abbey-road.html",
			path:        "/album/for-those-that-wish-to-exist-at-abbey-road",
			wantTitle:   "For Those That Wish To Exist At Abbey Road",
			wantArtist:  "Architects",
			wantTracks:  15,
			wantDate:    "2022-03-25",
			wantArtwork: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := mustReadTestFile(t, tt.fixture)
			server := newBandcampTestServer(func(string) map[string][]byte {
				return map[string][]byte{tt.path: page}
			})
			defer server.Close()

			adapter := New(server.Client())
			parsed := newBandcampAlbumSource(server.URL, strings.TrimPrefix(tt.path, "/album/"))

			album, err := adapter.FetchAlbum(context.Background(), parsed)
			require.NoError(t, err)
			assert.Equal(t, tt.wantTitle, album.Title)
			require.NotEmpty(t, album.Artists)
			assert.Equal(t, tt.wantArtist, album.Artists[0])
			assert.Equal(t, tt.wantTracks, album.TrackCount)
			assert.Equal(t, tt.wantDate, album.ReleaseDate)
			if tt.wantArtwork {
				assert.NotEmpty(t, album.ArtworkURL)
			}
		})
	}
}
