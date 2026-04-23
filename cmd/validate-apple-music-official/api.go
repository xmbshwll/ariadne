package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	defaultAPIBaseURL  = "https://api.music.apple.com/v1"
	defaultSearchLimit = 5
)

var (
	errAppleMusicAlbumPayloadMissing = errors.New("official apple music album payload did not include a data resource")
	errAppleMusicMetadataMissing     = errors.New("official apple music album payload did not provide enough metadata for search validation")
	errAppleMusicAPIStatus           = errors.New("unexpected apple music api status")
)

type appleMusicAlbumDocument struct {
	Data []appleMusicAlbumResource `json:"data"`
}

type appleMusicSearchDocument struct {
	Data []json.RawMessage `json:"data"`
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

func collectValidationArtifacts(ctx context.Context, inputs validationInputs) (validationArtifacts, error) {
	client := &http.Client{Timeout: inputs.appConfig.HTTPTimeout}

	albumBody, album, err := fetchAppleMusicAlbum(ctx, client, inputs)
	if err != nil {
		return validationArtifacts{}, err
	}

	attrs := album.Attributes
	title := strings.TrimSpace(attrs.Name)
	artist := strings.TrimSpace(attrs.ArtistName)
	releaseDate := strings.TrimSpace(attrs.ReleaseDate)
	label := strings.TrimSpace(attrs.RecordLabel)
	upc := strings.TrimSpace(attrs.UPC)
	isrcs := albumISRCs(album)
	metadataTerm := strings.TrimSpace(strings.Join([]string{title, artist}, " "))
	if metadataTerm == "" {
		return validationArtifacts{}, errAppleMusicMetadataMissing
	}

	catalogBaseURL := appleMusicCatalogBaseURL(inputs)
	metadataURL := catalogBaseURL + "/search?types=albums&limit=" + strconv.Itoa(defaultSearchLimit) + "&term=" + url.QueryEscape(metadataTerm)
	metadataBody, err := getAPI(ctx, client, metadataURL, inputs.developerToken)
	if err != nil {
		return validationArtifacts{}, fmt.Errorf("search official apple music metadata: %w", err)
	}

	upcBody, err := fetchAppleMusicUPCSearch(ctx, client, inputs, upc)
	if err != nil {
		return validationArtifacts{}, err
	}
	isrcBody, err := fetchAppleMusicISRCSearch(ctx, client, inputs, isrcs)
	if err != nil {
		return validationArtifacts{}, err
	}

	artifacts := validationArtifacts{
		albumBody:    albumBody,
		metadataBody: metadataBody,
		upcBody:      upcBody,
		isrcBody:     isrcBody,
	}
	artifacts.summary = buildValidationSummary(inputs, artifacts, title, artist, releaseDate, label, upc, isrcs)
	return artifacts, nil
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

func fetchAppleMusicAlbum(ctx context.Context, client *http.Client, inputs validationInputs) ([]byte, appleMusicAlbumResource, error) {
	albumURL := appleMusicCatalogBaseURL(inputs) + "/albums/" + inputs.parsed.ID + "?include=tracks"
	albumBody, err := getAPI(ctx, client, albumURL, inputs.developerToken)
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

func fetchAppleMusicUPCSearch(ctx context.Context, client *http.Client, inputs validationInputs, upc string) ([]byte, error) {
	if upc == "" {
		return nil, nil
	}
	upcURL := appleMusicCatalogBaseURL(inputs) + "/albums?filter[upc]=" + url.QueryEscape(upc)
	upcBody, err := getAPI(ctx, client, upcURL, inputs.developerToken)
	if err != nil {
		return nil, fmt.Errorf("search official apple music by upc: %w", err)
	}
	return upcBody, nil
}

func fetchAppleMusicISRCSearch(ctx context.Context, client *http.Client, inputs validationInputs, isrcs []string) ([]byte, error) {
	if len(isrcs) == 0 {
		return nil, nil
	}

	bodies := make([][]byte, 0, len(isrcs))
	for _, isrc := range isrcs {
		isrcURL := appleMusicCatalogBaseURL(inputs) + "/songs?filter[isrc]=" + url.QueryEscape(isrc)
		isrcBody, err := getAPI(ctx, client, isrcURL, inputs.developerToken)
		if err != nil {
			return nil, fmt.Errorf("search official apple music by isrc: %w", err)
		}
		bodies = append(bodies, isrcBody)
	}
	mergedBody, err := mergeAppleMusicSearchBodies(bodies)
	if err != nil {
		return nil, fmt.Errorf("merge official apple music isrc search results: %w", err)
	}
	return mergedBody, nil
}

func appleMusicCatalogBaseURL(inputs validationInputs) string {
	return inputs.opts.apiBaseURL + "/catalog/" + inputs.storefront
}

func mergeAppleMusicSearchBodies(bodies [][]byte) ([]byte, error) {
	merged := appleMusicSearchDocument{Data: make([]json.RawMessage, 0, len(bodies))}
	seen := make(map[string]struct{})
	for _, body := range bodies {
		var doc appleMusicSearchDocument
		if err := json.Unmarshal(body, &doc); err != nil {
			return nil, fmt.Errorf("decode official apple music isrc search payload: %w", err)
		}
		for _, item := range doc.Data {
			key := string(item)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			merged.Data = append(merged.Data, item)
		}
	}
	body, err := json.Marshal(merged)
	if err != nil {
		return nil, fmt.Errorf("encode official apple music isrc search payload: %w", err)
	}
	return body, nil
}

func getAPI(ctx context.Context, client *http.Client, endpoint string, developerToken string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build apple music api request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+developerToken)
	req.Header.Set("User-Agent", "ariadne/0.1 (+https://github.com/xmbshwll/ariadne)")

	resp, err := client.Do(req)
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
		if len(isrcs) >= defaultSearchLimit {
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
