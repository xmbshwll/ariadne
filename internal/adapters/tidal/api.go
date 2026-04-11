package tidal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/xmbshwll/ariadne/internal/model"
)

func (a *Adapter) getAPIJSON(ctx context.Context, endpoint string, target any) error {
	token, err := a.accessToken(ctx)
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
		return fmt.Errorf("decode api response: %w", err)
	}
	return nil
}

func (a *Adapter) accessToken(ctx context.Context) (string, error) {
	if !a.hasCredentials() {
		return "", ErrCredentialsNotConfigured
	}
	a.tokenMu.Lock()
	defer a.tokenMu.Unlock()
	if a.token.accessToken != "" && time.Now().Before(a.token.expiresAt) {
		return a.token.accessToken, nil
	}

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
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read token response: %w", err)
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
	a.token = cachedToken{
		accessToken: token.AccessToken,
		expiresAt:   time.Now().Add(time.Duration(ttl) * time.Second),
	}
	return a.token.accessToken, nil
}

func (a *Adapter) hasCredentials() bool {
	return strings.TrimSpace(a.clientID) != "" && strings.TrimSpace(a.clientSecret) != ""
}

func (a *Adapter) countryCodeFor(regionHint string) string {
	countryCode := normalizeCountryCode(regionHint)
	if countryCode == "" {
		return a.defaultCountryCode
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
	parts := make([]string, 0, 2)
	if album.Title != "" {
		parts = append(parts, album.Title)
	}
	if len(album.Artists) > 0 {
		parts = append(parts, album.Artists[0])
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func songMetadataQuery(song model.CanonicalSong) string {
	parts := make([]string, 0, 2)
	if song.Title != "" {
		parts = append(parts, song.Title)
	}
	if len(song.Artists) > 0 {
		parts = append(parts, song.Artists[0])
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func firstDataResource(document apiDocument) *apiResource {
	resources := documentData(document)
	if len(resources) == 0 {
		return nil
	}
	resource := resources[0]
	return &resource
}

func documentData(document apiDocument) []apiResource {
	switch data := document.Data.(type) {
	case nil:
		return nil
	case map[string]any:
		content, _ := json.Marshal(data)
		var resource apiResource
		if err := json.Unmarshal(content, &resource); err != nil {
			return nil
		}
		return []apiResource{resource}
	case []any:
		resources := make([]apiResource, 0, len(data))
		for _, item := range data {
			content, _ := json.Marshal(item)
			var resource apiResource
			if err := json.Unmarshal(content, &resource); err != nil {
				continue
			}
			resources = append(resources, resource)
		}
		return resources
	default:
		return nil
	}
}

func albumIDsFromTrackDocument(document apiDocument) []string {
	resources := documentData(document)
	seen := make(map[string]struct{})
	ids := make([]string, 0, len(resources))
	for _, included := range document.Included {
		if included.Type != "albums" {
			continue
		}
		if included.ID == "" {
			continue
		}
		if _, ok := seen[included.ID]; ok {
			continue
		}
		seen[included.ID] = struct{}{}
		ids = append(ids, included.ID)
	}
	if len(ids) > 0 {
		return ids
	}
	for _, resource := range resources {
		for _, relation := range resource.Relationships.Albums.Data {
			if relation.ID == "" {
				continue
			}
			if _, ok := seen[relation.ID]; ok {
				continue
			}
			seen[relation.ID] = struct{}{}
			ids = append(ids, relation.ID)
		}
	}
	return ids
}
