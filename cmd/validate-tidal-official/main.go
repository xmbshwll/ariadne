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
	"github.com/xmbshwll/ariadne/internal/model"
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

type validationInputs struct {
	opts        options
	appConfig   config.Config
	rawURL      string
	outputDir   string
	parsed      *model.ParsedAlbumURL
	countryCode string
}

func (i validationInputs) OutputDir() string {
	return i.outputDir
}

func (i validationInputs) SuccessMessage() string {
	return "wrote TIDAL official artifacts to " + i.outputDir
}

type validationArtifacts struct {
	targets map[string][]byte
	summary map[string]any
}

type tidalAlbumDocument struct {
	Data     tidalResource           `json:"data"`
	Included []tidalIncludedResource `json:"included"`
}

type tidalResource struct {
	ID            string             `json:"id"`
	Attributes    tidalAttributes    `json:"attributes"`
	Relationships tidalRelationships `json:"relationships"`
}

type tidalIncludedResource struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Attributes tidalAttributes `json:"attributes"`
}

type tidalAttributes struct {
	Title       string `json:"title"`
	Name        string `json:"name"`
	BarcodeID   string `json:"barcodeId"`
	UPC         string `json:"upc"`
	ReleaseDate string `json:"releaseDate"`
	ISRC        string `json:"isrc"`
}

type tidalRelationships struct {
	Artists tidalRelationship `json:"artists"`
}

type tidalRelationship struct {
	Data []tidalRelationshipData `json:"data"`
}

