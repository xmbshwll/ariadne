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
	"strconv"
	"strings"
	"time"

	"github.com/xmbshwll/ariadne/cmd/internal/validation"
	"github.com/xmbshwll/ariadne/internal/applemusicauth"
	"github.com/xmbshwll/ariadne/internal/config"
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/parse"
)

const (
	defaultAPIBaseURL  = "https://api.music.apple.com/v1"
	defaultSearchLimit = 5
)

var (
	errAppleMusicCredentialsRequired = errors.New("APPLE_MUSIC_KEY_ID, APPLE_MUSIC_TEAM_ID, and APPLE_MUSIC_PRIVATE_KEY_PATH must be set")
	errAppleMusicAlbumPayloadMissing = errors.New("official apple music album payload did not include a data resource")
	errAppleMusicMetadataMissing     = errors.New("official apple music album payload did not provide enough metadata for search validation")

	errAppleMusicValidateUsage     = errors.New("usage: go run ./cmd/validate-apple-music-official [-url <apple-music-album-url>] [-sample-url-file <path>] [-out-dir <dir>] [-storefront <code>]")
	errAppleMusicSampleURLEmpty    = errors.New("apple music sample url file is empty")
	errAppleMusicSampleURLRequired = errors.New("provide either -url or -sample-url-file")
	errAppleMusicAPIStatus         = errors.New("unexpected apple music api status")
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

type validationInputs struct {
	opts           options
	appConfig      config.Config
	developerToken string
	rawURL         string
	outputDir      string
	parsed         *model.ParsedAlbumURL
	storefront     string
}

type validationArtifacts struct {
	albumBody    []byte
	metadataBody []byte
	upcBody      []byte
	isrcBody     []byte
	summary      map[string]any
}

type appleMusicAlbumDocument struct {
	Data []appleMusicAlbumResource `json:"data"`
}

type appleMusicAlbumResource struct {
	Attributes    appleMusicAlbumAttributes    `json:"attributes"`
	Relationships appleMusicAlbumRelationships `json:"relationships"`
}

type appleMusicAlbumAttributes struct {
	Name        string `json:"name"`
	ArtistName  string `json:"artistName"`
	ReleaseDate string `json:"releaseDate"`
	RecordLabel string `json:"recordLabel"`
	UPC         string `json:"upc"`
}

type appleMusicAlbumRelationships struct {
	Tracks struct {
		Data []appleMusicSongResource `json:"data"`
	} `json:"tracks"`
}

type appleMusicSongResource struct {
	Attributes appleMusicSongAttributes `json:"attributes"`
}

type appleMusicSongAttributes struct {
	ISRC string `json:"isrc"`
}

func run(args []string) error {
	inputs, err := loadValidationInputs(args)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	artifacts, err := collectValidationArtifacts(ctx, inputs)
	if err != nil {
		return err
	}
	if err := writeValidationArtifacts(inputs.outputDir, artifacts); err != nil {
		return err
	}

	fmt.Printf("wrote Apple Music official artifacts to %s\n", inputs.outputDir)
	return nil
}

func loadValidationInputs(args []string) (validationInputs, error) {
	opts, err := parseFlags(args)
	if err != nil {
		return validationInputs{}, err
	}

	appConfig := config.Load()
	if !appConfig.AppleMusic.AuthEnabled() {
		return validationInputs{}, errAppleMusicCredentialsRequired
	}
	developerToken, err := applemusicauth.GenerateDeveloperToken(applemusicauth.Config{
		KeyID:          appConfig.AppleMusic.KeyID,
		TeamID:         appConfig.AppleMusic.TeamID,
		PrivateKeyPath: appConfig.AppleMusic.PrivateKeyPath,
	}, time.Now().UTC())
	if err != nil {
		return validationInputs{}, fmt.Errorf("generate apple music developer token: %w", err)
	}

	rawURL, err := validation.LoadSampleURL(opts.sampleURL, opts.sampleURLPath, "apple music", errAppleMusicSampleURLRequired, errAppleMusicSampleURLEmpty)
	if err != nil {
		return validationInputs{}, fmt.Errorf("load apple music sample url: %w", err)
	}
	outputDir, err := validation.ResolveOutputDir(opts.outputDir, "ariadne-apple-music-validation-")
	if err != nil {
		return validationInputs{}, fmt.Errorf("resolve apple music output dir: %w", err)
	}
	parsed, err := parse.AppleMusicAlbumURL(rawURL)
	if err != nil {
		return validationInputs{}, fmt.Errorf("parse sample apple music album url: %w", err)
	}

	return validationInputs{
		opts:           opts,
		appConfig:      appConfig,
		developerToken: developerToken,
		rawURL:         rawURL,
		outputDir:      outputDir,
		parsed:         parsed,
		storefront:     resolveStorefront(opts.storefront, parsed.RegionHint, appConfig.AppleMusic.Storefront),
	}, nil
}

func collectValidationArtifacts(ctx context.Context, inputs validationInputs) (validationArtifacts, error) {
	albumBody, album, err := fetchAppleMusicAlbum(ctx, inputs)
	if err != nil {
		return validationArtifacts{}, err
	}

	title := strings.TrimSpace(album.Attributes.Name)
	artist := strings.TrimSpace(album.Attributes.ArtistName)
	releaseDate := strings.TrimSpace(album.Attributes.ReleaseDate)
	label := strings.TrimSpace(album.Attributes.RecordLabel)
	upc := strings.TrimSpace(album.Attributes.UPC)
	isrcs := albumISRCs(album)
	metadataQuery := strings.TrimSpace(strings.Join([]string{title, artist}, " "))
	if metadataQuery == "" {
		return validationArtifacts{}, errAppleMusicMetadataMissing
	}

	metadataBody, err := getAPI(ctx, inputs.opts.apiBaseURL+"/catalog/"+inputs.storefront+"/search?types=albums&limit="+strconv.Itoa(defaultSearchLimit)+"&term="+url.QueryEscape(metadataQuery), inputs.developerToken)
	if err != nil {
		return validationArtifacts{}, fmt.Errorf("search official apple music metadata: %w", err)
	}

	upcBody, err := fetchAppleMusicUPCSearch(ctx, inputs, upc)
	if err != nil {
		return validationArtifacts{}, err
	}
	isrcBody, err := fetchAppleMusicISRCSearch(ctx, inputs, isrcs)
	if err != nil {
		return validationArtifacts{}, err
	}

	return validationArtifacts{
		albumBody:    albumBody,
		metadataBody: metadataBody,
		upcBody:      upcBody,
		isrcBody:     isrcBody,
		summary:      buildValidationSummary(inputs, title, artist, releaseDate, label, upc, isrcs),
	}, nil
}

func resolveStorefront(flagValue, parsedRegion, configuredStorefront string) string {
	for _, storefront := range []string{flagValue, parsedRegion, configuredStorefront, "us"} {
		storefront = strings.ToLower(strings.TrimSpace(storefront))
		if storefront != "" {
			return storefront
		}
	}
	return "us"
}

func fetchAppleMusicAlbum(ctx context.Context, inputs validationInputs) ([]byte, appleMusicAlbumResource, error) {
	albumBody, err := getAPI(ctx, inputs.opts.apiBaseURL+"/catalog/"+inputs.storefront+"/albums/"+inputs.parsed.ID+"?include=tracks", inputs.developerToken)
	if err != nil {
		return nil, appleMusicAlbumResource{}, fmt.Errorf("fetch official apple music album payload: %w", err)
	}

	var albumPayload appleMusicAlbumDocument
	if err := json.Unmarshal(albumBody, &albumPayload); err != nil {
		return nil, appleMusicAlbumResource{}, fmt.Errorf("decode official apple music album payload: %w", err)
	}
	if len(albumPayload.Data) == 0 {
		return nil, appleMusicAlbumResource{}, errAppleMusicAlbumPayloadMissing
	}
	return albumBody, albumPayload.Data[0], nil
}

func fetchAppleMusicUPCSearch(ctx context.Context, inputs validationInputs, upc string) ([]byte, error) {
	if upc == "" {
		return nil, nil
	}
	upcBody, err := getAPI(ctx, inputs.opts.apiBaseURL+"/catalog/"+inputs.storefront+"/albums?filter[upc]="+url.QueryEscape(upc), inputs.developerToken)
	if err != nil {
		return nil, fmt.Errorf("search official apple music by upc: %w", err)
	}
	return upcBody, nil
}

func fetchAppleMusicISRCSearch(ctx context.Context, inputs validationInputs, isrcs []string) ([]byte, error) {
	if len(isrcs) == 0 {
		return nil, nil
	}
	isrcBody, err := getAPI(ctx, inputs.opts.apiBaseURL+"/catalog/"+inputs.storefront+"/songs?filter[isrc]="+url.QueryEscape(isrcs[0]), inputs.developerToken)
	if err != nil {
		return nil, fmt.Errorf("search official apple music by isrc: %w", err)
	}
	return isrcBody, nil
}

func buildValidationSummary(inputs validationInputs, title, artist, releaseDate, label, upc string, isrcs []string) map[string]any {
	artifacts := map[string]string{
		"source_payload_official":  filepath.ToSlash(filepath.Join(inputs.outputDir, "source-payload-official.json")),
		"search_metadata_official": filepath.ToSlash(filepath.Join(inputs.outputDir, "search-metadata-official.json")),
		"official_summary":         filepath.ToSlash(filepath.Join(inputs.outputDir, "official-summary.json")),
	}
	if upc != "" {
		artifacts["search_upc_official"] = filepath.ToSlash(filepath.Join(inputs.outputDir, "search-upc-official.json"))
	}
	if len(isrcs) > 0 {
		artifacts["search_isrc_official"] = filepath.ToSlash(filepath.Join(inputs.outputDir, "search-isrc-official.json"))
	}

	return map[string]any{
		"sample_url":         inputs.rawURL,
		"album_id":           inputs.parsed.ID,
		"canonical_url":      inputs.parsed.CanonicalURL,
		"storefront":         inputs.storefront,
		"auth_mode":          "generated_p8_token",
		"title":              title,
		"artists":            nonEmptyStrings(artist),
		"release_date":       releaseDate,
		"label":              label,
		"upc":                upc,
		"track_isrc_samples": isrcs,
		"generated_at":       time.Now().UTC().Format(time.RFC3339),
		"artifacts":          artifacts,
	}
}

func writeValidationArtifacts(outputDir string, artifacts validationArtifacts) error {
	if err := writePrettyJSON(filepath.Join(outputDir, "source-payload-official.json"), artifacts.albumBody); err != nil {
		return err
	}
	if err := writePrettyJSON(filepath.Join(outputDir, "search-metadata-official.json"), artifacts.metadataBody); err != nil {
		return err
	}
	if len(artifacts.upcBody) > 0 {
		if err := writePrettyJSON(filepath.Join(outputDir, "search-upc-official.json"), artifacts.upcBody); err != nil {
			return err
		}
	}
	if len(artifacts.isrcBody) > 0 {
		if err := writePrettyJSON(filepath.Join(outputDir, "search-isrc-official.json"), artifacts.isrcBody); err != nil {
			return err
		}
	}
	if err := writeJSON(filepath.Join(outputDir, "official-summary.json"), artifacts.summary); err != nil {
		return err
	}
	return nil
}

func parseFlags(args []string) (options, error) {
	opts := options{
		apiBaseURL: defaultAPIBaseURL,
	}

	fs := flag.NewFlagSet("validate-apple-music-official", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&opts.sampleURL, "url", "", "apple music album url to validate")
	fs.StringVar(&opts.sampleURLPath, "sample-url-file", opts.sampleURLPath, "path to file containing apple music album url")
	fs.StringVar(&opts.outputDir, "out-dir", opts.outputDir, "directory for validation artifacts")
	fs.StringVar(&opts.apiBaseURL, "api-base-url", opts.apiBaseURL, "apple music api base url")
	fs.StringVar(&opts.storefront, "storefront", "", "apple music storefront override")
	if err := fs.Parse(args); err != nil {
		return options{}, errAppleMusicValidateUsage
	}
	if len(fs.Args()) != 0 {
		return options{}, errAppleMusicValidateUsage
	}
	return opts, nil
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
		return nil, fmt.Errorf("%w %d: %s", errAppleMusicAPIStatus, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

func albumISRCs(album appleMusicAlbumResource) []string {
	seen := map[string]struct{}{}
	isrcs := make([]string, 0, len(album.Relationships.Tracks.Data))
	for _, track := range album.Relationships.Tracks.Data {
		isrc := strings.TrimSpace(track.Attributes.ISRC)
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
