package main

import (
	"bytes"
	"context"
	"encoding/base64"
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
	defaultSpotifyAPIBaseURL  = "https://api.spotify.com/v1"
	defaultSpotifyAuthBaseURL = "https://accounts.spotify.com/api"
	searchLimit               = 5
)

var (
	errSpotifyUPCMissing      = errors.New("album payload did not include external_ids.upc")
	errSpotifyISRCMissing     = errors.New("spotify track detail payloads did not include any external_ids.isrc values")
	errSpotifyMetadataMissing = errors.New("album payload did not provide enough metadata for search validation")
	errSpotifyTokenStatus     = errors.New("unexpected spotify token status")
	errSpotifyTokenMissing    = errors.New("spotify token response did not include access_token")
	errSpotifyAPIStatus       = errors.New("unexpected spotify api status")
)

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

func collectValidationArtifacts(ctx context.Context, inputs validationInputs) (validationArtifacts, error) {
	apiBaseURL := normalizeBaseURL(inputs.opts.apiBaseURL)
	token, err := fetchToken(ctx, inputs.opts.authBaseURL, inputs.appConfig.Spotify.ClientID, inputs.appConfig.Spotify.ClientSecret)
	if err != nil {
		return validationArtifacts{}, err
	}

	albumBody, album, err := fetchSpotifyAlbum(ctx, apiBaseURL, inputs.parsed.ID, token)
	if err != nil {
		return validationArtifacts{}, err
	}

	upc, isrcs, metadata, err := validateSpotifyAlbumMetadata(ctx, apiBaseURL, token, album)
	if err != nil {
		return validationArtifacts{}, err
	}

	upcBody, err := getAPI(ctx, apiURL(apiBaseURL, "/search?q="+url.QueryEscape("upc:"+upc)+"&type=album&limit="+strconv.Itoa(searchLimit)), token)
	if err != nil {
		return validationArtifacts{}, fmt.Errorf("search spotify by upc: %w", err)
	}
	isrcBody, err := getAPI(ctx, apiURL(apiBaseURL, "/search?q="+url.QueryEscape("isrc:"+isrcs[0])+"&type=track&limit="+strconv.Itoa(searchLimit)), token)
	if err != nil {
		return validationArtifacts{}, fmt.Errorf("search spotify by isrc: %w", err)
	}
	metadataBody, err := getAPI(ctx, apiURL(apiBaseURL, "/search?q="+url.QueryEscape(metadata)+"&type=album&limit="+strconv.Itoa(searchLimit)), token)
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
	albumBody, err := getAPI(ctx, apiURL(apiBaseURL, "/albums/"+albumID), token)
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
		body, err := getAPI(ctx, apiURL(apiBaseURL, "/tracks/"+trackID), token)
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

func normalizeBaseURL(baseURL string) string {
	return strings.TrimRight(baseURL, "/")
}

func apiURL(baseURL string, path string) string {
	return normalizeBaseURL(baseURL) + "/" + strings.TrimLeft(path, "/")
}
