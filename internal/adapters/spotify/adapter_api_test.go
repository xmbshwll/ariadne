package spotify_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	spotify "github.com/xmbshwll/ariadne/internal/adapters/spotify"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xmbshwll/ariadne/internal/model"
)

func TestAPIBackedAlbumAndSongOperations(t *testing.T) {
	adapter := newSpotifyAPIAdapter(t, func(mux *http.ServeMux) {
		mux.HandleFunc("/albums/album-good", func(w http.ResponseWriter, r *http.Request) {
			requireSpotifyBearerAuth(t, r)
			writeJSON(t, w, spotify.APIAlbumResponse{
				ID:          "album-good",
				Name:        "Abbey Road (Remastered)",
				ReleaseDate: "1969-09-26",
				Label:       "EMI Catalogue",
				TotalTracks: 17,
				Images:      []spotify.APIImage{{URL: "https://i.scdn.co/image/best", Width: 640}},
				Artists:     []spotify.APIArtist{{Name: "The Beatles"}},
				ExternalIDs: spotify.APIExternalIDs{UPC: "602547670342"},
				Tracks: spotify.APITrackPage{Items: []spotify.APITrack{
					{ID: "track-1", Name: "Come Together", TrackNumber: 1, DiscNumber: 1, DurationMS: 258947, Artists: []spotify.APIArtist{{Name: "The Beatles"}}},
					{ID: "track-2", Name: "Something", TrackNumber: 2, DiscNumber: 1, DurationMS: 182293, Artists: []spotify.APIArtist{{Name: "The Beatles"}}},
				}},
			})
		})
		mux.HandleFunc("/albums/album-weak", func(w http.ResponseWriter, r *http.Request) {
			requireSpotifyBearerAuth(t, r)
			writeJSON(t, w, spotify.APIAlbumResponse{
				ID:          "album-weak",
				Name:        "Abbey Road",
				ReleaseDate: "2020-01-01",
				Label:       "Other Label",
				TotalTracks: 17,
				Images:      []spotify.APIImage{{URL: "https://i.scdn.co/image/weak", Width: 640}},
				Artists:     []spotify.APIArtist{{Name: "The Beatles Complete On Ukulele"}},
				Tracks: spotify.APITrackPage{Items: []spotify.APITrack{
					{ID: "track-weak-1", Name: "Come Together", TrackNumber: 1, DiscNumber: 1, DurationMS: 200000, Artists: []spotify.APIArtist{{Name: "The Beatles Complete On Ukulele"}}},
				}},
			})
		})
		registerSpotifyTrackEndpoint(t, mux, map[string]spotify.APITrack{
			"track-1":      {ID: "track-1", Name: "Come Together", TrackNumber: 1, DiscNumber: 1, DurationMS: 258947, ExternalIDs: spotify.APIExternalIDs{ISRC: "GBAYE0601690"}, Artists: []spotify.APIArtist{{Name: "The Beatles"}}, Album: spotify.APITrackAlbum{ID: "album-good", Name: "Abbey Road (Remastered)", ReleaseDate: "1969-09-26", Images: []spotify.APIImage{{URL: "https://i.scdn.co/image/best", Width: 640}}, Artists: []spotify.APIArtist{{Name: "The Beatles"}}}},
			"track-2":      {ID: "track-2", Name: "Something", TrackNumber: 2, DiscNumber: 1, DurationMS: 182293, ExternalIDs: spotify.APIExternalIDs{ISRC: "GBAYE0601691"}, Artists: []spotify.APIArtist{{Name: "The Beatles"}}, Album: spotify.APITrackAlbum{ID: "album-good", Name: "Abbey Road (Remastered)", ReleaseDate: "1969-09-26", Images: []spotify.APIImage{{URL: "https://i.scdn.co/image/best", Width: 640}}, Artists: []spotify.APIArtist{{Name: "The Beatles"}}}},
			"track-weak-1": {ID: "track-weak-1", Name: "Come Together", TrackNumber: 1, DiscNumber: 1, DurationMS: 200000, ExternalIDs: spotify.APIExternalIDs{ISRC: "OTHER0001"}, Artists: []spotify.APIArtist{{Name: "The Beatles Complete On Ukulele"}}, Album: spotify.APITrackAlbum{ID: "album-weak", Name: "Abbey Road", ReleaseDate: "2020-01-01", Images: []spotify.APIImage{{URL: "https://i.scdn.co/image/weak", Width: 640}}, Artists: []spotify.APIArtist{{Name: "The Beatles Complete On Ukulele"}}}},
		})
		mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
			requireSpotifyBearerAuth(t, r)
			query := r.URL.Query().Get("q")
			switch {
			case strings.Contains(query, "upc:602547670342"):
				writeJSON(t, w, spotify.APIAlbumSearchResponse{Albums: spotify.APIAlbumSearchPage{Items: []spotify.APIAlbumSummary{{ID: "album-good"}}}})
			case strings.Contains(query, "isrc:GBAYE0601690"):
				writeJSON(t, w, spotify.APITrackSearchResponse{Tracks: spotify.APITrackSearchPage{Items: []spotify.APITrackSearchItem{{ID: "track-1", Name: "Come Together", DurationMS: 258947, Artists: []spotify.APIArtist{{Name: "The Beatles"}}, ExternalIDs: spotify.APIExternalIDs{ISRC: "GBAYE0601690"}, Album: spotify.APITrackAlbum{ID: "album-good", Name: "Abbey Road (Remastered)", ReleaseDate: "1969-09-26", Images: []spotify.APIImage{{URL: "https://i.scdn.co/image/best", Width: 640}}, Artists: []spotify.APIArtist{{Name: "The Beatles"}}}}}}})
			case strings.Contains(query, "isrc:GBAYE0601691"):
				writeJSON(t, w, spotify.APITrackSearchResponse{Tracks: spotify.APITrackSearchPage{Items: []spotify.APITrackSearchItem{{ID: "track-2", Name: "Something", DurationMS: 182293, Artists: []spotify.APIArtist{{Name: "The Beatles"}}, ExternalIDs: spotify.APIExternalIDs{ISRC: "GBAYE0601691"}, Album: spotify.APITrackAlbum{ID: "album-good", Name: "Abbey Road (Remastered)", ReleaseDate: "1969-09-26", Images: []spotify.APIImage{{URL: "https://i.scdn.co/image/best", Width: 640}}, Artists: []spotify.APIArtist{{Name: "The Beatles"}}}}}}})
			case strings.Contains(query, "album:Abbey Road (Remastered)"), strings.Contains(query, "album:Abbey Road artist:The Beatles"), strings.Contains(query, "album:Abbey Road"):
				writeJSON(t, w, spotify.APIAlbumSearchResponse{Albums: spotify.APIAlbumSearchPage{Items: []spotify.APIAlbumSummary{{ID: "album-good"}, {ID: "album-weak"}}}})
			case strings.Contains(query, "track:Come Together artist:The Beatles"), strings.Contains(query, "track:Come Together"):
				writeJSON(t, w, spotify.APITrackSearchResponse{Tracks: spotify.APITrackSearchPage{Items: []spotify.APITrackSearchItem{{ID: "track-1", Name: "Come Together", DurationMS: 258947, Artists: []spotify.APIArtist{{Name: "The Beatles"}}, ExternalIDs: spotify.APIExternalIDs{ISRC: "GBAYE0601690"}, Album: spotify.APITrackAlbum{ID: "album-good", Name: "Abbey Road (Remastered)", ReleaseDate: "1969-09-26", Images: []spotify.APIImage{{URL: "https://i.scdn.co/image/best", Width: 640}}, Artists: []spotify.APIArtist{{Name: "The Beatles"}}}}, {ID: "track-weak-1", Name: "Come Together", DurationMS: 200000, Artists: []spotify.APIArtist{{Name: "The Beatles Complete On Ukulele"}}, ExternalIDs: spotify.APIExternalIDs{ISRC: "OTHER0001"}, Album: spotify.APITrackAlbum{ID: "album-weak", Name: "Abbey Road", ReleaseDate: "2020-01-01", Images: []spotify.APIImage{{URL: "https://i.scdn.co/image/weak", Width: 640}}, Artists: []spotify.APIArtist{{Name: "The Beatles Complete On Ukulele"}}}}}}})
			default:
				http.NotFound(w, r)
			}
		})
	})

	parsed := model.ParsedAlbumURL{Service: model.ServiceSpotify, EntityType: model.EntityTypeAlbum, ID: "album-good", CanonicalURL: "https://open.spotify.com/album/album-good"}
	album, err := adapter.FetchAlbum(context.Background(), parsed)
	require.NoError(t, err)
	require.NotNil(t, album)
	require.NotEmpty(t, album.UPC)
	require.NotEmpty(t, album.Tracks)
	assert.Equal(t, "602547670342", album.UPC)
	assert.Equal(t, "GBAYE0601690", album.Tracks[0].ISRC)

	upcResults, err := adapter.SearchAlbumByUPC(context.Background(), "602547670342")
	require.NoError(t, err)
	assertSingleAlbum(t, upcResults, "album-good")

	isrcResults, err := adapter.SearchAlbumByISRC(context.Background(), []string{"GBAYE0601690", "GBAYE0601691"})
	require.NoError(t, err)
	assertSingleAlbum(t, isrcResults, "album-good")

	metadataResults, err := adapter.SearchAlbumByMetadata(context.Background(), model.CanonicalAlbum{Title: "Abbey Road (Remastered)", Artists: []string{"The Beatles"}})
	require.NoError(t, err)
	require.Len(t, metadataResults, 2)
	assert.Equal(t, "album-good", metadataResults[0].CandidateID)

	song, err := adapter.FetchSong(context.Background(), model.ParsedURL{Service: model.ServiceSpotify, EntityType: model.EntityTypeSong, ID: "track-1", CanonicalURL: "https://open.spotify.com/track/track-1"})
	require.NoError(t, err)
	require.NotNil(t, song)
	require.NotEmpty(t, song.ISRC)
	require.NotEmpty(t, song.AlbumTitle)
	assert.Equal(t, "GBAYE0601690", song.ISRC)
	assert.Equal(t, "Abbey Road (Remastered)", song.AlbumTitle)

	songISRCResults, err := adapter.SearchSongByISRC(context.Background(), "GBAYE0601690")
	require.NoError(t, err)
	assertSingleSong(t, songISRCResults, "track-1")

	songMetadataResults, err := adapter.SearchSongByMetadata(context.Background(), model.CanonicalSong{Title: "Come Together", Artists: []string{"The Beatles"}})
	require.NoError(t, err)
	require.Len(t, songMetadataResults, 2)
	assert.Equal(t, "track-1", songMetadataResults[0].CandidateID)
}
