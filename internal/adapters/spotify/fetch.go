package spotify

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/xmbshwll/ariadne/internal/model"
)

// FetchAlbum loads a Spotify album via the Web API when credentials are configured,
// otherwise falls back to the public album page bootstrap.
func (a *Adapter) FetchAlbum(ctx context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalAlbum, error) {
	if parsed.Service != model.ServiceSpotify {
		return nil, fmt.Errorf("%w: %s", errUnexpectedSpotifyService, parsed.Service)
	}

	if a.hasCredentials() {
		album, err := a.fetchAlbumAPI(ctx, parsed.ID)
		if err == nil {
			return toCanonicalAlbumAPI(parsed.CanonicalURL, album), nil
		}
		if !errors.Is(err, errSpotifyAlbumNotFound) {
			return nil, err
		}
	}

	return a.fetchAlbumBootstrap(ctx, parsed)
}

// SearchByUPC searches Spotify albums by UPC via the Web API.
func (a *Adapter) SearchByUPC(ctx context.Context, upc string) ([]model.CandidateAlbum, error) {
	upc = strings.TrimSpace(upc)
	if upc == "" {
		return nil, nil
	}
	if !a.hasCredentials() {
		return nil, ErrCredentialsNotConfigured
	}

	endpoint := fmt.Sprintf("%s/search?q=%s&type=album&limit=%d", a.apiBaseURL, url.QueryEscape("upc:"+upc), searchLimit)
	var response apiAlbumSearchResponse
	if err := a.getAPIJSON(ctx, endpoint, &response); err != nil {
		return nil, fmt.Errorf("spotify search by upc: %w", err)
	}
	return a.hydrateAlbumCandidates(ctx, response.Albums.Items)
}

// SearchByISRC searches Spotify track results by ISRC, then hydrates the owning albums.
func (a *Adapter) SearchByISRC(ctx context.Context, isrcs []string) ([]model.CandidateAlbum, error) {
	trimmedISRCs := make([]string, 0, len(isrcs))
	for _, isrc := range isrcs {
		isrc = strings.TrimSpace(isrc)
		if isrc == "" {
			continue
		}
		trimmedISRCs = append(trimmedISRCs, isrc)
	}
	if len(trimmedISRCs) == 0 {
		return nil, nil
	}
	if !a.hasCredentials() {
		return nil, ErrCredentialsNotConfigured
	}

	albumIDs := make([]string, 0, len(trimmedISRCs))
	seen := make(map[string]struct{}, len(trimmedISRCs))
	for _, isrc := range trimmedISRCs {
		endpoint := fmt.Sprintf("%s/search?q=%s&type=track&limit=%d", a.apiBaseURL, url.QueryEscape("isrc:"+isrc), 1)
		var response apiTrackSearchResponse
		if err := a.getAPIJSON(ctx, endpoint, &response); err != nil {
			if len(albumIDs) == 0 {
				return nil, fmt.Errorf("spotify search by isrc %s: %w", isrc, err)
			}
			continue
		}
		for _, item := range response.Tracks.Items {
			if item.Album.ID == "" {
				continue
			}
			if _, ok := seen[item.Album.ID]; ok {
				continue
			}
			seen[item.Album.ID] = struct{}{}
			albumIDs = append(albumIDs, item.Album.ID)
			if len(albumIDs) >= searchLimit {
				return a.hydrateAlbumCandidates(ctx, albumIDsToSummaries(albumIDs))
			}
		}
	}
	return a.hydrateAlbumCandidates(ctx, albumIDsToSummaries(albumIDs))
}

// SearchByMetadata searches Spotify albums by title and artist metadata.
func (a *Adapter) SearchByMetadata(ctx context.Context, album model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
	queries := metadataQueries(album)
	if len(queries) == 0 {
		return nil, nil
	}
	if !a.hasCredentials() {
		return nil, ErrCredentialsNotConfigured
	}

	items, err := collectSpotifySearchResults(
		queries,
		func(query string) ([]apiAlbumSummary, error) {
			endpoint := fmt.Sprintf("%s/search?q=%s&type=album&limit=%d", a.apiBaseURL, url.QueryEscape(query), searchLimit)
			var response apiAlbumSearchResponse
			if err := a.getAPIJSON(ctx, endpoint, &response); err != nil {
				return nil, fmt.Errorf("spotify search by metadata %q: %w", query, err)
			}
			return response.Albums.Items, nil
		},
		func(item apiAlbumSummary) string { return item.ID },
	)
	if err != nil {
		return nil, err
	}
	return a.hydrateAlbumCandidates(ctx, items)
}

