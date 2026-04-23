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
