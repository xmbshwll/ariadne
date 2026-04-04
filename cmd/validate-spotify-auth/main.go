package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xmbshwll/ariadne/internal/config"
	"github.com/xmbshwll/ariadne/internal/parse"
)

const (
	defaultSpotifyAPIBaseURL  = "https://api.spotify.com/v1"
	defaultSpotifyAuthBaseURL = "https://accounts.spotify.com/api"
	defaultSampleURLPath      = "service-samples/spotify/sample-url.txt"
	defaultOutputDir          = "service-samples/spotify"
	searchLimit               = 5
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	opts, err := parseFlags(args)
	if err != nil {
		return err
	}

	appConfig := config.Load()
	if !appConfig.Spotify.Enabled() {
		return errors.New("SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET must be set")
	}

	rawURL, err := loadSampleURL(opts.sampleURL, opts.sampleURLPath)
	if err != nil {
		return err
	}
	parsed, err := parse.SpotifyAlbumURL(rawURL)
	if err != nil {
		return fmt.Errorf("parse sample spotify album url: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	token, err := fetchToken(ctx, opts.authBaseURL, appConfig.Spotify.ClientID, appConfig.Spotify.ClientSecret)
	if err != nil {
		return err
	}

	albumPath := "/albums/" + parsed.ID
	albumBody, err := getAPI(ctx, opts.apiBaseURL+albumPath, token)
	if err != nil {
		return fmt.Errorf("fetch spotify album payload: %w", err)
	}

	var album map[string]any
	if err := json.Unmarshal(albumBody, &album); err != nil {
		return fmt.Errorf("decode album payload: %w", err)
	}

	upc := nestedString(album, "external_ids", "upc")
	if upc == "" {
		return errors.New("album payload did not include external_ids.upc")
	}
	isrcs, err := collectTrackISRCs(ctx, opts.apiBaseURL, token, album)
	if err != nil {
		return fmt.Errorf("collect spotify track isrcs: %w", err)
	}
	if len(isrcs) == 0 {
		return errors.New("spotify track detail payloads did not include any external_ids.isrc values")
	}

	metadataQuery := metadataQuery(album)
	if metadataQuery == "" {
		return errors.New("album payload did not provide enough metadata for search validation")
	}

	upcBody, err := getAPI(ctx, opts.apiBaseURL+"/search?q="+url.QueryEscape("upc:"+upc)+"&type=album&limit="+fmt.Sprint(searchLimit), token)
	if err != nil {
		return fmt.Errorf("search spotify by upc: %w", err)
	}
	isrcBody, err := getAPI(ctx, opts.apiBaseURL+"/search?q="+url.QueryEscape("isrc:"+isrcs[0])+"&type=track&limit="+fmt.Sprint(searchLimit), token)
	if err != nil {
		return fmt.Errorf("search spotify by isrc: %w", err)
	}
	metadataBody, err := getAPI(ctx, opts.apiBaseURL+"/search?q="+url.QueryEscape(metadataQuery)+"&type=album&limit="+fmt.Sprint(searchLimit), token)
	if err != nil {
		return fmt.Errorf("search spotify by metadata: %w", err)
	}

	if err := os.MkdirAll(opts.outputDir, 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	summary := map[string]any{
		"sample_url":         rawURL,
		"album_id":           parsed.ID,
		"canonical_url":      parsed.CanonicalURL,
		"title":              strings.TrimSpace(asString(album["name"])),
		"artists":            albumArtists(album),
		"release_date":       strings.TrimSpace(asString(album["release_date"])),
		"label":              strings.TrimSpace(asString(album["label"])),
		"upc":                upc,
		"track_isrc_samples": isrcs,
		"generated_at":       time.Now().UTC().Format(time.RFC3339),
		"artifacts": map[string]string{
			"source_payload_api":    filepath.ToSlash(filepath.Join(opts.outputDir, "source-payload-api.json")),
			"search_upc_results":    filepath.ToSlash(filepath.Join(opts.outputDir, "search-upc-results.json")),
			"search_isrc_results":   filepath.ToSlash(filepath.Join(opts.outputDir, "search-isrc-results.json")),
			"search_metadata":       filepath.ToSlash(filepath.Join(opts.outputDir, "search-metadata-results.json")),
			"authenticated_summary": filepath.ToSlash(filepath.Join(opts.outputDir, "authenticated-summary.json")),
		},
	}

	if err := writePrettyJSON(filepath.Join(opts.outputDir, "source-payload-api.json"), albumBody); err != nil {
		return err
	}
	if err := writePrettyJSON(filepath.Join(opts.outputDir, "search-upc-results.json"), upcBody); err != nil {
		return err
	}
	if err := writePrettyJSON(filepath.Join(opts.outputDir, "search-isrc-results.json"), isrcBody); err != nil {
		return err
	}
	if err := writePrettyJSON(filepath.Join(opts.outputDir, "search-metadata-results.json"), metadataBody); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(opts.outputDir, "authenticated-summary.json"), summary); err != nil {
		return err
	}

	fmt.Printf("wrote Spotify authenticated artifacts to %s\n", opts.outputDir)
	return nil
}

type options struct {
	sampleURL     string
	sampleURLPath string
	outputDir     string
	apiBaseURL    string
	authBaseURL   string
}

func parseFlags(args []string) (options, error) {
	opts := options{
		sampleURLPath: defaultSampleURLPath,
		outputDir:     defaultOutputDir,
		apiBaseURL:    defaultSpotifyAPIBaseURL,
		authBaseURL:   defaultSpotifyAuthBaseURL,
	}

	fs := flag.NewFlagSet("validate-spotify-auth", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&opts.sampleURL, "url", "", "spotify album url to validate")
	fs.StringVar(&opts.sampleURLPath, "sample-url-file", opts.sampleURLPath, "path to file containing spotify album url")
	fs.StringVar(&opts.outputDir, "out-dir", opts.outputDir, "directory for validation artifacts")
	fs.StringVar(&opts.apiBaseURL, "api-base-url", opts.apiBaseURL, "spotify api base url")
	fs.StringVar(&opts.authBaseURL, "auth-base-url", opts.authBaseURL, "spotify auth base url")
	if err := fs.Parse(args); err != nil {
		return options{}, errors.New("usage: go run ./cmd/validate-spotify-auth [-url <spotify-album-url>] [-sample-url-file <path>] [-out-dir <dir>]")
	}
	if len(fs.Args()) != 0 {
		return options{}, errors.New("usage: go run ./cmd/validate-spotify-auth [-url <spotify-album-url>] [-sample-url-file <path>] [-out-dir <dir>]")
	}
	return opts, nil
}

func loadSampleURL(rawURL string, path string) (string, error) {
	if strings.TrimSpace(rawURL) != "" {
		return strings.TrimSpace(rawURL), nil
	}
	content, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return "", fmt.Errorf("read spotify sample url file: %w", err)
	}
	value := strings.TrimSpace(string(content))
	if value == "" {
		return "", fmt.Errorf("spotify sample url file %s is empty", path)
	}
	return value, nil
}

func fetchToken(ctx context.Context, authBaseURL, clientID, clientSecret string) (string, error) {
	form := url.Values{}
	form.Set("grant_type", "client_credentials")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(authBaseURL, "/")+"/token", bytes.NewBufferString(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("build spotify token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(clientID+":"+clientSecret)))
	req.Header.Set("User-Agent", "ariadne/0.1 (+https://github.com/xmbshwll/ariadne)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute spotify token request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read spotify token response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected spotify token status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var token struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &token); err != nil {
		return "", fmt.Errorf("decode spotify token response: %w", err)
	}
	if token.AccessToken == "" {
		return "", errors.New("spotify token response did not include access_token")
	}
	return token.AccessToken, nil
}

func getAPI(ctx context.Context, endpoint string, token string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build spotify api request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", "ariadne/0.1 (+https://github.com/xmbshwll/ariadne)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute spotify api request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read spotify api response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected spotify api status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

func metadataQuery(album map[string]any) string {
	title := strings.TrimSpace(asString(album["name"]))
	artists := albumArtists(album)
	if title == "" || len(artists) == 0 {
		return ""
	}
	return fmt.Sprintf("album:%s artist:%s", title, artists[0])
}

func albumArtists(album map[string]any) []string {
	items, _ := album["artists"].([]any)
	artists := make([]string, 0, len(items))
	for _, item := range items {
		artist, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name := strings.TrimSpace(asString(artist["name"]))
		if name == "" {
			continue
		}
		artists = append(artists, name)
	}
	return artists
}

func collectTrackISRCs(ctx context.Context, apiBaseURL string, token string, album map[string]any) ([]string, error) {
	trackIDs := albumTrackIDs(album)
	if len(trackIDs) == 0 {
		return nil, nil
	}

	seen := map[string]struct{}{}
	isrcs := make([]string, 0, len(trackIDs))
	for _, trackID := range trackIDs {
		body, err := getAPI(ctx, strings.TrimRight(apiBaseURL, "/")+"/tracks/"+trackID, token)
		if err != nil {
			return nil, err
		}
		var track map[string]any
		if err := json.Unmarshal(body, &track); err != nil {
			return nil, fmt.Errorf("decode spotify track details payload: %w", err)
		}
		externalIDs, ok := track["external_ids"].(map[string]any)
		if !ok {
			continue
		}
		isrc := strings.TrimSpace(asString(externalIDs["isrc"]))
		if isrc == "" {
			continue
		}
		if _, exists := seen[isrc]; exists {
			continue
		}
		seen[isrc] = struct{}{}
		isrcs = append(isrcs, isrc)
		if len(isrcs) >= 5 {
			return isrcs, nil
		}
	}
	return isrcs, nil
}

func albumTrackIDs(album map[string]any) []string {
	tracks, ok := album["tracks"].(map[string]any)
	if !ok {
		return nil
	}
	items, _ := tracks["items"].([]any)
	ids := make([]string, 0, len(items))
	for _, item := range items {
		track, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id := strings.TrimSpace(asString(track["id"]))
		if id == "" {
			continue
		}
		ids = append(ids, id)
	}
	return ids
}

func nestedString(root map[string]any, keys ...string) string {
	var current any = root
	for _, key := range keys {
		next, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = next[key]
	}
	return strings.TrimSpace(asString(current))
}

func asString(value any) string {
	text, _ := value.(string)
	return text
}

func writePrettyJSON(path string, raw []byte) error {
	var payload any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("decode raw json for %s: %w", path, err)
	}
	return writeJSON(path, payload)
}

func writeJSON(path string, payload any) error {
	content, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	content = append(content, '\n')
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