// FetchSong loads a Spotify track via the Web API.
func (a *Adapter) FetchSong(ctx context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalSong, error) {
	if parsed.Service != model.ServiceSpotify {
		return nil, fmt.Errorf("%w: %s", errUnexpectedSpotifyService, parsed.Service)
	}
	if !a.hasCredentials() {
		return nil, ErrCredentialsNotConfigured
	}

	track, err := a.fetchTrackAPI(ctx, parsed.ID)
	if err != nil {
		return nil, fmt.Errorf("spotify fetch song api %s: %w", parsed.ID, err)
	}
	return toCanonicalSongAPI(parsed.CanonicalURL, track), nil
}

// SearchSongByISRC searches Spotify tracks by ISRC.
func (a *Adapter) SearchSongByISRC(ctx context.Context, isrc string) ([]model.CandidateSong, error) {
	if strings.TrimSpace(isrc) == "" {
		return nil, nil
	}
	if !a.hasCredentials() {
		return nil, ErrCredentialsNotConfigured
	}

	endpoint := fmt.Sprintf("%s/search?q=%s&type=track&limit=%d", a.apiBaseURL, url.QueryEscape("isrc:"+strings.TrimSpace(isrc)), searchLimit)
	var response apiTrackSearchResponse
	if err := a.getAPIJSON(ctx, endpoint, &response); err != nil {
		return nil, fmt.Errorf("spotify song search by isrc %s: %w", isrc, err)
	}
	return a.hydrateSongCandidates(ctx, response.Tracks.Items)
}

// SearchSongByMetadata searches Spotify tracks by title and artist metadata.
func (a *Adapter) SearchSongByMetadata(ctx context.Context, song model.CanonicalSong) ([]model.CandidateSong, error) {
	queries := songMetadataQueries(song)
	if len(queries) == 0 {
		return nil, nil
	}
	if !a.hasCredentials() {
		return nil, ErrCredentialsNotConfigured
	}

	items, err := collectSpotifySearchResults(
		queries,
		func(query string) ([]apiTrackSearchItem, error) {
			endpoint := fmt.Sprintf("%s/search?q=%s&type=track&limit=%d", a.apiBaseURL, url.QueryEscape(query), searchLimit)
			var response apiTrackSearchResponse
			if err := a.getAPIJSON(ctx, endpoint, &response); err != nil {
				return nil, fmt.Errorf("spotify song search by metadata %q: %w", query, err)
			}
			return response.Tracks.Items, nil
		},
		func(item apiTrackSearchItem) string { return item.ID },
	)
	if err != nil {
		return nil, err
	}
	return a.hydrateSongCandidates(ctx, items)
}

