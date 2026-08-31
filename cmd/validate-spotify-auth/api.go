package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/xmbshwll/ariadne/cmd/internal/validation"
)

const (
	defaultSpotifyAPIBaseURL  = "https://api.spotify.com/v1"
	defaultSpotifyAuthBaseURL = "https://accounts.spotify.com/api"
	searchLimit               = 5
	isrcSampleLimit           = 5
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
	client := &http.Client{Timeout: inputs.appConfig.HTTPTimeout}
	apiBaseURL := validation.JoinURL(inputs.opts.apiBaseURL)
	token, err := fetchToken(ctx, client, inputs.opts.authBaseURL, inputs.appConfig.Spotify.ClientID, inputs.appConfig.Spotify.ClientSecret)
	if err != nil {
		return validationArtifacts{}, err
	}

	albumBody, album, err := fetchSpotifyAlbum(ctx, client, apiBaseURL, inputs.parsed.ID, token)
	if err != nil {
		return validationArtifacts{}, err
	}

	upc, isrcs, metadata, err := validateSpotifyAlbumMetadata(ctx, client, apiBaseURL, token, album)
	if err != nil {
		return validationArtifacts{}, err
	}

	upcBody, err := getAPI(ctx, client, spotifySearchURL(apiBaseURL, "upc:"+upc, "album"), token)
	if err != nil {
		return validationArtifacts{}, fmt.Errorf("search spotify by upc: %w", err)
	}
	isrcBody, err := getAPI(ctx, client, spotifySearchURL(apiBaseURL, "isrc:"+isrcs[0], "track"), token)
	if err != nil {
		return validationArtifacts{}, fmt.Errorf("search spotify by isrc: %w", err)
	}
	metadataBody, err := getAPI(ctx, client, spotifySearchURL(apiBaseURL, metadata, "album"), token)
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

func fetchSpotifyAlbum(ctx context.Context, client *http.Client, apiBaseURL, albumID, token string) ([]byte, spotifyAlbumPayload, error) {
	albumBody, err := getAPI(ctx, client, validation.JoinURL(apiBaseURL, "albums", albumID), token)
	if err != nil {
		return nil, spotifyAlbumPayload{}, fmt.Errorf("fetch spotify album payload: %w", err)
	}

	var album spotifyAlbumPayload
	if err := validation.DecodeJSONInto(albumBody, &album, "decode album payload"); err != nil {
		return nil, spotifyAlbumPayload{}, err
	}
	return albumBody, album, nil
}

func validateSpotifyAlbumMetadata(ctx context.Context, client *http.Client, apiBaseURL, token string, album spotifyAlbumPayload) (string, []string, string, error) {
	upc := strings.TrimSpace(album.ExternalIDs.UPC)
	if upc == "" {
		return "", nil, "", errSpotifyUPCMissing
	}

	isrcs, err := collectTrackISRCs(ctx, client, apiBaseURL, token, album)
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

func fetchToken(ctx context.Context, client *http.Client, authBaseURL, clientID, clientSecret string) (string, error) {
	return validation.FetchClientCredentialsToken(ctx, validation.TokenRequest{
		Client:       client,
		Endpoint:     strings.TrimRight(authBaseURL, "/") + "/token",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		UseBasicAuth: true,
		BuildError:   "build spotify token request",
		ExecuteError: "execute spotify token request",
		StatusError:  errSpotifyTokenStatus,
		DecodeError:  "decode spotify token response",
		MissingError: errSpotifyTokenMissing,
	})
}

func getAPI(ctx context.Context, client *http.Client, endpoint string, token string) ([]byte, error) {
	return validation.AuthenticatedGet(ctx, validation.GetRequest{
		Client:       client,
		URL:          endpoint,
		Token:        token,
		BuildError:   "build spotify api request",
		ExecuteError: "execute spotify api request",
		StatusError:  errSpotifyAPIStatus,
		ReadError:    "read spotify api response",
	})
}

func collectTrackISRCs(ctx context.Context, client *http.Client, apiBaseURL string, token string, album spotifyAlbumPayload) ([]string, error) {
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
		body, err := getAPI(ctx, client, validation.JoinURL(apiBaseURL, "tracks", trackID), token)
		if err != nil {
			return nil, err
		}
		var payload spotifyTrackPayload
		if err := validation.DecodeJSONInto(body, &payload, "decode spotify track details payload"); err != nil {
			return nil, err
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
		if len(isrcs) >= isrcSampleLimit {
			return isrcs, nil
		}
	}
	return isrcs, nil
}

func spotifySearchURL(apiBaseURL string, query string, entityType string) string {
	path := "/search?q=" + url.QueryEscape(query) + "&type=" + entityType + "&limit=" + strconv.Itoa(searchLimit)
	return apiURL(apiBaseURL, path)
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

func normalizeBaseURL(baseURL string) string {
	return strings.TrimRight(baseURL, "/")
}

func apiURL(baseURL string, path string) string {
	return normalizeBaseURL(baseURL) + "/" + strings.TrimLeft(path, "/")
}
