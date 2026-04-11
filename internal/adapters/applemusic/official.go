package applemusic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/xmbshwll/ariadne/internal/applemusicauth"
	"github.com/xmbshwll/ariadne/internal/model"
)

// SearchByUPC uses the official Apple Music catalog API when MusicKit auth is configured.
func (a *Adapter) SearchByUPC(ctx context.Context, upc string) ([]model.CandidateAlbum, error) {
	upc = strings.TrimSpace(upc)
	if upc == "" || !a.authEnabled() {
		return nil, nil
	}

	storefront := a.defaultStorefront
	endpoint := fmt.Sprintf("%s/catalog/%s/albums?filter[upc]=%s", a.apiBaseURL, url.PathEscape(storefront), url.QueryEscape(upc))
	var payload map[string]any
	if err := a.getOfficialJSON(ctx, endpoint, &payload); err != nil {
		return nil, fmt.Errorf("search apple music by upc: %w", err)
	}
	albumIDs := officialAlbumIDs(payload)
	return a.hydrateOfficialAlbums(ctx, albumIDs, storefront)
}

// SearchByISRC uses the official Apple Music catalog API when MusicKit auth is configured.
func (a *Adapter) SearchByISRC(ctx context.Context, isrcs []string) ([]model.CandidateAlbum, error) {
	if !a.authEnabled() {
		return nil, nil
	}

	storefront := a.defaultStorefront
	seenAlbumIDs := make(map[string]struct{}, len(isrcs))
	albumIDs := make([]string, 0, len(isrcs))
	for _, isrc := range isrcs {
		isrc = strings.TrimSpace(isrc)
		if isrc == "" {
			continue
		}
		endpoint := fmt.Sprintf("%s/catalog/%s/songs?filter[isrc]=%s", a.apiBaseURL, url.PathEscape(storefront), url.QueryEscape(isrc))
		var payload map[string]any
		if err := a.getOfficialJSON(ctx, endpoint, &payload); err != nil {
			return nil, fmt.Errorf("search apple music by isrc: %w", err)
		}
		for _, albumID := range officialAlbumIDsFromSongs(payload) {
			if _, ok := seenAlbumIDs[albumID]; ok {
				continue
			}
			seenAlbumIDs[albumID] = struct{}{}
			albumIDs = append(albumIDs, albumID)
			if len(albumIDs) >= searchLimit {
				return a.hydrateOfficialAlbums(ctx, albumIDs, storefront)
			}
		}
	}
	return a.hydrateOfficialAlbums(ctx, albumIDs, storefront)
}

// SearchSongByISRC uses the official Apple Music catalog API when MusicKit auth is configured.
func (a *Adapter) SearchSongByISRC(ctx context.Context, isrc string) ([]model.CandidateSong, error) {
	isrc = strings.TrimSpace(isrc)
	if isrc == "" || !a.authEnabled() {
		return nil, nil
	}

	storefront := a.defaultStorefront
	endpoint := fmt.Sprintf("%s/catalog/%s/songs?filter[isrc]=%s", a.apiBaseURL, url.PathEscape(storefront), url.QueryEscape(isrc))
	var payload map[string]any
	if err := a.getOfficialJSON(ctx, endpoint, &payload); err != nil {
		return nil, fmt.Errorf("search apple music song by isrc: %w", err)
	}
	songIDs := officialSongIDs(payload)
	return a.hydrateSongs(ctx, songIDs, storefront)
}

func (a *Adapter) authEnabled() bool {
	return a.appleMusicKeyID != "" && a.appleMusicTeamID != "" && a.appleMusicPrivateKeyPath != ""
}

func (a *Adapter) developerToken() (string, error) {
	if !a.authEnabled() {
		return "", ErrCredentialsNotConfigured
	}

	a.tokenMu.Lock()
	defer a.tokenMu.Unlock()
	now := time.Now()
	if a.cachedToken != "" && now.Before(a.tokenExpiresAt) {
		return a.cachedToken, nil
	}

	token, err := applemusicauth.GenerateDeveloperToken(applemusicauth.Config{
		KeyID:          a.appleMusicKeyID,
		TeamID:         a.appleMusicTeamID,
		PrivateKeyPath: a.appleMusicPrivateKeyPath,
		TTL:            time.Hour,
	}, now.UTC())
	if err != nil {
		return "", fmt.Errorf("generate apple music developer token: %w", err)
	}
	a.cachedToken = token
	a.tokenExpiresAt = now.Add(55 * time.Minute)
	return a.cachedToken, nil
}

