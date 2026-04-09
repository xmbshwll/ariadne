package applemusic

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
)

func TestAdapter(t *testing.T) {
	const (
		abbeyRoadRemastered = "Abbey Road (Remastered)"
		comeTogetherTitle   = "Come Together"
		comeTogetherISRC    = "GBAYE0601690"
	)

	lookupPayload := mustReadTestFile(t, "testdata/source-payload.json")
	searchPayload := `{
		"resultCount": 2,
		"results": [
			{
				"wrapperType": "collection",
				"collectionType": "Album",
				"artistId": 136975,
				"collectionId": 1474815798,
				"artistName": "The Beatles",
				"collectionName": "Abbey Road (2019 Mix)",
				"collectionViewUrl": "https://music.apple.com/us/album/abbey-road-2019-mix/1474815798?uo=4"
			},
			{
				"wrapperType": "collection",
				"collectionType": "Album",
				"artistId": 136975,
				"collectionId": 1441164426,
				"artistName": "The Beatles",
				"collectionName": "Abbey Road (Remastered)",
				"collectionViewUrl": "https://music.apple.com/us/album/abbey-road-remastered/1441164426?uo=4"
			}
		]
	}`
	lookup2019MixPayload := `{
		"resultCount": 2,
		"results": [
			{
				"wrapperType": "collection",
				"collectionType": "Album",
				"artistId": 136975,
				"collectionId": 1474815798,
				"artistName": "The Beatles",
				"collectionName": "Abbey Road (2019 Mix)",
				"collectionViewUrl": "https://music.apple.com/us/album/abbey-road-2019-mix/1474815798?uo=4",
				"artworkUrl100": "https://is1-ssl.mzstatic.com/image/thumb/Music211/v4/48/53/43/485343e3-dd6a-0034-faec-f4b6403f8108/13UMGIM63890.rgb.jpg/100x100bb.jpg",
				"trackCount": 17,
				"copyright": "℗ 2019 Calderstone Productions Limited",
				"releaseDate": "1969-09-26T07:00:00Z",
				"collectionExplicitness": "notExplicit"
			},
			{
				"wrapperType": "track",
				"kind": "song",
				"collectionId": 1474815798,
				"artistName": "The Beatles",
				"trackName": "Come Together",
				"discNumber": 1,
				"trackNumber": 1,
				"trackTimeMillis": 259227,
				"trackExplicitness": "notExplicit"
			}
		]
	}`
	officialAlbumPayload := fmt.Sprintf(`{
		"data": [{
			"id": "1441164426",
			"type": "albums",
			"attributes": {
				"artistName": "The Beatles",
				"name": "Abbey Road (Remastered)",
				"recordLabel": "UMC (Universal Music Catalogue)",
				"releaseDate": "1969-09-26",
				"trackCount": 18,
				"upc": "00602567713449",
				"url": "https://music.apple.com/gb/album/abbey-road-remastered/1441164426",
				"artwork": {"url": "https://image.test/{w}x{h}bb.jpg"}
			},
			"relationships": {
				"tracks": {"data": [
					{"id":"1441164430","type":"songs","attributes":{"artistName":"The Beatles","name":"Come Together","discNumber":1,"trackNumber":1,"durationInMillis":258947,"isrc":"%s","url":"https://music.apple.com/gb/album/come-together/1441164426?i=1441164430"}},
					{"id":"1441164582","type":"songs","attributes":{"artistName":"The Beatles","name":"Something","discNumber":1,"trackNumber":2,"durationInMillis":182293,"isrc":"GBAYE0601691","url":"https://music.apple.com/gb/album/something/1441164426?i=1441164582"}}
				]}
			}
		}]
	}`,
		comeTogetherISRC,
	)
	officialUPCSearchPayload := `{
		"data": [{
			"id": "401186200",
			"type": "albums",
			"attributes": {
				"artistName": "The Beatles",
				"name": "Abbey Road (Remastered)",
				"upc": "00602567713449",
				"url": "https://music.apple.com/gb/album/abbey-road-remastered/1441164426",
				"playParams": {"id": "401186200", "kind": "album"}
			}
		}]
	}`
	officialISRCSearchPayload := fmt.Sprintf(`{
		"data": [{
			"id": "1441164430",
			"type": "songs",
			"attributes": {
				"artistName": "The Beatles",
				"name": "Come Together",
				"isrc": "%s"
			},
			"relationships": {
				"albums": {"data": [{"id": "1441164426", "type": "albums"}]}
			}
		}]
	}`,
		comeTogetherISRC,
	)
	lookupSongPayload := fmt.Sprintf(`{
		"resultCount": 1,
		"results": [{
			"wrapperType": "track",
			"kind": "song",
			"artistId": 136975,
			"collectionId": 1441164426,
			"trackId": 1441164430,
			"artistName": "The Beatles",
			"collectionName": "Abbey Road (Remastered)",
			"collectionViewUrl": "https://music.apple.com/us/album/abbey-road-remastered/1441164426?uo=4",
			"trackName": "Come Together",
			"discNumber": 1,
			"trackNumber": 1,
			"trackTimeMillis": 258947,
			"trackIsrc": "%s",
			"releaseDate": "1969-09-26T07:00:00Z",
			"artworkUrl100": "https://image.test/100x100bb.jpg",
			"trackExplicitness": "notExplicit"
		}]
	}`,
		comeTogetherISRC,
	)
	searchSongPayload := fmt.Sprintf(`{
		"resultCount": 2,
		"results": [
			{
				"wrapperType": "track",
				"kind": "song",
				"artistId": 136975,
				"collectionId": 1441164426,
				"trackId": 1441164430,
				"artistName": "The Beatles",
				"collectionName": "Abbey Road (Remastered)",
				"collectionViewUrl": "https://music.apple.com/us/album/abbey-road-remastered/1441164426?uo=4",
				"trackName": "Come Together",
				"discNumber": 1,
				"trackNumber": 1,
				"trackTimeMillis": 258947,
				"trackIsrc": "%s",
				"releaseDate": "1969-09-26T07:00:00Z",
				"artworkUrl100": "https://image.test/100x100bb.jpg",
				"trackExplicitness": "notExplicit"
			},
			{
				"wrapperType": "track",
				"kind": "song",
				"artistId": 999,
				"collectionId": 555,
				"trackId": 999999,
				"artistName": "Tribute Band",
				"collectionName": "Abbey Road Live",
				"collectionViewUrl": "https://music.apple.com/us/album/abbey-road-live/555?uo=4",
				"trackName": "Come Together (Live)",
				"discNumber": 1,
				"trackNumber": 8,
				"trackTimeMillis": 300000,
				"trackIsrc": "LIVE0000001",
				"releaseDate": "2021-01-01T07:00:00Z",
				"artworkUrl100": "https://image.test/weak.jpg",
				"trackExplicitness": "notExplicit"
			}
		]
	}`,
		comeTogetherISRC,
	)
	keyPath := writeTestPrivateKey(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/lookup":
			if got := r.URL.Query().Get("country"); got != "us" && got != "gb" {
				http.Error(w, "missing country", http.StatusBadRequest)
				return
			}
			switch r.URL.Query().Get("id") {
			case "1441164426":
				_, _ = w.Write(lookupPayload)
			case "1474815798":
				_, _ = w.Write([]byte(lookup2019MixPayload))
			case "1441164430":
				_, _ = w.Write([]byte(lookupSongPayload))
			case "999999":
				_, _ = w.Write([]byte(`{"resultCount":1,"results":[{"wrapperType":"track","kind":"song","artistId":999,"collectionId":555,"trackId":999999,"artistName":"Tribute Band","collectionName":"Abbey Road Live","collectionViewUrl":"https://music.apple.com/us/album/abbey-road-live/555?uo=4","trackName":"Come Together (Live)","discNumber":1,"trackNumber":8,"trackTimeMillis":300000,"trackIsrc":"LIVE0000001","releaseDate":"2021-01-01T07:00:00Z","artworkUrl100":"https://image.test/weak.jpg","trackExplicitness":"notExplicit"}]}`))
			default:
				http.NotFound(w, r)
			}
		case "/search":
			if got := r.URL.Query().Get("country"); got != "gb" {
				http.Error(w, "expected gb storefront", http.StatusBadRequest)
				return
			}
			if r.URL.Query().Get("entity") == entitySong {
				_, _ = w.Write([]byte(searchSongPayload))
				return
			}
			_, _ = w.Write([]byte(searchPayload))
		case "/catalog/gb/albums":
			if r.Header.Get("Authorization") == "" {
				http.Error(w, "missing auth", http.StatusUnauthorized)
				return
			}
			if r.URL.Query().Get("filter[upc]") != "00602567713449" {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write([]byte(officialUPCSearchPayload))
		case "/catalog/gb/songs":
			if r.Header.Get("Authorization") == "" {
				http.Error(w, "missing auth", http.StatusUnauthorized)
				return
			}
			if r.URL.Query().Get("filter[isrc]") != comeTogetherISRC {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write([]byte(officialISRCSearchPayload))
		case "/catalog/gb/albums/1441164426":
			if r.Header.Get("Authorization") == "" {
				http.Error(w, "missing auth", http.StatusUnauthorized)
				return
			}
			_, _ = w.Write([]byte(officialAlbumPayload))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := New(server.Client(), WithLookupBaseURL(server.URL))
	authAdapter := New(
		server.Client(),
		WithLookupBaseURL(server.URL),
		WithAPIBaseURL(server.URL),
		WithDefaultStorefront("gb"),
		WithDeveloperTokenAuth("TEST12345", "TEAM123456", keyPath),
	)

	parsed := model.ParsedAlbumURL{
		Service:      model.ServiceAppleMusic,
		EntityType:   "album",
		ID:           "1441164426",
		CanonicalURL: "https://music.apple.com/us/album/abbey-road-remastered/1441164426",
		RegionHint:   "us",
		RawURL:       "https://music.apple.com/us/album/abbey-road-remastered/1441164426",
	}

	t.Run("fetch album", func(t *testing.T) {
		album, err := adapter.FetchAlbum(context.Background(), parsed)
		require.NoError(t, err)
		assert.Equal(t, abbeyRoadRemastered, album.Title)
		assert.Equal(t, "1441164426", album.SourceID)
		assert.Equal(t, "https://music.apple.com/us/album/abbey-road-remastered/1441164426", album.SourceURL)
		assert.Equal(t, 17, album.TrackCount)
		require.Len(t, album.Tracks, 17)
		assert.Equal(t, comeTogetherTitle, album.Tracks[0].Title)
		assert.Equal(t, 258947, album.Tracks[0].DurationMS)
		assert.NotEmpty(t, album.ArtworkURL)
		assert.Equal(t, "1969-09-26", album.ReleaseDate)
	})

	t.Run("search by metadata", func(t *testing.T) {
		results, err := adapter.SearchByMetadata(context.Background(), model.CanonicalAlbum{
			Title:      abbeyRoadRemastered,
			Artists:    []string{"The Beatles"},
			RegionHint: "gb",
		})
		require.NoError(t, err)
		require.Len(t, results, 2)
		assert.Equal(t, "1474815798", results[0].CandidateID)
		assert.Equal(t, "1441164426", results[1].CandidateID)
		assert.Equal(t, "https://music.apple.com/us/album/abbey-road-remastered/1441164426", results[1].MatchURL)
		assert.Equal(t, "gb", results[1].RegionHint)
	})

	t.Run("search by metadata uses adapter default storefront", func(t *testing.T) {
		defaultStorefrontAdapter := New(server.Client(), WithLookupBaseURL(server.URL), WithDefaultStorefront("gb"))
		results, err := defaultStorefrontAdapter.SearchByMetadata(context.Background(), model.CanonicalAlbum{
			Title:   "Abbey Road (Remastered)",
			Artists: []string{"The Beatles"},
		})
		require.NoError(t, err)
		require.NotEmpty(t, results)
		assert.Equal(t, "gb", results[0].RegionHint)
	})

	t.Run("search by upc without auth returns no results", func(t *testing.T) {
		results, err := adapter.SearchByUPC(context.Background(), "123")
		require.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("search by isrc without auth returns no results", func(t *testing.T) {
		results, err := adapter.SearchByISRC(context.Background(), []string{"ABC"})
		require.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("search by upc with official auth", func(t *testing.T) {
		results, err := authAdapter.SearchByUPC(context.Background(), "00602567713449")
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, "1441164426", results[0].CandidateID)
		assert.Equal(t, "https://music.apple.com/gb/album/abbey-road-remastered/1441164426", results[0].MatchURL)
		assert.Equal(t, "gb", results[0].RegionHint)
		assert.Equal(t, "00602567713449", results[0].UPC)
	})

	t.Run("search by isrc with official auth", func(t *testing.T) {
		results, err := authAdapter.SearchByISRC(context.Background(), []string{"GBAYE0601690"})
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, "1441164426", results[0].CandidateID)
		require.Len(t, results[0].Tracks, 2)
		assert.Equal(t, "GBAYE0601690", results[0].Tracks[0].ISRC)
	})

	t.Run("fetch song", func(t *testing.T) {
		song, err := adapter.FetchSong(context.Background(), model.ParsedAlbumURL{
			Service:      model.ServiceAppleMusic,
			EntityType:   entitySong,
			ID:           "1441164430",
			CanonicalURL: "https://music.apple.com/us/album/abbey-road-remastered/1441164426?i=1441164430",
			RegionHint:   "us",
		})
		require.NoError(t, err)
		assert.Equal(t, comeTogetherTitle, song.Title)
		assert.Equal(t, "Abbey Road (Remastered)", song.AlbumTitle)
		assert.Equal(t, 1, song.TrackNumber)
		assert.Equal(t, comeTogetherISRC, song.ISRC)
	})

	t.Run("search song by metadata", func(t *testing.T) {
		results, err := adapter.SearchSongByMetadata(context.Background(), model.CanonicalSong{
			Title:      comeTogetherTitle,
			Artists:    []string{"The Beatles"},
			RegionHint: "gb",
		})
		require.NoError(t, err)
		require.Len(t, results, 2)
		assert.Equal(t, "1441164430", results[0].CandidateID)
		assert.Equal(t, "Abbey Road (Remastered)", results[0].AlbumTitle)
		assert.Equal(t, comeTogetherISRC, results[0].ISRC)
	})

	t.Run("search song by isrc with official auth", func(t *testing.T) {
		results, err := authAdapter.SearchSongByISRC(context.Background(), comeTogetherISRC)
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, "1441164430", results[0].CandidateID)
		assert.Equal(t, comeTogetherTitle, results[0].Title)
	})
}

func mustReadTestFile(t *testing.T, relativePath string) []byte {
	t.Helper()
	path := filepath.Clean(relativePath)
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return content
}

func writeTestPrivateKey(t *testing.T) string {
	t.Helper()
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(privateKey)
	require.NoError(t, err)
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	path := filepath.Join(t.TempDir(), "AuthKey_TEST12345.p8")
	require.NoError(t, os.WriteFile(path, pemBytes, 0o600))
	return path
}
