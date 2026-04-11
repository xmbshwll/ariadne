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
	"strconv"
	"strings"
	"time"

	"github.com/xmbshwll/ariadne/cmd/internal/validation"
	"github.com/xmbshwll/ariadne/internal/config"
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/parse"
)

const (
	defaultSpotifyAPIBaseURL  = "https://api.spotify.com/v1"
	defaultSpotifyAuthBaseURL = "https://accounts.spotify.com/api"
	searchLimit               = 5
)

var (
	errSpotifyCredentialsRequired = errors.New("SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET must be set")
	errSpotifyUPCMissing          = errors.New("album payload did not include external_ids.upc")
	errSpotifyISRCMissing         = errors.New("spotify track detail payloads did not include any external_ids.isrc values")
	errSpotifyMetadataMissing     = errors.New("album payload did not provide enough metadata for search validation")

	errSpotifyValidateUsage     = errors.New("usage: go run ./cmd/validate-spotify-auth [-url <spotify-album-url>] [-sample-url-file <path>] [-out-dir <dir>]")
	errSpotifySampleURLEmpty    = errors.New("spotify sample url file is empty")
	errSpotifySampleURLRequired = errors.New("provide either -url or -sample-url-file")

	errSpotifyTokenStatus  = errors.New("unexpected spotify token status")
	errSpotifyTokenMissing = errors.New("spotify token response did not include access_token")
	errSpotifyAPIStatus    = errors.New("unexpected spotify api status")
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
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

	fmt.Printf("wrote Spotify authenticated artifacts to %s\n", inputs.outputDir)
	return nil
}

func loadValidationInputs(args []string) (validationInputs, error) {
	opts, err := parseFlags(args)
	if err != nil {
		return validationInputs{}, err
	}

	appConfig := config.Load()
	if !appConfig.Spotify.Enabled() {
		return validationInputs{}, errSpotifyCredentialsRequired
	}

	rawURL, err := validation.LoadSampleURL(opts.sampleURL, opts.sampleURLPath, "spotify", errSpotifySampleURLRequired, errSpotifySampleURLEmpty)
	if err != nil {
		return validationInputs{}, fmt.Errorf("load spotify sample url: %w", err)
	}
	outputDir, err := validation.ResolveOutputDir(opts.outputDir, "ariadne-spotify-validation-")
	if err != nil {
		return validationInputs{}, fmt.Errorf("resolve spotify output dir: %w", err)
	}
	parsed, err := parse.SpotifyAlbumURL(rawURL)
	if err != nil {
		return validationInputs{}, fmt.Errorf("parse sample spotify album url: %w", err)
	}

	return validationInputs{
		opts:      opts,
		appConfig: appConfig,
		rawURL:    rawURL,
		outputDir: outputDir,
		parsed:    parsed,
	}, nil
}

func collectValidationArtifacts(ctx context.Context, inputs validationInputs) (validationArtifacts, error) {
	token, err := fetchToken(ctx, inputs.opts.authBaseURL, inputs.appConfig.Spotify.ClientID, inputs.appConfig.Spotify.ClientSecret)
	if err != nil {
		return validationArtifacts{}, err
	}

	albumBody, album, err := fetchSpotifyAlbum(ctx, inputs.opts.apiBaseURL, inputs.parsed.ID, token)
	if err != nil {
		return validationArtifacts{}, err
	}

	upc, isrcs, metadata, err := validateSpotifyAlbumMetadata(ctx, inputs.opts.apiBaseURL, token, album)
	if err != nil {
		return validationArtifacts{}, err
	}

	upcBody, err := getAPI(ctx, inputs.opts.apiBaseURL+"/search?q="+url.QueryEscape("upc:"+upc)+"&type=album&limit="+strconv.Itoa(searchLimit), token)
	if err != nil {
		return validationArtifacts{}, fmt.Errorf("search spotify by upc: %w", err)
	}
	isrcBody, err := getAPI(ctx, inputs.opts.apiBaseURL+"/search?q="+url.QueryEscape("isrc:"+isrcs[0])+"&type=track&limit="+strconv.Itoa(searchLimit), token)
	if err != nil {
		return validationArtifacts{}, fmt.Errorf("search spotify by isrc: %w", err)
	}
	metadataBody, err := getAPI(ctx, inputs.opts.apiBaseURL+"/search?q="+url.QueryEscape(metadata)+"&type=album&limit="+strconv.Itoa(searchLimit), token)
	if err != nil {
		return validationArtifacts{}, fmt.Errorf("search spotify by metadata: %w", err)
	}

	return validationArtifacts{
		albumBody:    albumBody,
		upcBody:      upcBody,
		isrcBody:     isrcBody,
		metadataBody: metadataBody,
		summary:      buildValidationSummary(inputs, album, upc, isrcs),
	}, nil
}

