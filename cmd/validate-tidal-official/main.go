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

	"github.com/xmbshwll/ariadne/cmd/internal/validation"
	"github.com/xmbshwll/ariadne/internal/config"
	"github.com/xmbshwll/ariadne/internal/parse"
)

const (
	defaultTIDALAPIBaseURL  = "https://openapi.tidal.com/v2"
	defaultTIDALAuthBaseURL = "https://auth.tidal.com/v1"
	defaultCountryCode      = "US"
	defaultSearchLimit      = 5
)

var (
	errTIDALCredentialsRequired = errors.New("TIDAL_CLIENT_ID and TIDAL_CLIENT_SECRET must be set")
	errTIDALAlbumPayloadMissing = errors.New("tidal album payload did not include a data resource")

	errTIDALValidateUsage     = errors.New("usage: go run ./cmd/validate-tidal-official [-url <tidal-album-url>] [-sample-url-file <path>] [-out-dir <dir>] [-country-code <cc>]")
	errTIDALSampleURLEmpty    = errors.New("tidal sample url file is empty")
	errTIDALSampleURLRequired = errors.New("provide either -url or -sample-url-file")

	errTIDALTokenStatus  = errors.New("unexpected tidal token status")
	errTIDALTokenMissing = errors.New("tidal token response did not include access_token")
	errTIDALAPIStatus    = errors.New("unexpected tidal api status")
)

