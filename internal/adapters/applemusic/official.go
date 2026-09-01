package applemusic

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/xmbshwll/ariadne/internal/canonical"
	"github.com/xmbshwll/ariadne/internal/httpx"
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/normalize"
	"github.com/xmbshwll/ariadne/internal/targetsearch"
)

// SearchAlbumByUPC uses the official Apple Music catalog API when MusicKit auth is configured.
func (a *Adapter) SearchAlbumByUPC(ctx context.Context, upc string) ([]model.CandidateAlbum, error) {
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
	return a.HydrateOfficialAlbums(ctx, albumIDs, storefront)
}

// SearchAlbumByISRC uses the official Apple Music catalog API when MusicKit auth is configured.
func (a *Adapter) SearchAlbumByISRC(ctx context.Context, isrcs []string) ([]model.CandidateAlbum, error) {
	if !a.authEnabled() {
		return nil, nil
	}

	storefront := a.defaultStorefront
	isrcs = normalize.NonEmpty(isrcs)
	seenAlbumIDs := make(map[string]struct{}, len(isrcs))
	albumIDs := make([]string, 0, len(isrcs))
	var firstErr error
	for _, isrc := range isrcs {
		endpoint := fmt.Sprintf("%s/catalog/%s/songs?filter[isrc]=%s", a.apiBaseURL, url.PathEscape(storefront), url.QueryEscape(isrc))
		var payload map[string]any
		if err := a.getOfficialJSON(ctx, endpoint, &payload); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		albumIDs = appendUniqueOfficialAlbumIDs(albumIDs, seenAlbumIDs, officialAlbumIDsFromSongs(payload))
		if len(albumIDs) >= searchLimit {
			return a.HydrateOfficialAlbums(ctx, albumIDs, storefront)
		}
	}
	if len(albumIDs) == 0 && firstErr != nil {
		return nil, fmt.Errorf("search apple music by isrc: %w", firstErr)
	}
	return a.HydrateOfficialAlbums(ctx, albumIDs, storefront)
}

func appendUniqueOfficialAlbumIDs(dst []string, seen map[string]struct{}, ids []string) []string {
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		dst = append(dst, id)
	}
	return dst
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
	return a.HydrateSongs(ctx, songIDs, storefront)
}

func (a *Adapter) authEnabled() bool {
	return a.developerTokens.Configured()
}

func (a *Adapter) developerToken(ctx context.Context) (string, error) {
	if !a.authEnabled() {
		return "", ErrCredentialsNotConfigured
	}
	//nolint:wrapcheck // The token source preserves its own error identity.
	return a.developerTokens.AccessToken(ctx)
}

func (a *Adapter) getOfficialJSON(ctx context.Context, requestURL string, target any) error {
	developerToken, err := a.developerToken(ctx)
	if err != nil {
		return err
	}

	//nolint:wrapcheck // HTTP exchange spec supplies request/status/decode context.
	return httpx.GetJSON(ctx, httpx.JSONRequest{
		RequestSpec: httpx.RequestSpec{
			Client:       a.client,
			URL:          requestURL,
			Headers:      map[string]string{"Authorization": "Bearer " + developerToken},
			UserAgent:    httpx.DefaultUserAgent,
			BuildError:   "build apple music official request",
			ExecuteError: "execute apple music official request",
			StatusError:  httpx.StatusError(errUnexpectedAppleMusicOfficialStatus),
		},
		DecodeError:       "decode apple music official response",
		MalformedResponse: ErrMalformedAppleMusicOfficialResponse,
	}, target)
}

func (a *Adapter) HydrateOfficialAlbums(ctx context.Context, albumIDs []string, storefront string) ([]model.CandidateAlbum, error) {
	return hydrateAppleMusicOfficialCandidates(
		ctx,
		albumIDs,
		func(albumID string) string { return albumID },
		func(ctx context.Context, albumID string) (model.CandidateAlbum, error) {
			album, err := a.fetchOfficialAlbumByID(ctx, albumID, storefront)
			if err != nil {
				return model.CandidateAlbum{}, err
			}
			return canonical.CandidateAlbum(*album), nil
		},
	)
}

func (a *Adapter) HydrateSongs(ctx context.Context, songIDs []string, storefront string) ([]model.CandidateSong, error) {
	return hydrateAppleMusicOfficialCandidates(
		ctx,
		songIDs,
		func(songID string) string { return songID },
		func(ctx context.Context, songID string) (model.CandidateSong, error) {
			song, err := a.fetchSongByID(ctx, songID, "", storefront)
			if err != nil {
				return model.CandidateSong{}, err
			}
			return canonical.CandidateSong(*song), nil
		},
	)
}

func hydrateAppleMusicOfficialCandidates[Input any, Candidate any](ctx context.Context, items []Input, itemID func(Input) string, fetch func(context.Context, Input) (Candidate, error)) ([]Candidate, error) {
	//nolint:wrapcheck // Preserve per-item fetch errors from the shared candidate collector.
	return targetsearch.CollectCandidates(ctx, items, searchLimit, itemID, fetch)
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