func collectSpotifySearchResults[T any](queries []string, search func(string) ([]T, error), itemID func(T) string) ([]T, error) {
	items := make([]T, 0, searchLimit)
	seen := make(map[string]struct{}, searchLimit)
	for _, query := range queries {
		results, err := search(query)
		if err != nil {
			if len(items) == 0 {
				return nil, err
			}
			continue
		}
		for _, item := range results {
			id := strings.TrimSpace(itemID(item))
			if id == "" {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			items = append(items, item)
			if len(items) >= searchLimit {
				return items, nil
			}
		}
	}
	return items, nil
}

func (a *Adapter) fetchAlbumAPI(ctx context.Context, albumID string) (*apiAlbumResponse, error) {
	var album apiAlbumResponse
	endpoint := a.apiBaseURL + "/albums/" + albumID
	if err := a.getAPIJSON(ctx, endpoint, &album); err != nil {
		if isSpotifyAPIStatus(err, http.StatusNotFound) {
			return nil, fmt.Errorf("%w: %s", errSpotifyAlbumNotFound, albumID)
		}
		return nil, fmt.Errorf("spotify fetch album api %s: %w", albumID, err)
	}
	if err := a.hydrateAlbumTrackDetails(ctx, &album); err != nil {
		return nil, fmt.Errorf("spotify hydrate track details %s: %w", albumID, err)
	}
	return &album, nil
}

func (a *Adapter) hydrateAlbumTrackDetails(ctx context.Context, album *apiAlbumResponse) error {
	trackIDs := make([]string, 0, len(album.Tracks.Items))
	for _, track := range album.Tracks.Items {
		if track.ID == "" {
			continue
		}
		trackIDs = append(trackIDs, track.ID)
	}
	if len(trackIDs) == 0 {
		return nil
	}

	trackDetails, err := a.fetchTrackDetailsAPI(ctx, trackIDs)
	if err != nil {
		return err
	}
	byID := make(map[string]apiTrack, len(trackDetails))
	for _, track := range trackDetails {
		if track.ID == "" {
			continue
		}
		byID[track.ID] = track
	}
	for i := range album.Tracks.Items {
		track := album.Tracks.Items[i]
		detail, ok := byID[track.ID]
		if !ok {
			continue
		}
		album.Tracks.Items[i].ExternalIDs = detail.ExternalIDs
		if len(detail.Artists) > 0 {
			album.Tracks.Items[i].Artists = detail.Artists
		}
		if detail.DurationMS > 0 {
			album.Tracks.Items[i].DurationMS = detail.DurationMS
		}
		album.Tracks.Items[i].Explicit = detail.Explicit
	}
	return nil
}

func (a *Adapter) fetchTrackAPI(ctx context.Context, trackID string) (*apiTrack, error) {
	tracks, err := a.fetchTrackDetailsAPI(ctx, []string{trackID})
	if err != nil {
		return nil, err
	}
	if len(tracks) == 0 {
		return nil, fmt.Errorf("%w: %s", errSpotifyTrackNotFound, trackID)
	}
	return &tracks[0], nil
}

func (a *Adapter) fetchTrackDetailsAPI(ctx context.Context, trackIDs []string) ([]apiTrack, error) {
	if len(trackIDs) == 0 {
		return nil, nil
	}

	tracks := make([]apiTrack, 0, len(trackIDs))
	for _, trackID := range trackIDs {
		if trackID == "" {
			continue
		}
		endpoint := a.apiBaseURL + "/tracks/" + trackID
		var track apiTrack
		if err := a.getAPIJSON(ctx, endpoint, &track); err != nil {
			if isSpotifyAPIStatus(err, http.StatusNotFound) {
				return nil, fmt.Errorf("%w: %s", errSpotifyTrackNotFound, trackID)
			}
			return nil, err
		}
		tracks = append(tracks, track)
	}
	return tracks, nil
}

func (a *Adapter) fetchAlbumBootstrap(ctx context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalAlbum, error) {
	requestURL := parsed.CanonicalURL
	if parsed.CanonicalURL == "https://open.spotify.com/album/"+parsed.ID && a.webBaseURL != defaultWebBaseURL {
		requestURL = a.webBaseURL + "/album/" + parsed.ID
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build spotify request: %w", err)
	}
	req.Header.Set("User-Agent", "ariadne/0.1 (+https://github.com/xmbshwll/ariadne)")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute spotify request: %w", err)
	}

	body, err := io.ReadAll(resp.Body)
	closeErr := resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("read spotify response: %w", err)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close spotify response body: %w", closeErr)
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, errSpotifyAlbumNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d", errUnexpectedSpotifyStatus, resp.StatusCode)
	}

	payload, err := parseInitialState(body)
	if err != nil {
		return nil, fmt.Errorf("parse spotify initial state: %w", err)
	}

	entityKey := "spotify:album:" + parsed.ID
	album, ok := payload.Entities.Items[entityKey]
	if !ok {
		return nil, fmt.Errorf("%w: %s", errSpotifyAlbumNotFound, entityKey)
	}

	return toCanonicalAlbumBootstrap(parsed, album), nil
}

func (a *Adapter) hydrateAlbumCandidates(ctx context.Context, summaries []apiAlbumSummary) ([]model.CandidateAlbum, error) {
	results := make([]model.CandidateAlbum, 0, len(summaries))
	seen := make(map[string]struct{}, len(summaries))
	var firstErr error
	for _, summary := range summaries {
		if summary.ID == "" {
			continue
		}
		if _, ok := seen[summary.ID]; ok {
			continue
		}
		seen[summary.ID] = struct{}{}

		album, err := a.fetchAlbumAPI(ctx, summary.ID)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("hydrate spotify album %s: %w", summary.ID, err)
			}
			continue
		}
		canonical := toCanonicalAlbumAPI(canonicalAlbumURL(summary.ID), album)
		results = append(results, model.CandidateAlbum{
			CanonicalAlbum: *canonical,
			CandidateID:    canonical.SourceID,
			MatchURL:       canonical.SourceURL,
		})
	}
	if len(results) == 0 && firstErr != nil {
		return nil, firstErr
	}
	return results, nil
}

func (a *Adapter) hydrateSongCandidates(ctx context.Context, items []apiTrackSearchItem) ([]model.CandidateSong, error) {
	results := make([]model.CandidateSong, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	var firstErr error
	for _, item := range items {
		if item.ID == "" {
			continue
		}
		if _, ok := seen[item.ID]; ok {
			continue
		}
		seen[item.ID] = struct{}{}

		track, err := a.fetchTrackAPI(ctx, item.ID)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("hydrate spotify track %s: %w", item.ID, err)
			}
			continue
		}
		canonical := toCanonicalSongAPI(canonicalTrackURL(item.ID), track)
		results = append(results, model.CandidateSong{
			CanonicalSong: *canonical,
			CandidateID:   canonical.SourceID,
			MatchURL:      canonical.SourceURL,
		})
	}
	if len(results) == 0 && firstErr != nil {
		return nil, firstErr
	}
	return results, nil
}
