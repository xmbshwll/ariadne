package tidal

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
)

func TestAdapter(t *testing.T) {
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
		writeJSON(t, w, apiDocument{
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
				{ID: "156205494", Type: "tracks", Attributes: resourceAttributes{Title: "Kings of mist", Duration: "PT6M30S", ISRC: "QZMHK2043414"}, Relationships: resourceRelationships{Artists: relationship{Data: []relationshipData{{ID: "4152940", Type: "artists"}}}}},
				{ID: "156205495", Type: "tracks", Attributes: resourceAttributes{Title: "Something unspeakable", Duration: "PT7M00S", ISRC: "QZMHK2043415"}, Relationships: resourceRelationships{Artists: relationship{Data: []relationshipData{{ID: "4152940", Type: "artists"}}}}},
			},
		})
	})
	mux.HandleFunc("/tracks/156205494", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, apiDocument{
			Data: apiResource{
				ID:   "156205494",
				Type: "tracks",
				Attributes: resourceAttributes{
					Title:       "Kings of mist",
					Duration:    "PT6M30S",
					ISRC:        "QZMHK2043414",
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
		writeJSON(t, w, apiDocument{
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
		writeJSON(t, w, apiDocument{})
	})
	mux.HandleFunc("/albums", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("filter[barcodeId]") != "053000502692" {
			http.NotFound(w, r)
			return
		}
		writeJSON(t, w, apiDocument{Data: []apiResource{{ID: "156205493", Type: "albums"}}})
	})
	mux.HandleFunc("/tracks", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("filter[isrc]") != tidalTrackISRC {
			http.NotFound(w, r)
			return
		}
		writeJSON(t, w, apiDocument{Data: []apiResource{{ID: "156205494", Type: "tracks", Relationships: resourceRelationships{Albums: relationship{Data: []relationshipData{{ID: "156205493", Type: "albums"}}}}}}})
	})
	mux.HandleFunc("/searchResults/", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/searchResults/Shadows among trees Fetch/relationships/albums", "/searchResults/Shadows%20among%20trees%20Fetch/relationships/albums":
			writeJSON(t, w, apiDocument{Data: []apiResource{{ID: "156205493", Type: "albums"}}})
		case "/searchResults/Kings of mist Fetch/relationships/tracks", "/searchResults/Kings%20of%20mist%20Fetch/relationships/tracks":
			writeJSON(t, w, apiDocument{Data: []apiResource{{ID: "156205494", Type: "tracks"}, {ID: "156205495", Type: "tracks"}}})
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

	song, err := adapter.FetchSong(context.Background(), model.ParsedAlbumURL{Service: model.ServiceTIDAL, EntityType: "song", ID: "156205494", CanonicalURL: "https://tidal.com/track/156205494"})
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

	_, err = adapter.FetchSong(context.Background(), model.ParsedAlbumURL{Service: model.ServiceTIDAL, EntityType: "song", ID: "missing", CanonicalURL: "https://tidal.com/track/missing"})
	require.Error(t, err)
	assert.ErrorIs(t, err, errTIDALTrackNotFound)
}

func TestIncludedResourceLookupsUseTypeAndID(t *testing.T) {
	included := []apiResource{
		{ID: "shared", Type: "albums", Attributes: resourceAttributes{Title: "Album Resource"}},
		{ID: "shared", Type: "artists", Attributes: resourceAttributes{Name: "Artist Resource"}},
		{ID: "shared", Type: "artworks", Attributes: resourceAttributes{Files: []resourceFile{{Href: "https://resources.tidal.test/shared.jpg", Meta: fileMeta{Width: 1280, Height: 1280}}}}},
	}

	artistNames := includedArtistNames(included, []relationshipData{{ID: "shared", Type: "artists"}})
	assert.Equal(t, []string{"Artist Resource"}, artistNames)

	album := firstRelatedResource(included, []relationshipData{{ID: "shared", Type: "albums"}}, "albums")
	require.NotNil(t, album)
	assert.Equal(t, "Album Resource", album.Attributes.Title)

	artworkURL := artworkURLFromIncluded(included, []relationshipData{{ID: "shared", Type: "artworks"}})
	assert.Equal(t, "https://resources.tidal.test/shared.jpg", artworkURL)
}

func TestAdapterRequiresCredentialsForSourceAndSearch(t *testing.T) {
	adapter := New(nil)

	_, err := adapter.FetchAlbum(context.Background(), model.ParsedAlbumURL{Service: model.ServiceTIDAL, ID: "156205493", CanonicalURL: "https://tidal.com/album/156205493"})
	require.Error(t, err)
	_, err = adapter.FetchSong(context.Background(), model.ParsedAlbumURL{Service: model.ServiceTIDAL, ID: "156205494", CanonicalURL: "https://tidal.com/track/156205494"})
	require.Error(t, err)
	_, err = adapter.SearchByMetadata(context.Background(), model.CanonicalAlbum{Title: "Album"})
	require.Error(t, err)
	_, err = adapter.SearchSongByMetadata(context.Background(), model.CanonicalSong{Title: "Song"})
	require.Error(t, err)
}

func TestAdapterSkipsCredentialChecksForEmptySearches(t *testing.T) {
	adapter := New(nil)

	tests := []struct {
		name string
		fn   func() (any, error)
	}{
		{
			name: "album isrc search",
			fn: func() (any, error) {
				return adapter.SearchByISRC(context.Background(), []string{"", " "})
			},
		},
		{
			name: "album metadata search",
			fn: func() (any, error) {
				return adapter.SearchByMetadata(context.Background(), model.CanonicalAlbum{})
			},
		},
		{
			name: "song isrc search",
			fn: func() (any, error) {
				return adapter.SearchSongByISRC(context.Background(), " ")
			},
		},
		{
			name: "song metadata search",
			fn: func() (any, error) {
				return adapter.SearchSongByMetadata(context.Background(), model.CanonicalSong{})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := tt.fn()
			require.NoError(t, err)
			assert.Nil(t, results)
		})
	}
}

func assertSingleAlbum(t *testing.T, candidates []model.CandidateAlbum, wantID string) {
	t.Helper()
	require.Len(t, candidates, 1)
	assert.Equal(t, wantID, candidates[0].CandidateID)
	assert.Contains(t, candidates[0].MatchURL, wantID)
}

func assertSingleSong(t *testing.T, candidates []model.CandidateSong, wantID string) {
	t.Helper()
	require.Len(t, candidates, 1)
	assert.Equal(t, wantID, candidates[0].CandidateID)
	assert.Contains(t, candidates[0].MatchURL, wantID)
}

func writeJSON(_ *testing.T, w http.ResponseWriter, payload any) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(payload); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(buf.Bytes())
}
