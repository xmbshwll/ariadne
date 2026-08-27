package tidal_test

import (
	"context"
	"net/http"
	"testing"

	tidal "github.com/xmbshwll/ariadne/internal/adapters/tidal"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xmbshwll/ariadne/internal/model"
)

func TestAdapterRuntimeOperations(t *testing.T) {
	const tidalTrackISRC = "QZMHK2043414"

	adapter := newTIDALAPIAdapter(t, func(mux *http.ServeMux) {
		mux.HandleFunc("/albums/156205493", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, tidal.APIDocument{
				Data: tidal.APIResource{
					ID:   "156205493",
					Type: "albums",
					Attributes: tidal.ResourceAttributes{
						Title:         "Shadows among trees",
						BarcodeID:     "053000502692",
						ReleaseDate:   "2020-10-02",
						Duration:      "PT35M",
						Explicit:      false,
						NumberOfItems: 5,
						Copyright:     tidal.ResourceCopyright{Text: "Posev"},
					},
					Relationships: tidal.ResourceRelationships{
						Artists:  tidal.Relationship{Data: []tidal.RelationshipData{{ID: "4152940", Type: "artists"}}},
						CoverArt: tidal.Relationship{Data: []tidal.RelationshipData{{ID: "art-1", Type: "artworks"}}},
						Items:    tidal.Relationship{Data: []tidal.RelationshipData{{ID: "156205494", Type: "tracks", Meta: tidal.RelationshipMeta{TrackNumber: 1, VolumeNumber: 1}}, {ID: "156205495", Type: "tracks", Meta: tidal.RelationshipMeta{TrackNumber: 2, VolumeNumber: 1}}}},
					},
				},
				Included: []tidal.APIResource{
					{ID: "4152940", Type: "artists", Attributes: tidal.ResourceAttributes{Name: "Fetch"}},
					{ID: "art-1", Type: "artworks", Attributes: tidal.ResourceAttributes{Files: []tidal.ResourceFile{{Href: "https://resources.tidal.test/1280.jpg", Meta: tidal.FileMeta{Width: 1280, Height: 1280}}}}},
					{ID: "156205494", Type: "tracks", Attributes: tidal.ResourceAttributes{Title: "Kings of mist", Duration: "PT6M30S", ISRC: tidalTrackISRC}, Relationships: tidal.ResourceRelationships{Artists: tidal.Relationship{Data: []tidal.RelationshipData{{ID: "4152940", Type: "artists"}}}}},
					{ID: "156205495", Type: "tracks", Attributes: tidal.ResourceAttributes{Title: "Something unspeakable", Duration: "PT7M00S", ISRC: "QZMHK2043415"}, Relationships: tidal.ResourceRelationships{Artists: tidal.Relationship{Data: []tidal.RelationshipData{{ID: "4152940", Type: "artists"}}}}},
				},
			})
		})
		mux.HandleFunc("/tracks/156205494", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, tidal.APIDocument{
				Data: tidal.APIResource{
					ID:   "156205494",
					Type: "tracks",
					Attributes: tidal.ResourceAttributes{
						Title:       "Kings of mist",
						Duration:    "PT6M30S",
						ISRC:        tidalTrackISRC,
						Explicit:    false,
						ReleaseDate: "2020-10-02",
					},
					Relationships: tidal.ResourceRelationships{
						Artists: tidal.Relationship{Data: []tidal.RelationshipData{{ID: "4152940", Type: "artists"}}},
						Albums:  tidal.Relationship{Data: []tidal.RelationshipData{{ID: "156205493", Type: "albums", Meta: tidal.RelationshipMeta{TrackNumber: 1, VolumeNumber: 1}}}},
					},
				},
				Included: []tidal.APIResource{
					{ID: "4152940", Type: "artists", Attributes: tidal.ResourceAttributes{Name: "Fetch"}},
					{ID: "156205493", Type: "albums", Attributes: tidal.ResourceAttributes{Title: "Shadows among trees", ReleaseDate: "2020-09-01"}, Relationships: tidal.ResourceRelationships{Artists: tidal.Relationship{Data: []tidal.RelationshipData{{ID: "4152940", Type: "artists"}}}, CoverArt: tidal.Relationship{Data: []tidal.RelationshipData{{ID: "art-1", Type: "artworks"}}}}},
					{ID: "art-1", Type: "artworks", Attributes: tidal.ResourceAttributes{Files: []tidal.ResourceFile{{Href: "https://resources.tidal.test/1280.jpg", Meta: tidal.FileMeta{Width: 1280, Height: 1280}}}}},
				},
			})
		})
		mux.HandleFunc("/tracks/156205495", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, tidal.APIDocument{
				Data: tidal.APIResource{
					ID:   "156205495",
					Type: "tracks",
					Attributes: tidal.ResourceAttributes{
						Title:       "Kings of mist (Live)",
						Duration:    "PT7M10S",
						ISRC:        "OTHER0001",
						Explicit:    false,
						ReleaseDate: "2021-01-01",
					},
					Relationships: tidal.ResourceRelationships{
						Artists: tidal.Relationship{Data: []tidal.RelationshipData{{ID: "999", Type: "artists"}}},
						Albums:  tidal.Relationship{Data: []tidal.RelationshipData{{ID: "9999", Type: "albums", Meta: tidal.RelationshipMeta{TrackNumber: 8, VolumeNumber: 1}}}},
					},
				},
				Included: []tidal.APIResource{
					{ID: "999", Type: "artists", Attributes: tidal.ResourceAttributes{Name: "Tribute Band"}},
					{ID: "9999", Type: "albums", Attributes: tidal.ResourceAttributes{Title: "Shadows among trees Live", ReleaseDate: "2021-01-01"}},
				},
			})
		})
		mux.HandleFunc("/tracks/missing", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, tidal.APIDocument{})
		})
		mux.HandleFunc("/albums", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("filter[barcodeId]") != "053000502692" {
				http.NotFound(w, r)
				return
			}
			writeJSON(w, tidal.APIDocument{Data: []tidal.APIResource{{ID: "156205493", Type: "albums"}}})
		})
		mux.HandleFunc("/tracks", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("filter[isrc]") != tidalTrackISRC {
				http.NotFound(w, r)
				return
			}
			writeJSON(w, tidal.APIDocument{Data: []tidal.APIResource{{ID: "156205494", Type: "tracks", Relationships: tidal.ResourceRelationships{Albums: tidal.Relationship{Data: []tidal.RelationshipData{{ID: "156205493", Type: "albums"}}}}}}})
		})
		mux.HandleFunc("/searchResults", func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query()
			switch {
			case query.Get("filter[query]") == "Shadows among trees Fetch" && query.Get("include") == "albums":
				writeJSON(w, tidal.APIDocument{Data: []tidal.APIResource{{ID: "sr-1", Type: "searchResults", Relationships: tidal.ResourceRelationships{Albums: tidal.Relationship{Data: []tidal.RelationshipData{{ID: "156205493", Type: "albums"}}}}}}})
			case query.Get("filter[query]") == "Kings of mist Fetch" && query.Get("include") == "tracks":
				writeJSON(w, tidal.APIDocument{Data: []tidal.APIResource{{ID: "sr-2", Type: "searchResults", Relationships: tidal.ResourceRelationships{Tracks: tidal.Relationship{Data: []tidal.RelationshipData{{ID: "156205494", Type: "tracks"}, {ID: "156205495", Type: "tracks"}}}}}}})
			default:
				http.NotFound(w, r)
			}
		})
	})

	parsed := model.ParsedAlbumURL{Service: model.ServiceTIDAL, EntityType: model.EntityTypeAlbum, ID: "156205493", CanonicalURL: "https://tidal.com/album/156205493"}
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

	song, err := adapter.FetchSong(context.Background(), model.ParsedURL{Service: model.ServiceTIDAL, EntityType: model.EntityTypeSong, ID: "156205494", CanonicalURL: "https://tidal.com/track/156205494"})
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

	_, err = adapter.FetchSong(context.Background(), model.ParsedURL{Service: model.ServiceTIDAL, EntityType: model.EntityTypeSong, ID: "missing", CanonicalURL: "https://tidal.com/track/missing"})
	require.Error(t, err)
	assert.ErrorIs(t, err, tidal.ErrTIDALTrackNotFound)
}
