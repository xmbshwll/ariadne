package applemusic

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/xmbshwll/ariadne/internal/model"
)

func TestAdapter(t *testing.T) {
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
	officialAlbumPayload := `{
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
					{"id":"1441164430","type":"songs","attributes":{"artistName":"The Beatles","name":"Come Together","discNumber":1,"trackNumber":1,"durationInMillis":258947,"isrc":"GBAYE0601690","url":"https://music.apple.com/gb/album/come-together/1441164426?i=1441164430"}},
					{"id":"1441164582","type":"songs","attributes":{"artistName":"The Beatles","name":"Something","discNumber":1,"trackNumber":2,"durationInMillis":182293,"isrc":"GBAYE0601691","url":"https://music.apple.com/gb/album/something/1441164426?i=1441164582"}}
				]}
			}
		}]
	}`
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
	officialISRCSearchPayload := `{
		"data": [{
			"id": "1441164430",
			"type": "songs",
			"attributes": {
				"artistName": "The Beatles",
				"name": "Come Together",
				"isrc": "GBAYE0601690"
			},
			"relationships": {
				"albums": {"data": [{"id": "1441164426", "type": "albums"}]}
			}
		}]
	}`
	lookupSongPayload := `{
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
			"releaseDate": "1969-09-26T07:00:00Z",
			"artworkUrl100": "https://image.test/100x100bb.jpg",
			"trackExplicitness": "notExplicit"
		}]
	}`
	searchSongPayload := `{
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
				"releaseDate": "2021-01-01T07:00:00Z",
				"artworkUrl100": "https://image.test/weak.jpg",
				"trackExplicitness": "notExplicit"
			}
		]
	}`
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
				_, _ = w.Write([]byte(`{"resultCount":1,"results":[{"wrapperType":"track","kind":"song","artistId":999,"collectionId":555,"trackId":999999,"artistName":"Tribute Band","collectionName":"Abbey Road Live","collectionViewUrl":"https://music.apple.com/us/album/abbey-road-live/555?uo=4","trackName":"Come Together (Live)","discNumber":1,"trackNumber":8,"trackTimeMillis":300000,"releaseDate":"2021-01-01T07:00:00Z","artworkUrl100":"https://image.test/weak.jpg","trackExplicitness":"notExplicit"}]}`))
			default:
				http.NotFound(w, r)
			}
		case "/search":
			if got := r.URL.Query().Get("country"); got != "gb" {
				http.Error(w, "expected gb storefront", http.StatusBadRequest)
				return
			}
			if r.URL.Query().Get("entity") == "song" {
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
			if r.URL.Query().Get("filter[isrc]") != "GBAYE0601690" {
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
		if err != nil {
			t.Fatalf("FetchAlbum error: %v", err)
		}
		if album.Title != "Abbey Road (Remastered)" {
			t.Fatalf("title = %q", album.Title)
		}
		if album.SourceID != "1441164426" {
			t.Fatalf("source id = %q", album.SourceID)
		}
		if album.SourceURL != "https://music.apple.com/us/album/abbey-road-remastered/1441164426" {
			t.Fatalf("source url = %q", album.SourceURL)
		}
		if album.TrackCount != 17 {
			t.Fatalf("track count = %d", album.TrackCount)
		}
		if len(album.Tracks) != 17 {
			t.Fatalf("tracks len = %d", len(album.Tracks))
		}
		if album.Tracks[0].Title != "Come Together" {
			t.Fatalf("first track title = %q", album.Tracks[0].Title)
		}
		if album.Tracks[0].DurationMS != 258947 {
			t.Fatalf("first track duration = %d", album.Tracks[0].DurationMS)
		}
		if album.ArtworkURL == "" {
			t.Fatalf("expected artwork url")
		}
		if album.ReleaseDate != "1969-09-26" {
			t.Fatalf("release date = %q", album.ReleaseDate)
		}
	})

	t.Run("search by metadata", func(t *testing.T) {
		results, err := adapter.SearchByMetadata(context.Background(), model.CanonicalAlbum{
			Title:      "Abbey Road (Remastered)",
			Artists:    []string{"The Beatles"},
			RegionHint: "gb",
		})
		if err != nil {
			t.Fatalf("SearchByMetadata error: %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("result count = %d, want 2", len(results))
		}
		if results[0].CandidateID != "1474815798" {
			t.Fatalf("first candidate id = %q, want 1474815798", results[0].CandidateID)
		}
		if results[1].CandidateID != "1441164426" {
			t.Fatalf("second candidate id = %q, want 1441164426", results[1].CandidateID)
		}
		if results[1].MatchURL != "https://music.apple.com/us/album/abbey-road-remastered/1441164426" {
			t.Fatalf("second candidate url = %q", results[1].MatchURL)
		}
		if results[1].RegionHint != "gb" {
			t.Fatalf("second candidate region hint = %q, want gb", results[1].RegionHint)
		}
	})

	t.Run("search by metadata uses adapter default storefront", func(t *testing.T) {
		defaultStorefrontAdapter := New(server.Client(), WithLookupBaseURL(server.URL), WithDefaultStorefront("gb"))
		results, err := defaultStorefrontAdapter.SearchByMetadata(context.Background(), model.CanonicalAlbum{
			Title:   "Abbey Road (Remastered)",
			Artists: []string{"The Beatles"},
		})
		if err != nil {
			t.Fatalf("SearchByMetadata error: %v", err)
		}
		if len(results) == 0 {
			t.Fatalf("expected results")
		}
		if results[0].RegionHint != "gb" {
			t.Fatalf("first candidate region hint = %q, want gb", results[0].RegionHint)
		}
	})

	t.Run("search by upc without auth returns no results", func(t *testing.T) {
		results, err := adapter.SearchByUPC(context.Background(), "123")
		if err != nil {
			t.Fatalf("SearchByUPC error: %v", err)
		}
		if len(results) != 0 {
			t.Fatalf("result count = %d, want 0", len(results))
		}
	})

	t.Run("search by isrc without auth returns no results", func(t *testing.T) {
		results, err := adapter.SearchByISRC(context.Background(), []string{"ABC"})
		if err != nil {
			t.Fatalf("SearchByISRC error: %v", err)
		}
		if len(results) != 0 {
			t.Fatalf("result count = %d, want 0", len(results))
		}
	})

	t.Run("search by upc with official auth", func(t *testing.T) {
		results, err := authAdapter.SearchByUPC(context.Background(), "00602567713449")
		if err != nil {
			t.Fatalf("SearchByUPC error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("result count = %d, want 1", len(results))
		}
		if results[0].CandidateID != "1441164426" {
			t.Fatalf("candidate id = %q, want 1441164426", results[0].CandidateID)
		}
		if results[0].MatchURL != "https://music.apple.com/gb/album/abbey-road-remastered/1441164426" {
			t.Fatalf("candidate url = %q", results[0].MatchURL)
		}
		if results[0].RegionHint != "gb" {
			t.Fatalf("candidate region hint = %q, want gb", results[0].RegionHint)
		}
		if results[0].UPC != "00602567713449" {
			t.Fatalf("candidate upc = %q", results[0].UPC)
		}
	})

	t.Run("search by isrc with official auth", func(t *testing.T) {
		results, err := authAdapter.SearchByISRC(context.Background(), []string{"GBAYE0601690"})
		if err != nil {
			t.Fatalf("SearchByISRC error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("result count = %d, want 1", len(results))
		}
		if results[0].CandidateID != "1441164426" {
			t.Fatalf("candidate id = %q, want 1441164426", results[0].CandidateID)
		}
		if len(results[0].Tracks) != 2 {
			t.Fatalf("track count = %d, want 2", len(results[0].Tracks))
		}
		if results[0].Tracks[0].ISRC != "GBAYE0601690" {
			t.Fatalf("first isrc = %q", results[0].Tracks[0].ISRC)
		}
	})

	t.Run("fetch song", func(t *testing.T) {
		song, err := adapter.FetchSong(context.Background(), model.ParsedAlbumURL{
			Service:      model.ServiceAppleMusic,
			EntityType:   "song",
			ID:           "1441164430",
			CanonicalURL: "https://music.apple.com/us/album/abbey-road-remastered/1441164426?i=1441164430",
			RegionHint:   "us",
		})
		if err != nil {
			t.Fatalf("FetchSong error: %v", err)
		}
		if song.Title != "Come Together" {
			t.Fatalf("title = %q", song.Title)
		}
		if song.AlbumTitle != "Abbey Road (Remastered)" {
			t.Fatalf("album title = %q", song.AlbumTitle)
		}
		if song.TrackNumber != 1 {
			t.Fatalf("track number = %d", song.TrackNumber)
		}
	})

	t.Run("search song by metadata", func(t *testing.T) {
		results, err := adapter.SearchSongByMetadata(context.Background(), model.CanonicalSong{
			Title:      "Come Together",
			Artists:    []string{"The Beatles"},
			RegionHint: "gb",
		})
		if err != nil {
			t.Fatalf("SearchSongByMetadata error: %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("result count = %d, want 2", len(results))
		}
		if results[0].CandidateID != "1441164430" {
			t.Fatalf("first candidate id = %q, want 1441164430", results[0].CandidateID)
		}
		if results[0].AlbumTitle != "Abbey Road (Remastered)" {
			t.Fatalf("first album title = %q", results[0].AlbumTitle)
		}
	})

	t.Run("search song by isrc with official auth", func(t *testing.T) {
		results, err := authAdapter.SearchSongByISRC(context.Background(), "GBAYE0601690")
		if err != nil {
			t.Fatalf("SearchSongByISRC error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("result count = %d, want 1", len(results))
		}
		if results[0].CandidateID != "1441164430" {
			t.Fatalf("candidate id = %q, want 1441164430", results[0].CandidateID)
		}
		if results[0].Title != "Come Together" {
			t.Fatalf("title = %q", results[0].Title)
		}
	})
}

func mustReadTestFile(t *testing.T, relativePath string) []byte {
	t.Helper()
	path := filepath.Clean(relativePath)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return content
}

func writeTestPrivateKey(t *testing.T) string {
	t.Helper()
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate private key: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("marshal private key: %v", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	path := filepath.Join(t.TempDir(), "AuthKey_TEST12345.p8")
	if err := os.WriteFile(path, pemBytes, 0o600); err != nil {
		t.Fatalf("write private key: %v", err)
	}
	return path
}