func fetchSpotifyAlbum(ctx context.Context, apiBaseURL, albumID, token string) ([]byte, spotifyAlbumPayload, error) {
	albumBody, err := getAPI(ctx, apiBaseURL+"/albums/"+albumID, token)
	if err != nil {
		return nil, spotifyAlbumPayload{}, fmt.Errorf("fetch spotify album payload: %w", err)
	}

	var album spotifyAlbumPayload
	if err := json.Unmarshal(albumBody, &album); err != nil {
		return nil, spotifyAlbumPayload{}, fmt.Errorf("decode album payload: %w", err)
	}
	return albumBody, album, nil
}

func validateSpotifyAlbumMetadata(ctx context.Context, apiBaseURL, token string, album spotifyAlbumPayload) (string, []string, string, error) {
	upc := strings.TrimSpace(album.ExternalIDs.UPC)
	if upc == "" {
		return "", nil, "", errSpotifyUPCMissing
	}

	isrcs, err := collectTrackISRCs(ctx, apiBaseURL, token, album)
	if err != nil {
		return "", nil, "", fmt.Errorf("collect spotify track isrcs: %w", err)
	}
	if len(isrcs) == 0 {
		return "", nil, "", errSpotifyISRCMissing
	}

	metadata := metadataQuery(album)
	if metadata == "" {
		return "", nil, "", errSpotifyMetadataMissing
	}
	return upc, isrcs, metadata, nil
}

func buildValidationSummary(inputs validationInputs, album spotifyAlbumPayload, upc string, isrcs []string) map[string]any {
	return map[string]any{
		"sample_url":         inputs.rawURL,
		"album_id":           inputs.parsed.ID,
		"canonical_url":      inputs.parsed.CanonicalURL,
		"title":              strings.TrimSpace(album.Name),
		"artists":            albumArtists(album),
		"release_date":       strings.TrimSpace(album.ReleaseDate),
		"label":              strings.TrimSpace(album.Label),
		"upc":                upc,
		"track_isrc_samples": isrcs,
		"generated_at":       time.Now().UTC().Format(time.RFC3339),
		"artifacts": map[string]string{
			"source_payload_api":    filepath.ToSlash(filepath.Join(inputs.outputDir, "source-payload-api.json")),
			"search_upc_results":    filepath.ToSlash(filepath.Join(inputs.outputDir, "search-upc-results.json")),
			"search_isrc_results":   filepath.ToSlash(filepath.Join(inputs.outputDir, "search-isrc-results.json")),
			"search_metadata":       filepath.ToSlash(filepath.Join(inputs.outputDir, "search-metadata-results.json")),
			"authenticated_summary": filepath.ToSlash(filepath.Join(inputs.outputDir, "authenticated-summary.json")),
		},
	}
}

func writeValidationArtifacts(outputDir string, artifacts validationArtifacts) error {
	if err := writePrettyJSON(filepath.Join(outputDir, "source-payload-api.json"), artifacts.albumBody); err != nil {
		return err
	}
	if err := writePrettyJSON(filepath.Join(outputDir, "search-upc-results.json"), artifacts.upcBody); err != nil {
		return err
	}
	if err := writePrettyJSON(filepath.Join(outputDir, "search-isrc-results.json"), artifacts.isrcBody); err != nil {
		return err
	}
	if err := writePrettyJSON(filepath.Join(outputDir, "search-metadata-results.json"), artifacts.metadataBody); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(outputDir, "authenticated-summary.json"), artifacts.summary); err != nil {
		return err
	}
	return nil
}

