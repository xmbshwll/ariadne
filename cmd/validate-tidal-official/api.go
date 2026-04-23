package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	defaultTIDALAPIBaseURL  = "https://openapi.tidal.com/v2"
	defaultTIDALAuthBaseURL = "https://auth.tidal.com/v1"
	defaultCountryCode      = "US"
	defaultSearchLimit      = 5
)

var (
	errTIDALAlbumPayloadMissing = errors.New("tidal album payload did not include a data resource")
	errTIDALTokenStatus         = errors.New("unexpected tidal token status")
	errTIDALTokenMissing        = errors.New("tidal token response did not include access_token")
	errTIDALAPIStatus           = errors.New("unexpected tidal api status")
)

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

func collectValidationArtifacts(ctx context.Context, inputs validationInputs) (validationArtifacts, error) {
	client := &http.Client{Timeout: inputs.appConfig.HTTPTimeout}

	accessToken, err := fetchAccessToken(ctx, client, inputs.opts.authBaseURL, inputs.appConfig.TIDAL.ClientID, inputs.appConfig.TIDAL.ClientSecret)
	if err != nil {
		return validationArtifacts{}, fmt.Errorf("fetch tidal access token: %w", err)
	}

	albumBody, album, err := fetchTIDALAlbum(ctx, client, inputs, accessToken)
	if err != nil {
		return validationArtifacts{}, err
	}

	title := firstNonEmpty(album.Data.Attributes.Title, album.Data.Attributes.Name)
	artistNames := collectIncludedNames(album.Included, "artists")
	if len(artistNames) == 0 {
		artistNames = collectRelationshipNames(album.Data.Relationships.Artists.Data, album.Included)
	}
	trackTitles := collectIncludedTitles(album.Included, "tracks", defaultSearchLimit)
	trackISRCs := collectIncludedValues(album.Included, "tracks", defaultSearchLimit, includedISRC)
	upc := firstNonEmpty(album.Data.Attributes.BarcodeID, album.Data.Attributes.UPC)
	releaseDate := strings.TrimSpace(album.Data.Attributes.ReleaseDate)
	query := buildTIDALQuery(title, artistNames, inputs.parsed.ID)

	searchBody, err := fetchTIDALAlbumSearch(ctx, client, inputs, accessToken, query)
	if err != nil {
		return validationArtifacts{}, err
	}

	targets := map[string][]byte{
		"source-payload-official.json": albumBody,
		"search-albums-official.json":  searchBody,
	}
	if err := addTIDALUPCArtifact(ctx, client, inputs, accessToken, targets, upc); err != nil {
		return validationArtifacts{}, err
	}
	if err := addTIDALISRCArtifact(ctx, client, inputs, accessToken, targets, trackISRCs); err != nil {
		return validationArtifacts{}, err
	}

	return validationArtifacts{
		targets: targets,
		summary: buildValidationSummary(inputs, title, artistNames, releaseDate, upc, trackTitles, trackISRCs),
	}, nil
}

func fetchTIDALAlbum(ctx context.Context, client *http.Client, inputs validationInputs, accessToken string) ([]byte, tidalAlbumDocument, error) {
	albumURL := fmt.Sprintf("%s/albums/%s?countryCode=%s&include=%s", strings.TrimRight(inputs.opts.apiBaseURL, "/"), url.PathEscape(inputs.parsed.ID), url.QueryEscape(inputs.countryCode), url.QueryEscape("artists,items,coverArt"))
	albumBody, err := getAPI(ctx, client, albumURL, accessToken)
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

func fetchTIDALAlbumSearch(ctx context.Context, client *http.Client, inputs validationInputs, accessToken, query string) ([]byte, error) {
	searchURL := fmt.Sprintf("%s/searchResults/%s/relationships/albums?countryCode=%s", strings.TrimRight(inputs.opts.apiBaseURL, "/"), url.PathEscape(query), url.QueryEscape(inputs.countryCode))
	searchBody, err := getAPI(ctx, client, searchURL, accessToken)
	if err != nil {
		return nil, fmt.Errorf("search tidal albums: %w", err)
	}
	return searchBody, nil
}

func addTIDALUPCArtifact(ctx context.Context, client *http.Client, inputs validationInputs, accessToken string, targets map[string][]byte, upc string) error {
	if upc == "" {
		return nil
	}
	upcSearchURL := fmt.Sprintf("%s/albums?countryCode=%s&filter[barcodeId]=%s", strings.TrimRight(inputs.opts.apiBaseURL, "/"), url.QueryEscape(inputs.countryCode), url.QueryEscape(upc))
	upcSearchBody, err := getAPI(ctx, client, upcSearchURL, accessToken)
	if err != nil {
		return fmt.Errorf("search tidal albums by upc: %w", err)
	}
	targets["search-upc-official.json"] = upcSearchBody
	return nil
}

func addTIDALISRCArtifact(ctx context.Context, client *http.Client, inputs validationInputs, accessToken string, targets map[string][]byte, trackISRCs []string) error {
	if len(trackISRCs) == 0 {
		return nil
	}
	isrcSearchURL := fmt.Sprintf("%s/tracks?countryCode=%s&filter[isrc]=%s", strings.TrimRight(inputs.opts.apiBaseURL, "/"), url.QueryEscape(inputs.countryCode), url.QueryEscape(trackISRCs[0]))
	isrcSearchBody, err := getAPI(ctx, client, isrcSearchURL, accessToken)
	if err != nil {
		return fmt.Errorf("search tidal tracks by isrc: %w", err)
	}
	targets["search-isrc-official.json"] = isrcSearchBody
	return nil
}

func fetchAccessToken(ctx context.Context, client *http.Client, authBaseURL string, clientID string, clientSecret string) (string, error) {
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

	resp, err := client.Do(req)
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

func getAPI(ctx context.Context, client *http.Client, endpoint string, accessToken string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build tidal api request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.api+json")
	req.Header.Set("User-Agent", "ariadne/0.1 (+https://github.com/xmbshwll/ariadne)")

	resp, err := client.Do(req)
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
	return collectIncludedValues(included, typ, 0, func(attrs tidalAttributes) string {
		return firstNonEmpty(attrs.Name, attrs.Title)
	})
}

func collectRelationshipNames(relations []tidalRelationshipData, included []tidalIncludedResource) []string {
	if len(relations) == 0 {
		return []string{}
	}

	idToName := make(map[string]string, len(included))
	for _, resource := range included {
		if resource.Type != "artists" {
			continue
		}
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
	return collectIncludedValues(included, typ, limit, func(attrs tidalAttributes) string {
		return firstNonEmpty(attrs.Title, attrs.Name)
	})
}

func collectIncludedValues(included []tidalIncludedResource, typ string, limit int, value func(tidalAttributes) string) []string {
	capacity := len(included)
	if limit > 0 && limit < capacity {
		capacity = limit
	}

	results := make([]string, 0, capacity)
	seen := map[string]struct{}{}
	for _, resource := range included {
		if resource.Type != typ {
			continue
		}
		item := value(resource.Attributes)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		results = append(results, item)
		if limit > 0 && len(results) >= limit {
			break
		}
	}
	return results
}

func includedISRC(attrs tidalAttributes) string {
	return strings.TrimSpace(attrs.ISRC)
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