func (a *Adapter) getOfficialJSON(ctx context.Context, requestURL string, target any) error {
	developerToken, err := a.developerToken()
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return fmt.Errorf("build apple music official request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+developerToken)
	req.Header.Set("User-Agent", "ariadne/0.1 (+https://github.com/xmbshwll/ariadne)")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("execute apple music official request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("%w %d: %s", errUnexpectedAppleMusicOfficialStatus, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode apple music official response: %w", err)
	}
	return nil
}

func (a *Adapter) hydrateOfficialAlbums(ctx context.Context, albumIDs []string, storefront string) ([]model.CandidateAlbum, error) {
	results := make([]model.CandidateAlbum, 0, len(albumIDs))
	seen := make(map[string]struct{}, len(albumIDs))
	for _, albumID := range albumIDs {
		albumID = strings.TrimSpace(albumID)
		if albumID == "" {
			continue
		}
		if _, ok := seen[albumID]; ok {
			continue
		}
		seen[albumID] = struct{}{}
		album, err := a.fetchOfficialAlbumByID(ctx, albumID, storefront)
		if err != nil {
			return nil, err
		}
		results = append(results, toCandidateAlbum(*album))
		if len(results) >= searchLimit {
			break
		}
	}
	return results, nil
}

func (a *Adapter) hydrateSongs(ctx context.Context, songIDs []string, storefront string) ([]model.CandidateSong, error) {
	results := make([]model.CandidateSong, 0, len(songIDs))
	seen := make(map[string]struct{}, len(songIDs))
	for _, songID := range songIDs {
		songID = strings.TrimSpace(songID)
		if songID == "" {
			continue
		}
		if _, ok := seen[songID]; ok {
			continue
		}
		seen[songID] = struct{}{}
		song, err := a.fetchSongByID(ctx, songID, "", storefront)
		if err != nil {
			return nil, err
		}
		results = append(results, toCandidateSong(*song))
		if len(results) >= searchLimit {
			break
		}
	}
	return results, nil
}

func (a *Adapter) fetchOfficialAlbumByID(ctx context.Context, albumID string, storefront string) (*model.CanonicalAlbum, error) {
	endpoint := fmt.Sprintf("%s/catalog/%s/albums/%s?include=tracks", a.apiBaseURL, url.PathEscape(storefront), url.PathEscape(albumID))
	var payload map[string]any
	if err := a.getOfficialJSON(ctx, endpoint, &payload); err != nil {
		return nil, fmt.Errorf("fetch apple music official album %s: %w", albumID, err)
	}
	resource := firstOfficialResource(payload)
	if resource == nil {
		return nil, fmt.Errorf("%w: %s", errAppleMusicOfficialAlbumNotFound, albumID)
	}
	return officialResourceToCanonicalAlbum(resource, storefront), nil
}

func firstOfficialResource(payload map[string]any) map[string]any {
	data, _ := payload["data"].([]any)
	if len(data) == 0 {
		return nil
	}
	resource, _ := data[0].(map[string]any)
	return resource
}

func officialAlbumIDs(payload map[string]any) []string {
	data, _ := payload["data"].([]any)
	ids := make([]string, 0, len(data))
	seen := make(map[string]struct{}, len(data))
	for _, item := range data {
		resource, ok := item.(map[string]any)
		if !ok {
			continue
		}
		ids = appendUniqueString(ids, seen, officialAlbumID(resource))
	}
	return ids
}

func officialSongIDs(payload map[string]any) []string {
	data, _ := payload["data"].([]any)
	ids := make([]string, 0, len(data))
	seen := make(map[string]struct{}, len(data))
	for _, item := range data {
		resource, ok := item.(map[string]any)
		if !ok {
			continue
		}
		ids = appendUniqueString(ids, seen, officialString(resource, "id"))
	}
	return ids
}

func officialAlbumIDsFromSongs(payload map[string]any) []string {
	data, _ := payload["data"].([]any)
	ids := make([]string, 0, len(data))
	seen := make(map[string]struct{}, len(data))
	for _, item := range data {
		resource, ok := item.(map[string]any)
		if !ok {
			continue
		}
		relationships := officialMap(resource, "relationships")
		albums := officialMap(relationships, "albums")
		albumData, _ := albums["data"].([]any)
		for _, candidate := range albumData {
			albumResource, ok := candidate.(map[string]any)
			if !ok {
				continue
			}
			ids = appendUniqueString(ids, seen, officialString(albumResource, "id"))
		}
	}
	return ids
}
