package tidal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
)

func TestAdapterRuntimeOperations(t *testing.T) {
	const tidalTrackISRC = "QZMHK2043414"

	mux := http.NewServeMux()
	mux.HandleFunc("/oauth2/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		_ = json.NewEncoder(w).Encode(tokenResponse{AccessToken: "token-123", TokenType: "Bearer", ExpiresIn: 3600})
	})
	mux.HandleFunc("/albums/156205493", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, apiDocument{
			Data: apiResource{
				ID:   "156205493",
				Type: "albums",
				Attributes: resourceAttributes{
					Title:         "Shadows among trees",
					BarcodeID:     "053000502692",
					ReleaseDate:   "2020-10-02",
					Duration:      "PT35M",
					Explicit:      false,
					NumberOfItems: 5,
					Copyright:     resourceCopyright{Text: "Posev"},
				},
				Relationships: resourceRelationships{
					Artists:  relationship{Data: []relationshipData{{ID: "4152940", Type: "artists"}}},
					CoverArt: relationship{Data: []relationshipData{{ID: "art-1", Type: "artworks"}}},
					Items:    relationship{Data: []relationshipData{{ID: "156205494", Type: "tracks", Meta: relationshipMeta{TrackNumber: 1, VolumeNumber: 1}}, {ID: "156205495", Type: "tracks", Meta: relationshipMeta{TrackNumber: 2, VolumeNumber: 1}}}},
				},
			},
			Included: []apiResource{
				{ID: "4152940", Type: "artists", Attributes: resourceAttributes{Name: "Fetch"}},
				{ID: "art-1", Type: "artworks", Attributes: resourceAttributes{Files: []resourceFile{{Href: "https://resources.tidal.test/1280.jpg", Meta: fileMeta{Width: 1280, Height: 1280}}}}},
				{ID: "156205494", Type: "tracks", Attributes: resourceAttributes{Title: "Kings of mist", Duration: "PT6M30S", ISRC: tidalTrackISRC}, Relationships: resourceRelationships{Artists: relationship{Data: []relationshipData{{ID: "4152940", Type: "artists"}}}}},
				{ID: "156205495", Type: "tracks", Attributes: resourceAttributes{Title: "Something unspeakable", Duration: "PT7M00S", ISRC: "QZMHK2043415"}, Relationships: resourceRelationships{Artists: relationship{Data: []relationshipData{{ID: "4152940", Type: "artists"}}}}},
			},
		})
	})
	mux.HandleFunc("/tracks/156205494", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, apiDocument{
			Data: apiResource{
				ID:   "156205494",
				Type: "tracks",
				Attributes: resourceAttributes{
					Title:       "Kings of mist",
					Duration:    "PT6M30S",
					ISRC:        tidalTrackISRC,
					Explicit:    false,
					ReleaseDate: "2020-10-02",
				},
				Relationships: resourceRelationships{
					Artists: relationship{Data: []relationshipData{{ID: "4152940", Type: "artists"}}},
					Albums:  relationship{Data: []relationshipData{{ID: "156205493", Type: "albums", Meta: relationshipMeta{TrackNumber: 1, VolumeNumber: 1}}}},
				},
			},
			Included: []apiResource{
				{ID: "4152940", Type: "artists", Attributes: resourceAttributes{Name: "Fetch"}},
				{ID: "156205493", Type: "albums", Attributes: resourceAttributes{Title: "Shadows among trees", ReleaseDate: "2020-09-01"}, Relationships: resourceRelationships{Artists: relationship{Data: []relationshipData{{ID: "4152940", Type: "artists"}}}, CoverArt: relationship{Data: []relationshipData{{ID: "art-1", Type: "artworks"}}}}},
				{ID: "art-1", Type: "artworks", Attributes: resourceAttributes{Files: []resourceFile{{Href: "https://resources.tidal.test/1280.jpg", Meta: fileMeta{Width: 1280, Height: 1280}}}}},
			},
		})
	})
	mux.HandleFunc("/tracks/156205495", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, apiDocument{
			Data: apiResource{
				ID:   "156205495",
				Type: "tracks",
				Attributes: resourceAttributes{
					Title:       "Kings of mist (Live)",
					Duration:    "PT7M10S",
					ISRC:        "OTHER0001",
					Explicit:    false,
					ReleaseDate: "2021-01-01",
				},
				Relationships: resourceRelationships{
					Artists: relationship{Data: []relationshipData{{ID: "999", Type: "artists"}}},
					Albums:  relationship{Data: []relationshipData{{ID: "9999", Type: "albums", Meta: relationshipMeta{TrackNumber: 8, VolumeNumber: 1}}}},
				},
			},
			Included: []apiResource{
				{ID: "999", Type: "artists", Attributes: resourceAttributes{Name: "Tribute Band"}},
				{ID: "9999", Type: "albums", Attributes: resourceAttributes{Title: "Shadows among trees Live", ReleaseDate: "2021-01-01"}},
			},
		})
	})
	mux.HandleFunc("/tracks/missing", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, apiDocument{})
	})
	mux.HandleFunc("/albums", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("filter[barcodeId]") != "053000502692" {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, apiDocument{Data: []apiResource{{ID: "156205493", Type: "albums"}}})
	})
	mux.HandleFunc("/tracks", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("filter[isrc]") != tidalTrackISRC {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, apiDocument{Data: []apiResource{{ID: "156205494", Type: "tracks", Relationships: resourceRelationships{Albums: relationship{Data: []relationshipData{{ID: "156205493", Type: "albums"}}}}}}})
	})
	mux.HandleFunc("/searchResults/", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/searchResults/Shadows among trees Fetch/relationships/albums", "/searchResults/Shadows%20among%20trees%20Fetch/relationships/albums":
			writeJSON(w, apiDocument{Data: []apiResource{{ID: "156205493", Type: "albums"}}})
		case "/searchResults/Kings of mist Fetch/relationships/tracks", "/searchResults/Kings%20of%20mist%20Fetch/relationships/tracks":
			writeJSON(w, apiDocument{Data: []apiResource{{ID: "156205494", Type: "tracks"}, {ID: "156205495", Type: "tracks"}}})
		default:
			http.NotFound(w, r)
		}
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	adapter := New(
		server.Client(),
		WithCredentials("tidal-client", "tidal-secret"),
		WithAPIBaseURL(server.URL),
		WithAuthBaseURL(server.URL),
	)

	parsed := model.ParsedAlbumURL{Service: model.ServiceTIDAL, EntityType: "album", ID: "156205493", CanonicalURL: "https://tidal.com/album/156205493"}
	album, err := adapter.FetchAlbum(context.Background(), parsed)
	require.NoError(t, err)
	assert.Equal(t, "Shadows among trees", album.Title)
	assert.Equal(t, "053000502692", album.UPC)
	require.Len(t, album.Tracks, 2)
	assert.Equal(t, tidalTrackISRC, album.Tracks[0].ISRC)
	assert.NotEmpty(t, album.ArtworkURL)

	upcResults, err := adapter.SearchByUPC(context.Background(), "053000502692")
	require.NoError(t, err)
	assertSingleAlbum(t, upcResults, "156205493")

	isrcResults, err := adapter.SearchByISRC(context.Background(), []string{tidalTrackISRC})
	require.NoError(t, err)
	assertSingleAlbum(t, isrcResults, "156205493")

	metadataResults, err := adapter.SearchByMetadata(context.Background(), model.CanonicalAlbum{Title: "Shadows among trees", Artists: []string{"Fetch"}})
	require.NoError(t, err)
	assertSingleAlbum(t, metadataResults, "156205493")

	song, err := adapter.FetchSong(context.Background(), model.ParsedURL{Service: model.ServiceTIDAL, EntityType: "song", ID: "156205494", CanonicalURL: "https://tidal.com/track/156205494"})
	require.NoError(t, err)
	assert.Equal(t, tidalTrackISRC, song.ISRC)
	assert.Equal(t, "Shadows among trees", song.AlbumTitle)
	assert.Equal(t, "2020-10-02", song.ReleaseDate)

	songISRCResults, err := adapter.SearchSongByISRC(context.Background(), tidalTrackISRC)
	require.NoError(t, err)
	assertSingleSong(t, songISRCResults, "156205494")

	songMetadataResults, err := adapter.SearchSongByMetadata(context.Background(), model.CanonicalSong{Title: "Kings of mist", Artists: []string{"Fetch"}})
	require.NoError(t, err)
	require.Len(t, songMetadataResults, 2)
	assert.Equal(t, "156205494", songMetadataResults[0].CandidateID)

	_, err = adapter.FetchSong(context.Background(), model.ParsedURL{Service: model.ServiceTIDAL, EntityType: "song", ID: "missing", CanonicalURL: "https://tidal.com/track/missing"})
	require.Error(t, err)
	assert.ErrorIs(t, err, errTIDALTrackNotFound)
}
