package soundcloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/xmbshwll/ariadne/internal/model"
)

const maxSoundCloudClientIDProbes = 8

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
	var payload searchResponse
	if err := a.getSearchJSON(ctx, "/search/playlists", query, &payload); err != nil {
		return nil, fmt.Errorf("search soundcloud metadata: %w", err)
	}
	results := make([]model.CandidateAlbum, 0, min(len(payload.Collection), searchLimit))
	seen := make(map[string]struct{}, searchLimit)
	for _, playlist := range payload.Collection {
		if !validSoundCloudPlaylistSearchHit(playlist) {
			continue
		}
		canonical := toCanonicalAlbum(playlist)
		if !appendSoundCloudAlbumCandidate(&results, seen, canonical) {
			continue
		}
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
	var payload trackSearchResponse
	if err := a.getSearchJSON(ctx, "/search/tracks", query, &payload); err != nil {
		return nil, fmt.Errorf("search soundcloud song metadata: %w", err)
	}
	results := make([]model.CandidateSong, 0, min(len(payload.Collection), searchLimit))
	seen := make(map[string]struct{}, searchLimit)
	for _, track := range payload.Collection {
		if !validSoundCloudTrackSearchHit(track) {
			continue
		}
		canonical := toCanonicalSong(track)
		if !appendSoundCloudSongCandidate(&results, seen, canonical) {
			continue
		}
		if len(results) >= searchLimit {
			break
		}
	}
	return results, nil
}

func validSoundCloudPlaylistSearchHit(playlist soundPlaylist) bool {
	return playlist.Kind == "playlist" && strings.TrimSpace(playlist.PermalinkURL) != "" && strings.TrimSpace(playlist.Title) != ""
}

func validSoundCloudTrackSearchHit(track soundTrack) bool {
	return strings.TrimSpace(track.PermalinkURL) != "" && strings.TrimSpace(track.Title) != ""
}

func appendSoundCloudAlbumCandidate(results *[]model.CandidateAlbum, seen map[string]struct{}, canonical *model.CanonicalAlbum) bool {
	if canonical == nil || strings.TrimSpace(canonical.SourceID) == "" || strings.TrimSpace(canonical.SourceURL) == "" {
		return false
	}
	if _, ok := seen[canonical.SourceID]; ok {
		return false
	}
	seen[canonical.SourceID] = struct{}{}
	*results = append(*results, toCandidateAlbum(*canonical))
	return true
}

func appendSoundCloudSongCandidate(results *[]model.CandidateSong, seen map[string]struct{}, canonical *model.CanonicalSong) bool {
	if canonical == nil || strings.TrimSpace(canonical.SourceID) == "" || strings.TrimSpace(canonical.SourceURL) == "" {
		return false
	}
	if _, ok := seen[canonical.SourceID]; ok {
		return false
	}
	seen[canonical.SourceID] = struct{}{}
	*results = append(*results, toCandidateSong(*canonical))
	return true
}

func (a *Adapter) getSearchJSON(ctx context.Context, path string, query string, target any) error {
	clientID, err := a.clientIdentifier(ctx)
	if err != nil {
		return err
	}
	requestURL := a.searchURL(path, query, clientID)
	if err := a.getJSON(ctx, requestURL, target); err == nil {
		return nil
	} else if !isSoundCloudClientIDError(err) {
		return err
	}

	clientID, err = a.refreshClientIdentifier(ctx)
	if err != nil {
		return err
	}
	return a.getJSON(ctx, a.searchURL(path, query, clientID), target)
}

func (a *Adapter) searchURL(path string, query string, clientID string) string {
	return fmt.Sprintf("%s%s?q=%s&client_id=%s&limit=%d", a.apiBaseURL, path, url.QueryEscape(query), url.QueryEscape(clientID), searchLimit)
}

func (a *Adapter) refreshClientIdentifier(ctx context.Context) (string, error) {
	a.clientIDMu.Lock()
	a.clientID = ""
	a.clientIDMu.Unlock()
	return a.clientIdentifier(ctx)
}

func isSoundCloudClientIDError(err error) bool {
	if !errors.Is(err, errUnexpectedSoundCloudAPIStatus) {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, " 401:") || strings.Contains(message, " 403:") || strings.Contains(message, "client_id") || strings.Contains(message, "client id")
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
	if a.clientID != "" {
		return a.clientID, nil
	}
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
	probes := 0
	for _, match := range scriptMatches {
		if probes >= maxSoundCloudClientIDProbes {
			break
		}
		if len(match) != 2 {
			continue
		}
		scriptURL := strings.TrimSpace(string(match[1]))
		if scriptURL == "" {
			continue
		}
		resolvedURL, err := resolveSoundCloudAssetURL(a.siteBaseURL, scriptURL)
		if err != nil {
			continue
		}
		probes++
		assetBody, err := a.fetchPage(ctx, resolvedURL)
		if err != nil {
			continue
		}
		if clientID := extractClientID(assetBody); clientID != "" {
			return clientID, nil
		}
	}
	return "", errSoundCloudClientIDNotFound
}

func resolveSoundCloudAssetURL(baseURL string, assetURL string) (string, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("parse soundcloud asset base url: %w", err)
	}
	ref, err := url.Parse(assetURL)
	if err != nil {
		return "", fmt.Errorf("parse soundcloud asset url: %w", err)
	}
	if ref.Host != "" && ref.Scheme == "" {
		ref.Scheme = base.Scheme
		return ref.String(), nil
	}
	return base.ResolveReference(ref).String(), nil
}
