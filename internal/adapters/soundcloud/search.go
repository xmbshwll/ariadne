package soundcloud

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/xmbshwll/ariadne/internal/model"
)

func (a *Adapter) SearchByUPC(_ context.Context, _ string) ([]model.CandidateAlbum, error) {
	return nil, nil
}

func (a *Adapter) SearchByISRC(_ context.Context, _ []string) ([]model.CandidateAlbum, error) {
	return nil, nil
}

func (a *Adapter) SearchByMetadata(ctx context.Context, album model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
	query := metadataQuery(album)
	if query == "" {
		return nil, nil
	}
	clientID, err := a.clientIdentifier(ctx)
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("%s/search/playlists?q=%s&client_id=%s&limit=%d", a.apiBaseURL, url.QueryEscape(query), url.QueryEscape(clientID), searchLimit)
	var payload searchResponse
	if err := a.getJSON(ctx, endpoint, &payload); err != nil {
		return nil, fmt.Errorf("search soundcloud metadata: %w", err)
	}
	results := make([]model.CandidateAlbum, 0, min(len(payload.Collection), searchLimit))
	for _, playlist := range payload.Collection {
		if playlist.Kind != "playlist" {
			continue
		}
		canonical := toCanonicalAlbum(playlist)
		results = append(results, toCandidateAlbum(*canonical))
		if len(results) >= searchLimit {
			break
		}
	}
	return results, nil
}

func (a *Adapter) SearchSongByISRC(_ context.Context, _ string) ([]model.CandidateSong, error) {
	return nil, nil
}

func (a *Adapter) SearchSongByMetadata(ctx context.Context, song model.CanonicalSong) ([]model.CandidateSong, error) {
	query := songMetadataQuery(song)
	if query == "" {
		return nil, nil
	}
	clientID, err := a.clientIdentifier(ctx)
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("%s/search/tracks?q=%s&client_id=%s&limit=%d", a.apiBaseURL, url.QueryEscape(query), url.QueryEscape(clientID), searchLimit)
	var payload trackSearchResponse
	if err := a.getJSON(ctx, endpoint, &payload); err != nil {
		return nil, fmt.Errorf("search soundcloud song metadata: %w", err)
	}
	results := make([]model.CandidateSong, 0, min(len(payload.Collection), searchLimit))
	for _, track := range payload.Collection {
		canonical := toCanonicalSong(track)
		results = append(results, toCandidateSong(*canonical))
		if len(results) >= searchLimit {
			break
		}
	}
	return results, nil
}

func (a *Adapter) getJSON(ctx context.Context, requestURL string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return fmt.Errorf("build soundcloud api request: %w", err)
	}
	req.Header.Set("User-Agent", "ariadne/0.1 (+https://github.com/xmbshwll/ariadne)")
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("execute soundcloud api request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("%w %d: %s", errUnexpectedSoundCloudAPIStatus, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode soundcloud api response: %w", err)
	}
	return nil
}

func (a *Adapter) clientIdentifier(ctx context.Context) (string, error) {
	a.clientIDMu.Lock()
	cachedClientID := a.clientID
	a.clientIDMu.Unlock()
	if cachedClientID != "" {
		return cachedClientID, nil
	}

	body, err := a.fetchPage(ctx, a.siteBaseURL)
	if err != nil {
		return "", err
	}
	clientID, err := a.findClientID(ctx, body)
	if err != nil {
		return "", err
	}
	a.clientIDMu.Lock()
	defer a.clientIDMu.Unlock()
	a.clientID = clientID
	return a.clientID, nil
}

func (a *Adapter) maybeCacheClientIDFromPage(body []byte) {
	clientID := extractClientID(body)
	if clientID == "" {
		return
	}
	a.clientIDMu.Lock()
	defer a.clientIDMu.Unlock()
	if a.clientID == "" {
		a.clientID = clientID
	}
}

func (a *Adapter) findClientID(ctx context.Context, body []byte) (string, error) {
	if clientID := extractClientID(body); clientID != "" {
		return clientID, nil
	}
	scriptMatches := scriptSrcPattern.FindAllSubmatch(body, -1)
	for _, match := range scriptMatches {
		if len(match) != 2 {
			continue
		}
		scriptURL := strings.TrimSpace(string(match[1]))
		if scriptURL == "" {
			continue
		}
		if strings.HasPrefix(scriptURL, "/") {
			scriptURL = a.siteBaseURL + scriptURL
		}
		assetBody, err := a.fetchPage(ctx, scriptURL)
		if err != nil {
			continue
		}
		if clientID := extractClientID(assetBody); clientID != "" {
			return clientID, nil
		}
	}
	return "", errSoundCloudClientIDNotFound
}