type options struct {
	sampleURL     string
	sampleURLPath string
	outputDir     string
	apiBaseURL    string
	authBaseURL   string
	countryCode   string
}

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
	if !appConfig.TIDAL.Enabled() {
		return errTIDALCredentialsRequired
	}

	rawURL, err := validation.LoadSampleURL(opts.sampleURL, opts.sampleURLPath, "tidal", errTIDALSampleURLRequired, errTIDALSampleURLEmpty)
	if err != nil {
		return fmt.Errorf("load tidal sample url: %w", err)
	}
	outputDir, err := validation.ResolveOutputDir(opts.outputDir, "ariadne-tidal-validation-")
	if err != nil {
		return fmt.Errorf("resolve tidal output dir: %w", err)
	}
	parsed, err := parse.TIDALAlbumURL(rawURL)
	if err != nil {
		return fmt.Errorf("parse sample tidal album url: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	accessToken, err := fetchAccessToken(ctx, opts.authBaseURL, appConfig.TIDAL.ClientID, appConfig.TIDAL.ClientSecret)
	if err != nil {
		return fmt.Errorf("fetch tidal access token: %w", err)
	}

	albumURL := fmt.Sprintf("%s/albums/%s?countryCode=%s&include=%s", strings.TrimRight(opts.apiBaseURL, "/"), url.PathEscape(parsed.ID), url.QueryEscape(opts.countryCode), url.QueryEscape("artists,items,coverArt"))
	albumBody, err := getAPI(ctx, albumURL, accessToken)
	if err != nil {
		return fmt.Errorf("fetch tidal album payload: %w", err)
	}

	var albumPayload map[string]any
	if err := json.Unmarshal(albumBody, &albumPayload); err != nil {
		return fmt.Errorf("decode tidal album payload: %w", err)
	}

	data := firstObjectFromDocument(albumPayload)
	if data == nil {
		return errTIDALAlbumPayloadMissing
	}
	attributes := nestedMap(data, "attributes")
	title := firstNonEmpty(
		stringValue(attributes, "title"),
		stringValue(attributes, "name"),
	)
	artistNames := collectIncludedNames(albumPayload, "artists")
	if len(artistNames) == 0 {
		artistNames = collectRelationshipNames(data, albumPayload, "artists")
	}
	trackTitles := collectIncludedTitles(albumPayload, "tracks", defaultSearchLimit)
	trackISRCs := collectIncludedISRCs(albumPayload, defaultSearchLimit)
	upc := firstNonEmpty(stringValue(attributes, "barcodeId"), stringValue(attributes, "upc"))
	releaseDate := stringValue(attributes, "releaseDate")
	query := strings.TrimSpace(strings.Join([]string{title, firstArtist(artistNames)}, " "))
	if query == "" {
		query = parsed.ID
	}

	searchURL := fmt.Sprintf("%s/searchResults/%s/relationships/albums?countryCode=%s", strings.TrimRight(opts.apiBaseURL, "/"), url.PathEscape(query), url.QueryEscape(opts.countryCode))
	searchBody, err := getAPI(ctx, searchURL, accessToken)
	if err != nil {
		return fmt.Errorf("search tidal albums: %w", err)
	}

	targets := map[string][]byte{
		"source-payload-official.json": albumBody,
		"search-albums-official.json":  searchBody,
	}
	if upc != "" {
		upcSearchURL := fmt.Sprintf("%s/albums?countryCode=%s&filter[barcodeId]=%s", strings.TrimRight(opts.apiBaseURL, "/"), url.QueryEscape(opts.countryCode), url.QueryEscape(upc))
		upcSearchBody, err := getAPI(ctx, upcSearchURL, accessToken)
		if err != nil {
			return fmt.Errorf("search tidal albums by upc: %w", err)
		}
		targets["search-upc-official.json"] = upcSearchBody
	}
	if len(trackISRCs) > 0 {
		isrcSearchURL := fmt.Sprintf("%s/tracks?countryCode=%s&filter[isrc]=%s", strings.TrimRight(opts.apiBaseURL, "/"), url.QueryEscape(opts.countryCode), url.QueryEscape(trackISRCs[0]))
		isrcSearchBody, err := getAPI(ctx, isrcSearchURL, accessToken)
		if err != nil {
			return fmt.Errorf("search tidal tracks by isrc: %w", err)
		}
		targets["search-isrc-official.json"] = isrcSearchBody
	}

	summary := map[string]any{
		"sample_url":          rawURL,
		"album_id":            parsed.ID,
		"canonical_url":       parsed.CanonicalURL,
		"country_code":        strings.ToUpper(strings.TrimSpace(opts.countryCode)),
		"title":               title,
		"artists":             artistNames,
		"release_date":        releaseDate,
		"upc":                 upc,
		"track_title_samples": trackTitles,
		"track_isrc_samples":  trackISRCs,
		"generated_at":        time.Now().UTC().Format(time.RFC3339),
		"token_acquired":      true,
		"artifacts": map[string]string{
			"source_payload_official": filepath.ToSlash(filepath.Join(outputDir, "source-payload-official.json")),
			"search_albums_official":  filepath.ToSlash(filepath.Join(outputDir, "search-albums-official.json")),
			"search_upc_official":     filepath.ToSlash(filepath.Join(outputDir, "search-upc-official.json")),
			"search_isrc_official":    filepath.ToSlash(filepath.Join(outputDir, "search-isrc-official.json")),
			"official_summary":        filepath.ToSlash(filepath.Join(outputDir, "official-summary.json")),
		},
	}

	for name, raw := range targets {
		if err := writePrettyJSON(filepath.Join(outputDir, name), raw); err != nil {
			return err
		}
	}
	if err := writeJSON(filepath.Join(outputDir, "official-summary.json"), summary); err != nil {
		return err
	}

	fmt.Printf("wrote TIDAL official artifacts to %s\n", outputDir)
	return nil
}

func parseFlags(args []string) (options, error) {
	opts := options{
		apiBaseURL:  defaultTIDALAPIBaseURL,
		authBaseURL: defaultTIDALAuthBaseURL,
		countryCode: defaultCountryCode,
	}

	fs := flag.NewFlagSet("validate-tidal-official", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&opts.sampleURL, "url", "", "tidal album url to validate")
	fs.StringVar(&opts.sampleURLPath, "sample-url-file", opts.sampleURLPath, "path to file containing tidal album url")
	fs.StringVar(&opts.outputDir, "out-dir", opts.outputDir, "directory for validation artifacts")
	fs.StringVar(&opts.apiBaseURL, "api-base-url", opts.apiBaseURL, "tidal api base url")
	fs.StringVar(&opts.authBaseURL, "auth-base-url", opts.authBaseURL, "tidal auth base url")
	fs.StringVar(&opts.countryCode, "country-code", opts.countryCode, "tidal country code")
	if err := fs.Parse(args); err != nil {
		return options{}, errTIDALValidateUsage
	}
	if len(fs.Args()) != 0 {
		return options{}, errTIDALValidateUsage
	}
	return opts, nil
}

func fetchAccessToken(ctx context.Context, authBaseURL string, clientID string, clientSecret string) (string, error) {
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("grant_type", "client_credentials")

	endpoint := strings.TrimRight(authBaseURL, "/") + "/oauth2/token"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("build tidal token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("User-Agent", "ariadne/0.1 (+https://github.com/xmbshwll/ariadne)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute tidal token request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read tidal token response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w %d: %s", errTIDALTokenStatus, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("decode tidal token response: %w", err)
	}
	if strings.TrimSpace(payload.AccessToken) == "" {
		return "", errTIDALTokenMissing
	}
	return payload.AccessToken, nil
}

func getAPI(ctx context.Context, endpoint string, accessToken string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build tidal api request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.api+json")
	req.Header.Set("User-Agent", "ariadne/0.1 (+https://github.com/xmbshwll/ariadne)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute tidal api request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read tidal api response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w %d: %s", errTIDALAPIStatus, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

func firstObjectFromDocument(payload map[string]any) map[string]any {
	data, ok := payload["data"]
	if !ok {
		return nil
	}
	if single, ok := data.(map[string]any); ok {
		return single
	}
	list, _ := data.([]any)
	if len(list) == 0 {
		return nil
	}
	first, _ := list[0].(map[string]any)
	return first
}

func nestedMap(root map[string]any, key string) map[string]any {
	if root == nil {
		return nil
	}
	mapped, _ := root[key].(map[string]any)
	return mapped
}

func stringValue(root map[string]any, key string) string {
	if root == nil {
		return ""
	}
	text, _ := root[key].(string)
	return strings.TrimSpace(text)
}

func collectIncludedNames(payload map[string]any, typ string) []string {
	included, _ := payload["included"].([]any)
	results := make([]string, 0, len(included))
	seen := map[string]struct{}{}
	for _, item := range included {
		resource, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if stringValue(resource, "type") != typ {
			continue
		}
		name := firstNonEmpty(stringValue(nestedMap(resource, "attributes"), "name"), stringValue(nestedMap(resource, "attributes"), "title"))
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		results = append(results, name)
	}
	return results
}

func collectRelationshipNames(resource map[string]any, payload map[string]any, relationship string) []string {
	rel := nestedMap(nestedMap(resource, "relationships"), relationship)
	data, _ := rel["data"].([]any)
	if len(data) == 0 {
		return nil
	}
	idToName := make(map[string]string)
	included, _ := payload["included"].([]any)
	for _, item := range included {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id := stringValue(entry, "id")
		if id == "" {
			continue
		}
		name := firstNonEmpty(stringValue(nestedMap(entry, "attributes"), "name"), stringValue(nestedMap(entry, "attributes"), "title"))
		if name == "" {
			continue
		}
		idToName[id] = name
	}
	results := make([]string, 0, len(data))
	for _, item := range data {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if name := idToName[stringValue(entry, "id")]; name != "" {
			results = append(results, name)
		}
	}
	return results
}

func collectIncludedTitles(payload map[string]any, typ string, limit int) []string {
	included, _ := payload["included"].([]any)
	results := make([]string, 0, limit)
	seen := map[string]struct{}{}
	for _, item := range included {
		resource, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if stringValue(resource, "type") != typ {
			continue
		}
		title := firstNonEmpty(stringValue(nestedMap(resource, "attributes"), "title"), stringValue(nestedMap(resource, "attributes"), "name"))
		if title == "" {
			continue
		}
		if _, ok := seen[title]; ok {
			continue
		}
		seen[title] = struct{}{}
		results = append(results, title)
		if len(results) >= limit {
			break
		}
	}
	return results
}

func collectIncludedISRCs(payload map[string]any, limit int) []string {
	included, _ := payload["included"].([]any)
	results := make([]string, 0, limit)
	seen := map[string]struct{}{}
	for _, item := range included {
		resource, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if stringValue(resource, "type") != "tracks" {
			continue
		}
		isrc := stringValue(nestedMap(resource, "attributes"), "isrc")
		if isrc == "" {
			continue
		}
		if _, ok := seen[isrc]; ok {
			continue
		}
		seen[isrc] = struct{}{}
		results = append(results, isrc)
		if len(results) >= limit {
			break
		}
	}
	return results
}

func firstArtist(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
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
