package main

import (
	"context"
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

	"github.com/xmbshwll/ariadne/internal/applemusicauth"
	"github.com/xmbshwll/ariadne/internal/config"
	"github.com/xmbshwll/ariadne/internal/parse"
)

const (
	defaultAPIBaseURL  = "https://api.music.apple.com/v1"
	defaultSampleURL   = "service-samples/apple-music/sample-url.txt"
	defaultOutputDir   = "service-samples/apple-music"
	defaultSearchLimit = 5
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type options struct {
	sampleURL     string
	sampleURLPath string
	outputDir     string
	apiBaseURL    string
	storefront    string
}

func run(args []string) error {
	opts, err := parseFlags(args)
	if err != nil {
		return err
	}

	appConfig := config.Load()
	if !appConfig.AppleMusic.AuthEnabled() {
		return errors.New("APPLE_MUSIC_KEY_ID, APPLE_MUSIC_TEAM_ID, and APPLE_MUSIC_PRIVATE_KEY_PATH must be set")
	}
	developerToken, err := applemusicauth.GenerateDeveloperToken(applemusicauth.Config{
		KeyID:          appConfig.AppleMusic.KeyID,
		TeamID:         appConfig.AppleMusic.TeamID,
		PrivateKeyPath: appConfig.AppleMusic.PrivateKeyPath,
	}, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("generate apple music developer token: %w", err)
	}

	rawURL, err := loadSampleURL(opts.sampleURL, opts.sampleURLPath)
	if err != nil {
		return err
	}
	parsed, err := parse.AppleMusicAlbumURL(rawURL)
	if err != nil {
		return fmt.Errorf("parse sample apple music album url: %w", err)
	}

	storefront := strings.ToLower(strings.TrimSpace(opts.storefront))
	if storefront == "" {
		storefront = parsed.RegionHint
	}
	if storefront == "" {
		storefront = appConfig.AppleMusic.Storefront
	}
	if storefront == "" {
		storefront = "us"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	albumBody, err := getAPI(ctx, opts.apiBaseURL+"/catalog/"+storefront+"/albums/"+parsed.ID+"?include=tracks", developerToken)
	if err != nil {
		return fmt.Errorf("fetch official apple music album payload: %w", err)
	}

	var albumPayload map[string]any
	if err := json.Unmarshal(albumBody, &albumPayload); err != nil {
		return fmt.Errorf("decode official apple music album payload: %w", err)
	}

	albumData := firstResource(albumPayload)
	if albumData == nil {
		return errors.New("official apple music album payload did not include a data resource")
	}

	attributes := nestedMap(albumData, "attributes")
	title := strings.TrimSpace(asString(attributes["name"]))
	artist := strings.TrimSpace(asString(attributes["artistName"]))
	releaseDate := strings.TrimSpace(asString(attributes["releaseDate"]))
	label := strings.TrimSpace(asString(attributes["recordLabel"]))
	upc := strings.TrimSpace(asString(attributes["upc"]))
	isrcs := albumISRCs(albumData)
	metadataQuery := strings.TrimSpace(strings.Join([]string{title, artist}, " "))
	if metadataQuery == "" {
		return errors.New("official apple music album payload did not provide enough metadata for search validation")
	}

	metadataBody, err := getAPI(ctx, opts.apiBaseURL+"/catalog/"+storefront+"/search?types=albums&limit="+fmt.Sprint(defaultSearchLimit)+"&term="+url.QueryEscape(metadataQuery), developerToken)
	if err != nil {
		return fmt.Errorf("search official apple music metadata: %w", err)
	}

	var upcBody []byte
	if upc != "" {
		upcBody, err = getAPI(ctx, opts.apiBaseURL+"/catalog/"+storefront+"/albums?filter[upc]="+url.QueryEscape(upc), developerToken)
		if err != nil {
			return fmt.Errorf("search official apple music by upc: %w", err)
		}
	}

	var isrcBody []byte
	if len(isrcs) > 0 {
		isrcBody, err = getAPI(ctx, opts.apiBaseURL+"/catalog/"+storefront+"/songs?filter[isrc]="+url.QueryEscape(isrcs[0]), developerToken)
		if err != nil {
			return fmt.Errorf("search official apple music by isrc: %w", err)
		}
	}

	if err := os.MkdirAll(opts.outputDir, 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	summary := map[string]any{
		"sample_url":         rawURL,
		"album_id":           parsed.ID,
		"canonical_url":      parsed.CanonicalURL,
		"storefront":         storefront,
		"auth_mode":          "generated_p8_token",
		"title":              title,
		"artists":            nonEmptyStrings(artist),
		"release_date":       releaseDate,
		"label":              label,
		"upc":                upc,
		"track_isrc_samples": isrcs,
		"generated_at":       time.Now().UTC().Format(time.RFC3339),
		"artifacts": map[string]string{
			"source_payload_official":  filepath.ToSlash(filepath.Join(opts.outputDir, "source-payload-official.json")),
			"search_metadata_official": filepath.ToSlash(filepath.Join(opts.outputDir, "search-metadata-official.json")),
			"search_upc_official":      filepath.ToSlash(filepath.Join(opts.outputDir, "search-upc-official.json")),
			"search_isrc_official":     filepath.ToSlash(filepath.Join(opts.outputDir, "search-isrc-official.json")),
			"official_summary":         filepath.ToSlash(filepath.Join(opts.outputDir, "official-summary.json")),
		},
	}

	if err := writePrettyJSON(filepath.Join(opts.outputDir, "source-payload-official.json"), albumBody); err != nil {
		return err
	}
	if err := writePrettyJSON(filepath.Join(opts.outputDir, "search-metadata-official.json"), metadataBody); err != nil {
		return err
	}
	if len(upcBody) > 0 {
		if err := writePrettyJSON(filepath.Join(opts.outputDir, "search-upc-official.json"), upcBody); err != nil {
			return err
		}
	}
	if len(isrcBody) > 0 {
		if err := writePrettyJSON(filepath.Join(opts.outputDir, "search-isrc-official.json"), isrcBody); err != nil {
			return err
		}
	}
	if err := writeJSON(filepath.Join(opts.outputDir, "official-summary.json"), summary); err != nil {
		return err
	}

	fmt.Printf("wrote Apple Music official artifacts to %s\n", opts.outputDir)
	return nil
}

func parseFlags(args []string) (options, error) {
	opts := options{
		sampleURLPath: defaultSampleURL,
		outputDir:     defaultOutputDir,
		apiBaseURL:    defaultAPIBaseURL,
	}

	fs := flag.NewFlagSet("validate-apple-music-official", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&opts.sampleURL, "url", "", "apple music album url to validate")
	fs.StringVar(&opts.sampleURLPath, "sample-url-file", opts.sampleURLPath, "path to file containing apple music album url")
	fs.StringVar(&opts.outputDir, "out-dir", opts.outputDir, "directory for validation artifacts")
	fs.StringVar(&opts.apiBaseURL, "api-base-url", opts.apiBaseURL, "apple music api base url")
	fs.StringVar(&opts.storefront, "storefront", "", "apple music storefront override")
	if err := fs.Parse(args); err != nil {
		return options{}, errors.New("usage: go run ./cmd/validate-apple-music-official [-url <apple-music-album-url>] [-sample-url-file <path>] [-out-dir <dir>] [-storefront <code>]")
	}
	if len(fs.Args()) != 0 {
		return options{}, errors.New("usage: go run ./cmd/validate-apple-music-official [-url <apple-music-album-url>] [-sample-url-file <path>] [-out-dir <dir>] [-storefront <code>]")
	}
	return opts, nil
}

func loadSampleURL(rawURL string, path string) (string, error) {
	if strings.TrimSpace(rawURL) != "" {
		return strings.TrimSpace(rawURL), nil
	}
	content, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return "", fmt.Errorf("read apple music sample url file: %w", err)
	}
	value := strings.TrimSpace(string(content))
	if value == "" {
		return "", fmt.Errorf("apple music sample url file %s is empty", path)
	}
	return value, nil
}

func getAPI(ctx context.Context, endpoint string, developerToken string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build apple music api request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+developerToken)
	req.Header.Set("User-Agent", "ariadne/0.1 (+https://github.com/xmbshwll/ariadne)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute apple music api request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read apple music api response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected apple music api status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

func firstResource(payload map[string]any) map[string]any {
	data, _ := payload["data"].([]any)
	if len(data) == 0 {
		return nil
	}
	resource, _ := data[0].(map[string]any)
	return resource
}

func nestedMap(root map[string]any, keys ...string) map[string]any {
	var current any = root
	for _, key := range keys {
		next, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = next[key]
	}
	mapped, _ := current.(map[string]any)
	return mapped
}

func albumISRCs(albumData map[string]any) []string {
	tracks := nestedMap(albumData, "relationships", "tracks")
	data, _ := tracks["data"].([]any)
	seen := map[string]struct{}{}
	isrcs := make([]string, 0, len(data))
	for _, item := range data {
		track, ok := item.(map[string]any)
		if !ok {
			continue
		}
		attributes := nestedMap(track, "attributes")
		isrc := strings.TrimSpace(asString(attributes["isrc"]))
		if isrc == "" {
			continue
		}
		if _, exists := seen[isrc]; exists {
			continue
		}
		seen[isrc] = struct{}{}
		isrcs = append(isrcs, isrc)
		if len(isrcs) >= 5 {
			break
		}
	}
	return isrcs
}

func nonEmptyStrings(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
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
