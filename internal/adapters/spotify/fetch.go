package spotify

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/xmbshwll/ariadne/internal/adapters/adapterutil"
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
		if !errors.Is(err, ErrSpotifyAlbumNotFound) {
			return nil, err
		}
	}

	return a.FetchAlbumBootstrap(ctx, parsed)
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

	endpoint := a.searchEndpoint("upc:"+upc, "album", searchLimit)
	var response APIAlbumSearchResponse
	if err := a.getAPIJSON(ctx, endpoint, &response); err != nil {
		return nil, fmt.Errorf("spotify search by upc: %w", err)
	}
	return a.hydrateAlbumCandidates(ctx, response.Albums.Items)
}

// SearchByISRC searches Spotify track results by ISRC, then hydrates the owning albums.
func (a *Adapter) SearchByISRC(ctx context.Context, isrcs []string) ([]model.CandidateAlbum, error) {
	isrcs = adapterutil.TrimmedNonEmptyStrings(isrcs)
	if len(isrcs) == 0 {
		return nil, nil
	}
	if !a.hasCredentials() {
		return nil, ErrCredentialsNotConfigured
	}

	albumIDs := make([]string, 0, len(isrcs))
	seen := make(map[string]struct{}, len(isrcs))
	var firstErr error
	for _, isrc := range isrcs {
		endpoint := a.searchEndpoint("isrc:"+isrc, "track", 1)
		var response APITrackSearchResponse
		if err := a.getAPIJSON(ctx, endpoint, &response); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("spotify search by isrc %s: %w", isrc, err)
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
	results, err := a.hydrateAlbumCandidates(ctx, albumIDsToSummaries(albumIDs))
	if len(results) == 0 && firstErr != nil {
		return nil, firstErr
	}
	return results, err
}

// SearchByMetadata searches Spotify albums by title and artist metadata.
func (a *Adapter) SearchByMetadata(ctx context.Context, album model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
	queries := MetadataQueries(album)
	if len(queries) == 0 {
		return nil, nil
	}
	if !a.hasCredentials() {
		return nil, ErrCredentialsNotConfigured
	}

	targetSearch := adapterutil.MetadataQuerySearch[APIAlbumSummary, APIAlbumSummary]{
		Queries: queries,
		Limit:   searchLimit,
		Search: func(ctx context.Context, query string) ([]APIAlbumSummary, error) {
			endpoint := a.searchEndpoint(query, "album", searchLimit)
			var response APIAlbumSearchResponse
			if err := a.getAPIJSON(ctx, endpoint, &response); err != nil {
				return nil, fmt.Errorf("spotify search by metadata %q: %w", query, err)
			}
			return response.Albums.Items, nil
		},
		ItemID: func(item APIAlbumSummary) string { return item.ID },
		BuildCandidate: func(_ context.Context, item APIAlbumSummary) (APIAlbumSummary, error) {
			return item, nil
		},
	}
	items, err := targetSearch.Collect(ctx)
	if err != nil {
		return nil, fmt.Errorf("collect spotify metadata results: %w", err)
	}
	return a.hydrateAlbumCandidates(ctx, items)
}

// FetchSong loads a Spotify track via the Web API.
func (a *Adapter) FetchSong(ctx context.Context, parsed model.ParsedURL) (*model.CanonicalSong, error) {
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

	endpoint := a.searchEndpoint("isrc:"+strings.TrimSpace(isrc), "track", searchLimit)
	var response APITrackSearchResponse
	if err := a.getAPIJSON(ctx, endpoint, &response); err != nil {
		return nil, fmt.Errorf("spotify song search by isrc %s: %w", isrc, err)
	}
	return a.hydrateSongCandidates(ctx, response.Tracks.Items)
}

// SearchSongByMetadata searches Spotify tracks by title and artist metadata.
func (a *Adapter) SearchSongByMetadata(ctx context.Context, song model.CanonicalSong) ([]model.CandidateSong, error) {
	queries := SongMetadataQueries(song)
	if len(queries) == 0 {
		return nil, nil
	}
	if !a.hasCredentials() {
		return nil, ErrCredentialsNotConfigured
	}

	targetSearch := adapterutil.MetadataQuerySearch[APITrackSearchItem, APITrackSearchItem]{
		Queries: queries,
		Limit:   searchLimit,
		Search: func(ctx context.Context, query string) ([]APITrackSearchItem, error) {
			endpoint := a.searchEndpoint(query, "track", searchLimit)
			var response APITrackSearchResponse
			if err := a.getAPIJSON(ctx, endpoint, &response); err != nil {
				return nil, fmt.Errorf("spotify song search by metadata %q: %w", query, err)
			}
			return response.Tracks.Items, nil
		},
		ItemID: func(item APITrackSearchItem) string { return item.ID },
		BuildCandidate: func(_ context.Context, item APITrackSearchItem) (APITrackSearchItem, error) {
			return item, nil
		},
	}
	items, err := targetSearch.Collect(ctx)
	if err != nil {
		return nil, fmt.Errorf("collect spotify song metadata results: %w", err)
	}
	return a.hydrateSongCandidates(ctx, items)
}

func (a *Adapter) searchEndpoint(query string, entityType string, limit int) string {
	return fmt.Sprintf(
		"%s/search?q=%s&type=%s&limit=%d",
		a.apiBaseURL,
		url.QueryEscape(query),
		entityType,
		limit,
	)
}

func (a *Adapter) fetchAlbumAPI(ctx context.Context, albumID string) (*APIAlbumResponse, error) {
	var album APIAlbumResponse
	endpoint := a.apiBaseURL + "/albums/" + albumID
	if err := a.getAPIJSON(ctx, endpoint, &album); err != nil {
		if isSpotifyAPIStatus(err, http.StatusNotFound) {
			return nil, fmt.Errorf("%w: %s", ErrSpotifyAlbumNotFound, albumID)
		}
		return nil, fmt.Errorf("spotify fetch album api %s: %w", albumID, err)
	}
	if err := a.hydrateAlbumTrackDetails(ctx, &album); err != nil {
		return nil, fmt.Errorf("spotify hydrate track details %s: %w", albumID, err)
	}
	return &album, nil
}

func (a *Adapter) hydrateAlbumTrackDetails(ctx context.Context, album *APIAlbumResponse) error {
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
	byID := make(map[string]APITrack, len(trackDetails))
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

const spotifyTrackFetchParallelism = 8

func (a *Adapter) fetchTrackAPI(ctx context.Context, trackID string) (*APITrack, error) {
	trackID = strings.TrimSpace(trackID)
	if trackID == "" {
		return nil, fmt.Errorf("%w: %s", errSpotifyTrackNotFound, trackID)
	}

	var track APITrack
	endpoint := a.apiBaseURL + "/tracks/" + trackID
	if err := a.getAPIJSON(ctx, endpoint, &track); err != nil {
		if isSpotifyAPIStatus(err, http.StatusNotFound) {
			return nil, fmt.Errorf("%w: %s", errSpotifyTrackNotFound, trackID)
		}
		return nil, fmt.Errorf("spotify fetch track api %s: %w", trackID, err)
	}
	return &track, nil
}

func (a *Adapter) fetchTrackDetailsAPI(ctx context.Context, trackIDs []string) ([]APITrack, error) {
	if len(trackIDs) == 0 {
		return nil, nil
	}

	ids := make([]string, 0, len(trackIDs))
	for _, trackID := range trackIDs {
		trackID = strings.TrimSpace(trackID)
		if trackID == "" {
			continue
		}
		ids = append(ids, trackID)
	}
	if len(ids) == 0 {
		return nil, nil
	}

	results := make([]*APITrack, len(ids))
	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(spotifyTrackFetchParallelism)

	multiTrackFetch := len(ids) > 1
	for i, trackID := range ids {
		group.Go(func() error {
			track, err := a.fetchTrackAPI(groupCtx, trackID)
			if err != nil {
				if multiTrackFetch && shouldSkipSpotifyTrackDetailError(err) {
					return nil
				}
				return err
			}
			results[i] = track
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return nil, fmt.Errorf("fetch spotify track details: %w", err)
	}

	tracks := make([]APITrack, 0, len(results))
	for _, track := range results {
		if track == nil || strings.TrimSpace(track.ID) == "" {
			continue
		}
		tracks = append(tracks, *track)
	}
	return tracks, nil
}

func shouldSkipSpotifyTrackDetailError(err error) bool {
	return errors.Is(err, errSpotifyTrackNotFound) || shouldRetrySpotifyAPIError(err)
}

func (a *Adapter) FetchAlbumBootstrap(ctx context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalAlbum, error) {
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
		return nil, ErrSpotifyAlbumNotFound
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
		return nil, fmt.Errorf("%w: %s", ErrSpotifyAlbumNotFound, entityKey)
	}

	return toCanonicalAlbumBootstrap(parsed, album), nil
}

func (a *Adapter) hydrateAlbumCandidates(
	ctx context.Context,
	summaries []APIAlbumSummary,
) ([]model.CandidateAlbum, error) {
	return hydrateSpotifyCandidates(
		ctx,
		summaries,
		func(summary APIAlbumSummary) string { return summary.ID },
		func(ctx context.Context, summary APIAlbumSummary) (model.CandidateAlbum, error) {
			album, err := a.fetchAlbumAPI(ctx, summary.ID)
			if err != nil {
				return model.CandidateAlbum{}, fmt.Errorf("hydrate spotify album %s: %w", summary.ID, err)
			}
			canonical := toCanonicalAlbumAPI(canonicalAlbumURL(summary.ID), album)
			return model.CandidateAlbum{
				CanonicalAlbum: *canonical,
				CandidateID:    canonical.SourceID,
				MatchURL:       canonical.SourceURL,
			}, nil
		},
	)
}

func (a *Adapter) hydrateSongCandidates(
	ctx context.Context,
	items []APITrackSearchItem,
) ([]model.CandidateSong, error) {
	return hydrateSpotifyCandidates(
		ctx,
		items,
		func(item APITrackSearchItem) string { return item.ID },
		func(ctx context.Context, item APITrackSearchItem) (model.CandidateSong, error) {
			track, err := a.fetchTrackAPI(ctx, item.ID)
			if err != nil {
				return model.CandidateSong{}, fmt.Errorf("hydrate spotify track %s: %w", item.ID, err)
			}
			canonical := toCanonicalSongAPI(canonicalTrackURL(item.ID), track)
			return model.CandidateSong{
				CanonicalSong: *canonical,
				CandidateID:   canonical.SourceID,
				MatchURL:      canonical.SourceURL,
			}, nil
		},
	)
}

func hydrateSpotifyCandidates[Input any, Candidate any](
	ctx context.Context,
	items []Input,
	itemID func(Input) string,
	fetch func(context.Context, Input) (Candidate, error),
) ([]Candidate, error) {
	//nolint:wrapcheck // Preserve per-item fetch errors from the shared candidate collector.
	return adapterutil.CollectCandidates(ctx, items, searchLimit, itemID, fetch)
}
