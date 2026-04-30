package soundcloud

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/xmbshwll/ariadne/internal/adapters/adapterutil"
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
	var payload searchResponse
	if err := a.getSearchJSON(ctx, "/search/playlists", query, &payload); err != nil {
		return nil, fmt.Errorf("search soundcloud metadata: %w", err)
	}
	results, err := adapterutil.CollectCandidates(
		payload.Collection,
		searchLimit,
		soundCloudPlaylistCandidateID,
		soundCloudAlbumSearchCandidate,
	)
	if err != nil {
		return nil, fmt.Errorf("collect soundcloud album candidates: %w", err)
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
	results, err := adapterutil.CollectCandidates(
		payload.Collection,
		searchLimit,
		soundCloudSongCandidateID,
		soundCloudSongSearchCandidate,
	)
	if err != nil {
		return nil, fmt.Errorf("collect soundcloud song candidates: %w", err)
	}
	return results, nil
}

func validSoundCloudPlaylistSearchHit(playlist soundPlaylist) bool {
	return playlist.Kind == "playlist" && strings.TrimSpace(playlist.PermalinkURL) != "" && strings.TrimSpace(playlist.Title) != ""
}

func validSoundCloudTrackSearchHit(track soundTrack) bool {
	return strings.TrimSpace(track.PermalinkURL) != "" && strings.TrimSpace(track.Title) != ""
}

func soundCloudPlaylistCandidateID(playlist soundPlaylist) string {
	if !validSoundCloudPlaylistSearchHit(playlist) {
		return ""
	}
	return soundCloudCandidateID(playlist.PermalinkURL)
}

func soundCloudSongCandidateID(track soundTrack) string {
	if !validSoundCloudTrackSearchHit(track) {
		return ""
	}
	return soundCloudCandidateID(track.PermalinkURL)
}

func soundCloudCandidateID(rawURL string) string {
	canonicalURL := canonicalizeSoundCloudURL(rawURL)
	if canonicalURL == "" {
		return ""
	}
	return soundCloudSourceID(canonicalURL)
}

func soundCloudAlbumSearchCandidate(playlist soundPlaylist) (model.CandidateAlbum, error) {
	canonical := toCanonicalAlbum(playlist)
	return toCandidateAlbum(*canonical), nil
}

func soundCloudSongSearchCandidate(track soundTrack) (model.CandidateSong, error) {
	canonical := toCanonicalSong(track)
	return toCandidateSong(*canonical), nil
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
	//nolint:wrapcheck // HTTP exchange spec supplies request/status/decode context.
	return adapterutil.GetJSON(ctx, adapterutil.JSONRequest{
		RequestSpec: adapterutil.RequestSpec{
			Client:       a.client,
			URL:          requestURL,
			UserAgent:    adapterutil.DefaultUserAgent,
			BuildError:   "build soundcloud api request",
			ExecuteError: "execute soundcloud api request",
			StatusError:  adapterutil.StatusError(errUnexpectedSoundCloudAPIStatus),
		},
		DecodeError: "decode soundcloud api response",
	}, target)
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
	for _, match := range scriptMatches {
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
