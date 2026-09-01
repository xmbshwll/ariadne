package soundcloud

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/xmbshwll/ariadne/internal/adapters/canonical"
	"github.com/xmbshwll/ariadne/internal/adapters/search"
	"github.com/xmbshwll/ariadne/internal/httpx"
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/targetsearch"
)

func (a *Adapter) SearchAlbumByMetadata(ctx context.Context, album model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
	query := metadataQuery(album)
	if query == "" {
		return nil, nil
	}
	var payload searchResponse
	if err := a.getSearchJSON(ctx, "/search/playlists", query, &payload); err != nil {
		if targetsearch.IsUnavailable(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("search soundcloud metadata: %w", err)
	}
	results, err := search.CollectCandidates(
		ctx,
		payload.Collection,
		searchLimit,
		soundCloudPlaylistCandidateID,
		func(_ context.Context, playlist soundPlaylist) (model.CandidateAlbum, error) {
			return soundCloudAlbumSearchCandidate(playlist)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("collect soundcloud album candidates: %w", err)
	}
	return results, nil
}

func (a *Adapter) SearchSongByMetadata(ctx context.Context, song model.CanonicalSong) ([]model.CandidateSong, error) {
	query := songMetadataQuery(song)
	if query == "" {
		return nil, nil
	}
	var payload trackSearchResponse
	if err := a.getSearchJSON(ctx, "/search/tracks", query, &payload); err != nil {
		if targetsearch.IsUnavailable(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("search soundcloud song metadata: %w", err)
	}
	results, err := search.CollectCandidates(
		ctx,
		payload.Collection,
		searchLimit,
		soundCloudSongCandidateID,
		func(_ context.Context, track SoundTrack) (model.CandidateSong, error) {
			return soundCloudSongSearchCandidate(track)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("collect soundcloud song candidates: %w", err)
	}
	return results, nil
}

func validSoundCloudPlaylistSearchHit(playlist soundPlaylist) bool {
	return playlist.Kind == "playlist" && strings.TrimSpace(playlist.PermalinkURL) != "" && strings.TrimSpace(playlist.Title) != ""
}

func validSoundCloudTrackSearchHit(track SoundTrack) bool {
	return strings.TrimSpace(track.PermalinkURL) != "" && strings.TrimSpace(track.Title) != ""
}

func soundCloudPlaylistCandidateID(playlist soundPlaylist) string {
	if !validSoundCloudPlaylistSearchHit(playlist) {
		return ""
	}
	return soundCloudCandidateID(playlist.PermalinkURL)
}

func soundCloudSongCandidateID(track SoundTrack) string {
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
	mapped := ToCanonicalAlbum(playlist)
	return canonical.CandidateAlbum(*mapped), nil
}

func soundCloudSongSearchCandidate(track SoundTrack) (model.CandidateSong, error) {
	mapped := ToCanonicalSong(track)
	return canonical.CandidateSong(*mapped), nil
}

// isSoundCloudClientIDError answers whether a search error means the cached
// client id was rejected and must be rediscovered.
func isSoundCloudClientIDError(err error) bool {
	if statusErr, ok := errors.AsType[httpx.HTTPStatusError](err); ok {
		switch statusErr.HTTPStatusCode() {
		case http.StatusUnauthorized, http.StatusForbidden:
			return true
		}
	}
	if !errors.Is(err, errUnexpectedSoundCloudAPIStatus) {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "client_id") || strings.Contains(message, "client id")
}

// discoverClientID finds the SoundCloud web client id: fetch the site root,
// then follow asset scripts until the id appears.
func (a *Adapter) discoverClientID(ctx context.Context) (string, error) {
	body, err := a.fetchPage(ctx, a.siteBaseURL)
	if err != nil {
		return "", err
	}
	return a.findClientID(ctx, body)
}

func (a *Adapter) getSearchJSON(ctx context.Context, path string, query string, target any) error {
	clientID, err := a.clientIDs.Credential(ctx)
	if err != nil {
		return classifySoundCloudTargetSearchError(err)
	}
	err = a.getSearchJSONWithClientID(ctx, path, query, clientID, target)
	if err == nil {
		return nil
	}
	if !a.clientIDs.ShouldInvalidate(err) {
		return err
	}
	a.clientIDs.Invalidate()

	clientID, err = a.clientIDs.Credential(ctx)
	if err != nil {
		return classifySoundCloudTargetSearchError(err)
	}
	return a.getSearchJSONWithClientID(ctx, path, query, clientID, target)
}

func (a *Adapter) getSearchJSONWithClientID(ctx context.Context, path string, query string, clientID string, target any) error {
	return a.getJSON(ctx, a.searchURL(path, query, clientID), target)
}

func classifySoundCloudTargetSearchError(err error) error {
	if errors.Is(err, errSoundCloudClientIDNotFound) {
		return fmt.Errorf("soundcloud target search unavailable: %w", targetsearch.Unavailable(err))
	}
	return err
}

func (a *Adapter) searchURL(path string, query string, clientID string) string {
	return fmt.Sprintf("%s%s?q=%s&client_id=%s&limit=%d", a.apiBaseURL, path, url.QueryEscape(query), url.QueryEscape(clientID), searchLimit)
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

func (a *Adapter) getJSON(ctx context.Context, requestURL string, target any) error {
	//nolint:wrapcheck // HTTP exchange spec supplies request/status/decode context.
	return httpx.GetJSON(ctx, httpx.JSONRequest{
		RequestSpec: httpx.RequestSpec{
			Client:       a.client,
			URL:          requestURL,
			UserAgent:    httpx.DefaultUserAgent,
			BuildError:   "build soundcloud api request",
			ExecuteError: "execute soundcloud api request",
			StatusError:  httpx.StatusError(errUnexpectedSoundCloudAPIStatus),
		},
		DecodeError: "decode soundcloud api response",
	}, target)
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
