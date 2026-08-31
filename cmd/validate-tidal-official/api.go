package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/xmbshwll/ariadne/cmd/internal/validation"
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
	albumURL := validation.JoinURL(inputs.opts.apiBaseURL, "albums", inputs.parsed.ID) + "?countryCode=" + url.QueryEscape(inputs.countryCode) + "&include=" + url.QueryEscape("artists,items,coverArt")
	albumBody, err := getAPI(ctx, client, albumURL, accessToken)
	if err != nil {
		return nil, tidalAlbumDocument{}, fmt.Errorf("fetch tidal album payload: %w", err)
	}

	var album tidalAlbumDocument
	if err := validation.DecodeJSONInto(albumBody, &album, "decode tidal album payload"); err != nil {
		return nil, tidalAlbumDocument{}, err
	}
	if strings.TrimSpace(album.Data.ID) == "" {
		return nil, tidalAlbumDocument{}, errTIDALAlbumPayloadMissing
	}
	return albumBody, album, nil
}

func buildTIDALQuery(title string, artistNames []string, albumID string) string {
	query := strings.TrimSpace(strings.Join([]string{title, firstNonEmpty(artistNames...)}, " "))
	if query != "" {
		return query
	}
	return albumID
}

func fetchTIDALAlbumSearch(ctx context.Context, client *http.Client, inputs validationInputs, accessToken, query string) ([]byte, error) {
	searchURL := validation.JoinURL(inputs.opts.apiBaseURL, "searchResults") + "?countryCode=" + url.QueryEscape(inputs.countryCode) + "&filter[query]=" + url.QueryEscape(query) + "&include=albums"
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
	upcSearchURL := validation.JoinURL(inputs.opts.apiBaseURL, "albums") + "?countryCode=" + url.QueryEscape(inputs.countryCode) + "&filter[barcodeId]=" + url.QueryEscape(upc)
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
	isrcSearchURL := validation.JoinURL(inputs.opts.apiBaseURL, "tracks") + "?countryCode=" + url.QueryEscape(inputs.countryCode) + "&filter[isrc]=" + url.QueryEscape(trackISRCs[0])
	isrcSearchBody, err := getAPI(ctx, client, isrcSearchURL, accessToken)
	if err != nil {
		return fmt.Errorf("search tidal tracks by isrc: %w", err)
	}
	targets["search-isrc-official.json"] = isrcSearchBody
	return nil
}

func fetchAccessToken(ctx context.Context, client *http.Client, authBaseURL string, clientID string, clientSecret string) (string, error) {
	return validation.FetchClientCredentialsToken(ctx, validation.TokenRequest{
		Client:       client,
		Endpoint:     validation.JoinURL(authBaseURL, "oauth2/token"),
		ClientID:     clientID,
		ClientSecret: clientSecret,
		ContentType:  "application/x-www-form-urlencoded; charset=UTF-8",
		BuildError:   "build tidal token request",
		ExecuteError: "execute tidal token request",
		StatusError:  errTIDALTokenStatus,
		DecodeError:  "decode tidal token response",
		MissingError: errTIDALTokenMissing,
	})
}

func getAPI(ctx context.Context, client *http.Client, endpoint string, accessToken string) ([]byte, error) {
	return validation.AuthenticatedGet(ctx, validation.GetRequest{
		Client:       client,
		URL:          endpoint,
		Token:        accessToken,
		Headers:      map[string]string{"Accept": "application/vnd.api+json"},
		BuildError:   "build tidal api request",
		ExecuteError: "execute tidal api request",
		StatusError:  errTIDALAPIStatus,
		ReadError:    "read tidal api response",
	})
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