type options struct {
	sampleURL     string
	sampleURLPath string
	outputDir     string
	apiBaseURL    string
	authBaseURL   string
}

type validationInputs struct {
	opts      options
	appConfig config.Config
	rawURL    string
	outputDir string
	parsed    *model.ParsedAlbumURL
}

type validationArtifacts struct {
	albumBody    []byte
	upcBody      []byte
	isrcBody     []byte
	metadataBody []byte
	summary      map[string]any
}

type spotifyAlbumPayload struct {
	Name        string `json:"name"`
	ReleaseDate string `json:"release_date"`
	Label       string `json:"label"`
	ExternalIDs struct {
		UPC string `json:"upc"`
	} `json:"external_ids"`
	Artists []spotifyArtist `json:"artists"`
	Tracks  struct {
		Items []spotifyTrackSummary `json:"items"`
	} `json:"tracks"`
}

type spotifyArtist struct {
	Name string `json:"name"`
}

type spotifyTrackSummary struct {
	ID string `json:"id"`
}

type spotifyTrackPayload struct {
	ExternalIDs struct {
		ISRC string `json:"isrc"`
	} `json:"external_ids"`
}

func parseFlags(args []string) (options, error) {
	opts := options{
		apiBaseURL:  defaultSpotifyAPIBaseURL,
		authBaseURL: defaultSpotifyAuthBaseURL,
	}

	fs := flag.NewFlagSet("validate-spotify-auth", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&opts.sampleURL, "url", "", "spotify album url to validate")
	fs.StringVar(&opts.sampleURLPath, "sample-url-file", opts.sampleURLPath, "path to file containing spotify album url")
	fs.StringVar(&opts.outputDir, "out-dir", opts.outputDir, "directory for validation artifacts")
	fs.StringVar(&opts.apiBaseURL, "api-base-url", opts.apiBaseURL, "spotify api base url")
	fs.StringVar(&opts.authBaseURL, "auth-base-url", opts.authBaseURL, "spotify auth base url")
	if err := fs.Parse(args); err != nil {
		return options{}, errSpotifyValidateUsage
	}
	if len(fs.Args()) != 0 {
		return options{}, errSpotifyValidateUsage
	}
	return opts, nil
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
		return "", fmt.Errorf("%w %d: %s", errSpotifyTokenStatus, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var token struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &token); err != nil {
		return "", fmt.Errorf("decode spotify token response: %w", err)
	}
	if token.AccessToken == "" {
		return "", errSpotifyTokenMissing
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
		return nil, fmt.Errorf("%w %d: %s", errSpotifyAPIStatus, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

func metadataQuery(album spotifyAlbumPayload) string {
	title := strings.TrimSpace(album.Name)
	artists := albumArtists(album)
	if title == "" || len(artists) == 0 {
		return ""
	}
	return fmt.Sprintf("album:%s artist:%s", title, artists[0])
}

func albumArtists(album spotifyAlbumPayload) []string {
	artists := make([]string, 0, len(album.Artists))
	for _, artist := range album.Artists {
		name := strings.TrimSpace(artist.Name)
		if name == "" {
			continue
		}
		artists = append(artists, name)
	}
	return artists
}

func collectTrackISRCs(ctx context.Context, apiBaseURL string, token string, album spotifyAlbumPayload) ([]string, error) {
	if len(album.Tracks.Items) == 0 {
		return nil, nil
	}

	seen := map[string]struct{}{}
	isrcs := make([]string, 0, len(album.Tracks.Items))
	for _, track := range album.Tracks.Items {
		trackID := strings.TrimSpace(track.ID)
		if trackID == "" {
			continue
		}
		body, err := getAPI(ctx, strings.TrimRight(apiBaseURL, "/")+"/tracks/"+trackID, token)
		if err != nil {
			return nil, err
		}
		var payload spotifyTrackPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, fmt.Errorf("decode spotify track details payload: %w", err)
		}
		isrc := strings.TrimSpace(payload.ExternalIDs.ISRC)
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