type tidalRelationshipData struct {
	ID string `json:"id"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if err := validation.Run(args, os.Stdout, 30*time.Second, loadValidationInputs, collectValidationArtifacts, writeValidationArtifacts); err != nil {
		return fmt.Errorf("run tidal validation: %w", err)
	}
	return nil
}

func loadValidationInputs(args []string) (validationInputs, error) {
	opts, err := parseFlags(args)
	if err != nil {
		return validationInputs{}, err
	}

	appConfig := config.Load()
	if !appConfig.TIDAL.Enabled() {
		return validationInputs{}, errTIDALCredentialsRequired
	}

	rawURL, err := validation.LoadSampleURL(opts.sampleURL, opts.sampleURLPath, "tidal", errTIDALSampleURLRequired, errTIDALSampleURLEmpty)
	if err != nil {
		return validationInputs{}, fmt.Errorf("load tidal sample url: %w", err)
	}
	parsed, err := parse.TIDALAlbumURL(rawURL)
	if err != nil {
		return validationInputs{}, fmt.Errorf("parse sample tidal album url: %w", err)
	}
	outputDir, err := validation.ResolveOutputDir(opts.outputDir, "ariadne-tidal-validation-")
	if err != nil {
		return validationInputs{}, fmt.Errorf("resolve tidal output dir: %w", err)
	}

	countryCode := strings.ToUpper(strings.TrimSpace(opts.countryCode))
	if countryCode == "" {
		countryCode = "US"
	}

	return validationInputs{
		opts:        opts,
		appConfig:   appConfig,
		rawURL:      rawURL,
		outputDir:   outputDir,
		parsed:      parsed,
		countryCode: countryCode,
	}, nil
}

func collectValidationArtifacts(ctx context.Context, inputs validationInputs) (validationArtifacts, error) {
	accessToken, err := fetchAccessToken(ctx, inputs.opts.authBaseURL, inputs.appConfig.TIDAL.ClientID, inputs.appConfig.TIDAL.ClientSecret)
	if err != nil {
		return validationArtifacts{}, fmt.Errorf("fetch tidal access token: %w", err)
	}

	albumBody, album, err := fetchTIDALAlbum(ctx, inputs, accessToken)
	if err != nil {
		return validationArtifacts{}, err
	}

	title := firstNonEmpty(album.Data.Attributes.Title, album.Data.Attributes.Name)
	artistNames := collectIncludedNames(album.Included, "artists")
	if len(artistNames) == 0 {
		artistNames = collectRelationshipNames(album.Data.Relationships.Artists.Data, album.Included)
	}
	trackTitles := collectIncludedTitles(album.Included, "tracks", defaultSearchLimit)
	trackISRCs := collectIncludedISRCs(album.Included, defaultSearchLimit)
	upc := firstNonEmpty(album.Data.Attributes.BarcodeID, album.Data.Attributes.UPC)
	releaseDate := strings.TrimSpace(album.Data.Attributes.ReleaseDate)
	query := buildTIDALQuery(title, artistNames, inputs.parsed.ID)

	searchBody, err := fetchTIDALAlbumSearch(ctx, inputs, accessToken, query)
	if err != nil {
		return validationArtifacts{}, err
	}

	targets := map[string][]byte{
		"source-payload-official.json": albumBody,
		"search-albums-official.json":  searchBody,
	}
	if err := addTIDALUPCArtifact(ctx, inputs, accessToken, targets, upc); err != nil {
		return validationArtifacts{}, err
	}
	if err := addTIDALISRCArtifact(ctx, inputs, accessToken, targets, trackISRCs); err != nil {
		return validationArtifacts{}, err
	}

	return validationArtifacts{
		targets: targets,
		summary: buildValidationSummary(inputs, title, artistNames, releaseDate, upc, trackTitles, trackISRCs),
	}, nil
}

func fetchTIDALAlbum(ctx context.Context, inputs validationInputs, accessToken string) ([]byte, tidalAlbumDocument, error) {
	albumURL := fmt.Sprintf("%s/albums/%s?countryCode=%s&include=%s", strings.TrimRight(inputs.opts.apiBaseURL, "/"), url.PathEscape(inputs.parsed.ID), url.QueryEscape(inputs.countryCode), url.QueryEscape("artists,items,coverArt"))
	albumBody, err := getAPI(ctx, albumURL, accessToken)
	if err != nil {
		return nil, tidalAlbumDocument{}, fmt.Errorf("fetch tidal album payload: %w", err)
	}

	var album tidalAlbumDocument
	if err := json.Unmarshal(albumBody, &album); err != nil {
		return nil, tidalAlbumDocument{}, fmt.Errorf("decode tidal album payload: %w", err)
	}
	if strings.TrimSpace(album.Data.ID) == "" {
		return nil, tidalAlbumDocument{}, errTIDALAlbumPayloadMissing
	}
	return albumBody, album, nil
}

func buildTIDALQuery(title string, artistNames []string, albumID string) string {
	query := strings.TrimSpace(strings.Join([]string{title, firstArtist(artistNames)}, " "))
	if query != "" {
		return query
	}
	return albumID
}

func fetchTIDALAlbumSearch(ctx context.Context, inputs validationInputs, accessToken, query string) ([]byte, error) {
	searchURL := fmt.Sprintf("%s/searchResults/%s/relationships/albums?countryCode=%s", strings.TrimRight(inputs.opts.apiBaseURL, "/"), url.PathEscape(query), url.QueryEscape(inputs.countryCode))
	searchBody, err := getAPI(ctx, searchURL, accessToken)
	if err != nil {
		return nil, fmt.Errorf("search tidal albums: %w", err)
	}
	return searchBody, nil
}

func addTIDALUPCArtifact(ctx context.Context, inputs validationInputs, accessToken string, targets map[string][]byte, upc string) error {
	if upc == "" {
		return nil
	}
	upcSearchURL := fmt.Sprintf("%s/albums?countryCode=%s&filter[barcodeId]=%s", strings.TrimRight(inputs.opts.apiBaseURL, "/"), url.QueryEscape(inputs.countryCode), url.QueryEscape(upc))
	upcSearchBody, err := getAPI(ctx, upcSearchURL, accessToken)
	if err != nil {
		return fmt.Errorf("search tidal albums by upc: %w", err)
	}
	targets["search-upc-official.json"] = upcSearchBody
	return nil
}

func addTIDALISRCArtifact(ctx context.Context, inputs validationInputs, accessToken string, targets map[string][]byte, trackISRCs []string) error {
	if len(trackISRCs) == 0 {
		return nil
	}
	isrcSearchURL := fmt.Sprintf("%s/tracks?countryCode=%s&filter[isrc]=%s", strings.TrimRight(inputs.opts.apiBaseURL, "/"), url.QueryEscape(inputs.countryCode), url.QueryEscape(trackISRCs[0]))
	isrcSearchBody, err := getAPI(ctx, isrcSearchURL, accessToken)
	if err != nil {
		return fmt.Errorf("search tidal tracks by isrc: %w", err)
	}
	targets["search-isrc-official.json"] = isrcSearchBody
	return nil
}

func buildValidationSummary(inputs validationInputs, title string, artistNames []string, releaseDate string, upc string, trackTitles []string, trackISRCs []string) map[string]any {
	artifacts := map[string]string{
		"source_payload_official": filepath.ToSlash(filepath.Join(inputs.outputDir, "source-payload-official.json")),
		"search_albums_official":  filepath.ToSlash(filepath.Join(inputs.outputDir, "search-albums-official.json")),
		"official_summary":        filepath.ToSlash(filepath.Join(inputs.outputDir, "official-summary.json")),
	}
	if upc != "" {
		artifacts["search_upc_official"] = filepath.ToSlash(filepath.Join(inputs.outputDir, "search-upc-official.json"))
	}
	if len(trackISRCs) > 0 {
		artifacts["search_isrc_official"] = filepath.ToSlash(filepath.Join(inputs.outputDir, "search-isrc-official.json"))
	}

	return map[string]any{
		"sample_url":          inputs.rawURL,
		"album_id":            inputs.parsed.ID,
		"canonical_url":       inputs.parsed.CanonicalURL,
		"country_code":        inputs.countryCode,
		"title":               title,
		"artists":             artistNames,
		"release_date":        releaseDate,
		"upc":                 upc,
		"track_title_samples": trackTitles,
		"track_isrc_samples":  trackISRCs,
		"generated_at":        time.Now().UTC().Format(time.RFC3339),
		"token_acquired":      true,
		"artifacts":           artifacts,
	}
}

func writeValidationArtifacts(outputDir string, artifacts validationArtifacts) error {
	for name, raw := range artifacts.targets {
		path := filepath.Join(outputDir, name)
		if err := validation.WritePrettyJSON(path, raw); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
	}
	summaryPath := filepath.Join(outputDir, "official-summary.json")
	if err := validation.WriteJSON(summaryPath, artifacts.summary); err != nil {
		return fmt.Errorf("write %s: %w", summaryPath, err)
	}
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

func collectIncludedNames(included []tidalIncludedResource, typ string) []string {
	results := make([]string, 0, len(included))
	seen := map[string]struct{}{}
	for _, resource := range included {
		if resource.Type != typ {
			continue
		}
		name := firstNonEmpty(resource.Attributes.Name, resource.Attributes.Title)
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

func collectRelationshipNames(relations []tidalRelationshipData, included []tidalIncludedResource) []string {
	if len(relations) == 0 {
		return []string{}
	}

	idToName := make(map[string]string, len(included))
	for _, resource := range included {
		resourceID := strings.TrimSpace(resource.ID)
		name := firstNonEmpty(resource.Attributes.Name, resource.Attributes.Title)
		if resourceID == "" || name == "" {
			continue
		}
		idToName[resourceID] = name
	}

	results := make([]string, 0, len(relations))
	for _, relation := range relations {
		if name := idToName[strings.TrimSpace(relation.ID)]; name != "" {
			results = append(results, name)
		}
	}
	return results
}

func collectIncludedTitles(included []tidalIncludedResource, typ string, limit int) []string {
	results := make([]string, 0, limit)
	seen := map[string]struct{}{}
	for _, resource := range included {
		if resource.Type != typ {
			continue
		}
		title := firstNonEmpty(resource.Attributes.Title, resource.Attributes.Name)
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

func collectIncludedISRCs(included []tidalIncludedResource, limit int) []string {
	results := make([]string, 0, limit)
	seen := map[string]struct{}{}
	for _, resource := range included {
		if resource.Type != "tracks" {
			continue
		}
		isrc := strings.TrimSpace(resource.Attributes.ISRC)
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
