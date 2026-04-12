package tidal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/normalize"
)

const (
	maxTIDALTokenResponseBytes = 16 * 1024
	tidalTokenRefreshTimeout   = 30 * time.Second
)

var errTIDALTokenResponseTooLarge = errors.New("tidal token response too large")

func (a *Adapter) getAPIJSON(ctx context.Context, endpoint string, target any) error {
	token, err := a.accessToken()
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build api request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.api+json")
	req.Header.Set("User-Agent", "ariadne/0.1 (+https://github.com/xmbshwll/ariadne)")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("execute api request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("%w %d: %s", errUnexpectedTIDALAPIStatus, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode api response: %w", errors.Join(errMalformedTIDALAPIResponse, err))
	}
	return nil
}

func (a *Adapter) accessToken() (string, error) {
	if !a.hasCredentials() {
		return "", ErrCredentialsNotConfigured
	}

	if accessToken, ok := a.cachedAccessToken(); ok {
		return accessToken, nil
	}

	result, err, _ := a.tokenGroup.Do("tidal-token", func() (any, error) {
		if accessToken, ok := a.cachedAccessToken(); ok {
			return accessToken, nil
		}
		refreshCtx, cancel := context.WithTimeout(context.Background(), tidalTokenRefreshTimeout)
		defer cancel()
		return a.refreshAccessToken(refreshCtx)
	})
	if err != nil {
		//nolint:wrapcheck // Preserve refresh errors from the shared token fetch path.
		return "", err
	}
	accessToken, _ := result.(string)
	return accessToken, nil
}

func (a *Adapter) cachedAccessToken() (string, bool) {
	a.tokenMu.Lock()
	defer a.tokenMu.Unlock()
	if a.token.accessToken == "" || !time.Now().Before(a.token.expiresAt) {
		return "", false
	}
	return a.token.accessToken, true
}

func (a *Adapter) refreshAccessToken(ctx context.Context) (string, error) {
	form := url.Values{}
	form.Set("client_id", a.clientID)
	form.Set("client_secret", a.clientSecret)
	form.Set("grant_type", "client_credentials")
	endpoint := a.authBaseURL + "/oauth2/token"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("User-Agent", "ariadne/0.1 (+https://github.com/xmbshwll/ariadne)")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute token request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	limitedBody := io.LimitReader(resp.Body, maxTIDALTokenResponseBytes+1)
	body, err := io.ReadAll(limitedBody)
	if err != nil {
		return "", fmt.Errorf("read token response: %w", err)
	}
	if len(body) > maxTIDALTokenResponseBytes {
		return "", fmt.Errorf("read token response: %w (%d bytes max)", errTIDALTokenResponseTooLarge, maxTIDALTokenResponseBytes)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w %d: %s", errUnexpectedTIDALTokenStatus, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var token tokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}
	if strings.TrimSpace(token.AccessToken) == "" {
		return "", errEmptyTIDALAccessToken
	}
	ttl := max(token.ExpiresIn-30, 0)
	cached := cachedToken{
		accessToken: token.AccessToken,
		expiresAt:   time.Now().Add(time.Duration(ttl) * time.Second),
	}

	a.tokenMu.Lock()
	defer a.tokenMu.Unlock()
	if a.token.accessToken != "" && time.Now().Before(a.token.expiresAt) {
		return a.token.accessToken, nil
	}
	a.token = cached
	return a.token.accessToken, nil
}

func (a *Adapter) hasCredentials() bool {
	return strings.TrimSpace(a.clientID) != "" && strings.TrimSpace(a.clientSecret) != ""
}

func (a *Adapter) countryCodeFor(regionHint string) string {
	countryCode := normalizeCountryCode(regionHint)
	if countryCode == "" {
		return normalizeCountryCode(a.defaultCountryCode)
	}
	return countryCode
}

func normalizeCountryCode(value string) string {
	countryCode := strings.ToUpper(strings.TrimSpace(value))
	if len(countryCode) != 2 {
		return ""
	}
	return countryCode
}

func metadataQuery(album model.CanonicalAlbum) string {
	return normalize.SearchPrimaryQuery(album.Title, album.Artists)
}

func songMetadataQuery(song model.CanonicalSong) string {
	return normalize.SearchPrimaryQuery(song.Title, song.Artists)
}

func firstDataResource(document apiDocument) (apiResource, bool, error) {
	resources, err := documentData(document)
	if err != nil {
		return apiResource{}, false, err
	}
	if len(resources) == 0 {
		return apiResource{}, false, nil
	}
	return resources[0], true, nil
}

func documentData(document apiDocument) ([]apiResource, error) {
	switch data := document.Data.(type) {
	case nil:
		return nil, nil
	case map[string]any:
		content, err := json.Marshal(data)
		if err != nil {
			return nil, errMalformedTIDALAPIResponse
		}
		var resource apiResource
		if err := json.Unmarshal(content, &resource); err != nil {
			return nil, errMalformedTIDALAPIResponse
		}
		return []apiResource{resource}, nil
	case []any:
		resources := make([]apiResource, 0, len(data))
		for _, item := range data {
			content, err := json.Marshal(item)
			if err != nil {
				return nil, errMalformedTIDALAPIResponse
			}
			var resource apiResource
			if err := json.Unmarshal(content, &resource); err != nil {
				return nil, errMalformedTIDALAPIResponse
			}
			resources = append(resources, resource)
		}
		return resources, nil
	default:
		return nil, errMalformedTIDALAPIResponse
	}
}

func albumIDsFromTrackDocument(document apiDocument) ([]string, error) {
	resources, err := documentData(document)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{})
	ids := make([]string, 0, len(resources))
	appendUniqueID := func(id string) {
		id = strings.TrimSpace(id)
		if id == "" {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	for _, included := range document.Included {
		if included.Type != "albums" {
			continue
		}
		appendUniqueID(included.ID)
	}
	for _, resource := range resources {
		for _, relation := range resource.Relationships.Albums.Data {
			if relation.Type != "albums" {
				continue
			}
			appendUniqueID(relation.ID)
		}
	}
	return ids, nil
}
