package applemusic

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

// FetchAlbum loads Apple Music album metadata from the lookup API and maps it into the canonical model.
func (a *Adapter) FetchAlbum(ctx context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalAlbum, error) {
	if parsed.Service != model.ServiceAppleMusic {
		return nil, fmt.Errorf("%w: %s", errUnexpectedAppleMusicService, parsed.Service)
	}
	return a.fetchAlbumByID(ctx, parsed.ID, parsed.CanonicalURL, a.storefrontFor(parsed.RegionHint))
}

// FetchSong loads Apple Music song metadata from the lookup API and maps it into the canonical model.
func (a *Adapter) FetchSong(ctx context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalSong, error) {
	if parsed.Service != model.ServiceAppleMusic {
		return nil, fmt.Errorf("%w: %s", errUnexpectedAppleMusicService, parsed.Service)
	}
	return a.fetchSongByID(ctx, parsed.ID, parsed.CanonicalURL, a.storefrontFor(parsed.RegionHint))
}

func (a *Adapter) fetchAlbumByID(ctx context.Context, albumID string, canonicalURL string, storefront string) (*model.CanonicalAlbum, error) {
	lookupURL := fmt.Sprintf("%s/lookup?id=%s&entity=song&country=%s", a.lookupBaseURL, url.QueryEscape(albumID), url.QueryEscape(a.storefrontFor(storefront)))
	var payload lookupResponse
	if err := a.getJSON(ctx, lookupURL, &payload); err != nil {
		return nil, err
	}
	if len(payload.Results) == 0 {
		return nil, fmt.Errorf("%w: %s", errAppleMusicAlbumNotFound, albumID)
	}

	parsed := model.ParsedAlbumURL{
		Service:      model.ServiceAppleMusic,
		EntityType:   "album",
		ID:           albumID,
		CanonicalURL: canonicalURL,
		RegionHint:   a.storefrontFor(storefront),
	}
	if parsed.CanonicalURL == "" {
		parsed.CanonicalURL = canonicalCollectionURL(payload.Results[0].CollectionViewURL, "")
	}
	return toCanonicalAlbum(parsed, payload.Results), nil
}

func (a *Adapter) fetchSongByID(ctx context.Context, songID string, canonicalURL string, storefront string) (*model.CanonicalSong, error) {
	lookupURL := fmt.Sprintf("%s/lookup?id=%s&entity=song&country=%s", a.lookupBaseURL, url.QueryEscape(songID), url.QueryEscape(a.storefrontFor(storefront)))
	var payload lookupResponse
	if err := a.getJSON(ctx, lookupURL, &payload); err != nil {
		return nil, err
	}
	track, ok := firstSongLookupItem(payload.Results)
	if !ok {
		return nil, fmt.Errorf("%w: %s", errAppleMusicSongNotFound, songID)
	}

	parsed := model.ParsedAlbumURL{
		Service:      model.ServiceAppleMusic,
		EntityType:   entitySong,
		ID:           songID,
		CanonicalURL: canonicalURL,
		RegionHint:   a.storefrontFor(storefront),
	}
	if parsed.CanonicalURL == "" {
		parsed.CanonicalURL = canonicalTrackURL(track.CollectionViewURL, track.TrackID)
	}
	return toCanonicalSong(parsed, track), nil
}

func (a *Adapter) getJSON(ctx context.Context, requestURL string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return fmt.Errorf("build apple music request: %w", err)
	}
	req.Header.Set("User-Agent", "ariadne/0.1 (+https://github.com/xmbshwll/ariadne)")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("execute apple music request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("%w %d: %s", errUnexpectedAppleMusicStatus, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode apple music response: %w", err)
	}
	return nil
}

func firstSongLookupItem(items []lookupItem) (lookupItem, bool) {
	for _, item := range items {
		if item.TrackID == 0 {
			continue
		}
		if item.WrapperType != wrapperTypeTrack || item.Kind != entitySong {
			continue
		}
		return item, true
	}
	return lookupItem{}, false
}

func (a *Adapter) storefrontFor(regionHint string) string {
	if strings.TrimSpace(regionHint) == "" {
		return a.defaultStorefront
	}
	return strings.ToLower(regionHint)
}
