package applemusic_test

import (
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

	applemusic "github.com/xmbshwll/ariadne/internal/adapters/applemusic"

	"github.com/stretchr/testify/require"

	"github.com/xmbshwll/ariadne/internal/model"
)

const (
	abbeyRoadRemastered = "Abbey Road (Remastered)"
	comeTogetherTitle   = "Come Together"
	comeTogetherISRC    = "GBAYE0601690"
)

type testPayloads struct {
	lookup         []byte
	lookup2019Mix  []byte
	officialAlbum  []byte
	officialUPC    []byte
	officialISRC   []byte
	lookupSong     []byte
	searchAlbum    []byte
	searchSong     []byte
	lookupWeakSong []byte
	lookupNonSong  []byte
}

type testFixture struct {
	httpClient  *http.Client
	serverURL   string
	adapter     *applemusic.Adapter
	authAdapter *applemusic.Adapter
	parsed      model.ParsedAlbumURL
}

func buildTestPayloads(t *testing.T) testPayloads {
	t.Helper()

	return testPayloads{
		lookup:         mustReadTestFile(t, "testdata/source-payload.json"),
		lookup2019Mix:  mustReadTestFile(t, "testdata/lookup-2019-mix.json"),
		officialAlbum:  mustReadTestFile(t, "testdata/official-album.json"),
		officialUPC:    mustReadTestFile(t, "testdata/official-upc-search.json"),
		officialISRC:   mustReadTestFile(t, "testdata/official-isrc-search.json"),
		lookupSong:     mustReadTestFile(t, "testdata/lookup-song.json"),
		searchAlbum:    mustReadTestFile(t, "testdata/search-album.json"),
		searchSong:     mustReadTestFile(t, "testdata/search-song.json"),
		lookupWeakSong: mustReadTestFile(t, "testdata/lookup-weak-song.json"),
		lookupNonSong:  mustReadTestFile(t, "testdata/lookup-non-song.json"),
	}
}

func newTestFixture(t *testing.T, payloads testPayloads) testFixture {
	t.Helper()

	keyPath := writeTestPrivateKey(t)
	server := newTestServer(payloads)
	client := server.Client()
	serverURL := server.URL
	t.Cleanup(server.Close)

	return testFixture{
		httpClient: client,
		serverURL:  serverURL,
		adapter:    applemusic.New(client, applemusic.WithLookupBaseURL(serverURL)),
		authAdapter: applemusic.New(
			client,
			applemusic.WithLookupBaseURL(serverURL),
			applemusic.WithAPIBaseURL(serverURL),
			applemusic.WithDefaultStorefront("gb"),
			applemusic.WithDeveloperTokenAuth("TEST12345", "TEAM123456", keyPath),
		),
		parsed: model.ParsedAlbumURL{
			Service:      model.ServiceAppleMusic,
			EntityType:   model.EntityTypeAlbum,
			ID:           "1441164426",
			CanonicalURL: "https://music.apple.com/us/album/abbey-road-remastered/1441164426",
			RegionHint:   "us",
			RawURL:       "https://music.apple.com/us/album/abbey-road-remastered/1441164426",
		},
	}
}

func newOfficialTestAdapter(t *testing.T, registerHandlers func(*http.ServeMux)) *applemusic.Adapter {
	t.Helper()
	keyPath := writeTestPrivateKey(t)
	mux := http.NewServeMux()
	registerHandlers(mux)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return applemusic.New(
		server.Client(),
		applemusic.WithAPIBaseURL(server.URL),
		applemusic.WithDefaultStorefront("gb"),
		applemusic.WithDeveloperTokenAuth("TEST12345", "TEAM123456", keyPath),
	)
}

func newTestServer(payloads testPayloads) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/lookup", lookupHandler(payloads))
	mux.HandleFunc("/search", searchHandler(payloads))
	mux.HandleFunc("/catalog/gb/albums", officialAlbumsHandler(payloads))
	mux.HandleFunc("/catalog/gb/songs", officialSongsHandler(payloads))
	mux.HandleFunc("/catalog/gb/albums/1441164426", officialAlbumHandler(payloads))
	return httptest.NewServer(mux)
}

func lookupHandler(payloads testPayloads) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("country"); got != "us" && got != "gb" {
			http.Error(w, "missing country", http.StatusBadRequest)
			return
		}

		var payload []byte
		switch r.URL.Query().Get("id") {
		case "1441164426":
			payload = payloads.lookup
		case "1474815798":
			payload = payloads.lookup2019Mix
		case "1441164430":
			payload = payloads.lookupSong
		case "999999":
			payload = payloads.lookupWeakSong
		case "123456789":
			payload = payloads.lookupNonSong
		default:
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(payload)
	}
}

func searchHandler(payloads testPayloads) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("country"); got != "gb" {
			http.Error(w, "expected gb storefront", http.StatusBadRequest)
			return
		}
		if r.URL.Query().Get("entity") == applemusic.EntitySong {
			_, _ = w.Write(payloads.searchSong)
			return
		}
		_, _ = w.Write(payloads.searchAlbum)
	}
}

func officialAlbumsHandler(payloads testPayloads) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !requireOfficialAuth(w, r) {
			return
		}
		if r.URL.Query().Get("filter[upc]") != "00602567713449" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(payloads.officialUPC)
	}
}

func officialSongsHandler(payloads testPayloads) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !requireOfficialAuth(w, r) {
			return
		}
		if r.URL.Query().Get("filter[isrc]") != comeTogetherISRC {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(payloads.officialISRC)
	}
}

func officialAlbumHandler(payloads testPayloads) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !requireOfficialAuth(w, r) {
			return
		}
		_, _ = w.Write(payloads.officialAlbum)
	}
}

func requireOfficialAuth(w http.ResponseWriter, r *http.Request) bool {
	if r.Header.Get("Authorization") == "" {
		http.Error(w, "missing auth", http.StatusUnauthorized)
		return false
	}
	return true
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
