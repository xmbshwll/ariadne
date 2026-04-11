package tidal

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/xmbshwll/ariadne/internal/model"
)

func (a *Adapter) FetchAlbum(ctx context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalAlbum, error) {
	if parsed.Service != model.ServiceTIDAL {
		return nil, fmt.Errorf("%w: %s", errUnexpectedTIDALService, parsed.Service)
	}
	return a.fetchAlbumByID(ctx, parsed.ID, parsed.CanonicalURL, parsed.RegionHint)
}

func (a *Adapter) SearchByUPC(ctx context.Context, upc string) ([]model.CandidateAlbum, error) {
	upc = strings.TrimSpace(upc)
	if upc == "" {
		return nil, nil
	}
	if !a.hasCredentials() {
		return nil, ErrCredentialsNotConfigured
	}

	endpoint := fmt.Sprintf("%s/albums?countryCode=%s&filter[barcodeId]=%s", a.apiBaseURL, url.QueryEscape(a.defaultCountryCode), url.QueryEscape(upc))
	var document apiDocument
	if err := a.getAPIJSON(ctx, endpoint, &document); err != nil {
		return nil, fmt.Errorf("tidal search by upc: %w", err)
	}
	resources := documentData(document)
	return a.hydrateAlbumCandidates(ctx, resourceIDs(resources), "", func(albumID string) string {
		return fmt.Sprintf("hydrate tidal album %s from upc", albumID)
	})
}

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

	results := make([]model.CandidateAlbum, 0, len(trimmedISRCs))
	seen := make(map[string]struct{}, len(trimmedISRCs))
	for _, isrc := range trimmedISRCs {
		endpoint := fmt.Sprintf("%s/tracks?countryCode=%s&filter[isrc]=%s&include=%s", a.apiBaseURL, url.QueryEscape(a.defaultCountryCode), url.QueryEscape(isrc), url.QueryEscape("albums"))
		var document apiDocument
		if err := a.getAPIJSON(ctx, endpoint, &document); err != nil {
			if err := continueTIDALSearchAfterQueryError(results, func() error {
				return fmt.Errorf("tidal search by isrc %s: %w", isrc, err)
			}); err != nil {
				return nil, err
			}
			continue
		}
		albumIDs := uniqueStrings(albumIDsFromTrackDocument(document), seen)
		hydrated, err := a.hydrateAlbumCandidates(ctx, albumIDs, "", func(albumID string) string {
			return fmt.Sprintf("hydrate tidal album %s from isrc %s", albumID, isrc)
		})
		if err != nil {
			if len(results) > 0 {
				continue
			}
			return nil, err
		}
		results = append(results, hydrated...)
		if len(results) >= searchLimit {
			return results[:searchLimit], nil
		}
	}
	return results, nil
}

func (a *Adapter) SearchByMetadata(ctx context.Context, album model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
	query := metadataQuery(album)
	if query == "" {
		return nil, nil
	}
	if !a.hasCredentials() {
		return nil, ErrCredentialsNotConfigured
	}
	countryCode := a.countryCodeFor(album.RegionHint)
	endpoint := fmt.Sprintf("%s/searchResults/%s/relationships/albums?countryCode=%s", a.apiBaseURL, url.PathEscape(query), url.QueryEscape(countryCode))
	var document apiDocument
	if err := a.getAPIJSON(ctx, endpoint, &document); err != nil {
		return nil, fmt.Errorf("tidal search by metadata: %w", err)
	}
	resources := documentData(document)
	return a.hydrateAlbumCandidates(ctx, resourceIDs(resources), album.RegionHint, func(albumID string) string {
		return fmt.Sprintf("hydrate tidal album %s from metadata", albumID)
	})
}

func (a *Adapter) FetchSong(ctx context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalSong, error) {
	if parsed.Service != model.ServiceTIDAL {
		return nil, fmt.Errorf("%w: %s", errUnexpectedTIDALService, parsed.Service)
	}
	return a.fetchSongByID(ctx, parsed.ID, parsed.CanonicalURL, parsed.RegionHint)
}

func (a *Adapter) SearchSongByISRC(ctx context.Context, isrc string) ([]model.CandidateSong, error) {
	isrc = strings.TrimSpace(isrc)
	if isrc == "" {
		return nil, nil
	}
	if !a.hasCredentials() {
		return nil, ErrCredentialsNotConfigured
	}

	endpoint := fmt.Sprintf("%s/tracks?countryCode=%s&filter[isrc]=%s", a.apiBaseURL, url.QueryEscape(a.defaultCountryCode), url.QueryEscape(isrc))
	var document apiDocument
	if err := a.getAPIJSON(ctx, endpoint, &document); err != nil {
		return nil, fmt.Errorf("tidal song search by isrc %s: %w", isrc, err)
	}
	resources := documentData(document)
	return a.hydrateSongCandidates(ctx, resourceIDs(resources), "", func(songID string) string {
		return fmt.Sprintf("hydrate tidal song %s from isrc", songID)
	})
}

func (a *Adapter) SearchSongByMetadata(ctx context.Context, song model.CanonicalSong) ([]model.CandidateSong, error) {
	query := songMetadataQuery(song)
	if query == "" {
		return nil, nil
	}
	if !a.hasCredentials() {
		return nil, ErrCredentialsNotConfigured
	}
	countryCode := a.countryCodeFor(song.RegionHint)
	endpoint := fmt.Sprintf("%s/searchResults/%s/relationships/tracks?countryCode=%s", a.apiBaseURL, url.PathEscape(query), url.QueryEscape(countryCode))
	var document apiDocument
	if err := a.getAPIJSON(ctx, endpoint, &document); err != nil {
		return nil, fmt.Errorf("tidal song search by metadata: %w", err)
	}
	resources := documentData(document)
	return a.hydrateSongCandidates(ctx, resourceIDs(resources), song.RegionHint, func(songID string) string {
		return fmt.Sprintf("hydrate tidal song %s from metadata", songID)
	})
}

func continueTIDALSearchAfterQueryError[T any](results []T, makeErr func() error) error {
	if len(results) == 0 {
		return makeErr()
	}
	return nil
}

func (a *Adapter) hydrateAlbumCandidates(ctx context.Context, albumIDs []string, regionHint string, errorMessage func(string) string) ([]model.CandidateAlbum, error) {
	return hydrateTIDALCandidates(albumIDs, func(albumID string) (model.CandidateAlbum, error) {
		canonical, err := a.fetchAlbumByID(ctx, albumID, canonicalAlbumURL(albumID), regionHint)
		if err != nil {
			return model.CandidateAlbum{}, fmt.Errorf("%s: %w", errorMessage(albumID), err)
		}
		return toCandidateAlbum(*canonical), nil
	})
}

func (a *Adapter) hydrateSongCandidates(ctx context.Context, songIDs []string, regionHint string, errorMessage func(string) string) ([]model.CandidateSong, error) {
	return hydrateTIDALCandidates(songIDs, func(songID string) (model.CandidateSong, error) {
		canonical, err := a.fetchSongByID(ctx, songID, canonicalTrackURL(songID), regionHint)
		if err != nil {
			return model.CandidateSong{}, fmt.Errorf("%s: %w", errorMessage(songID), err)
		}
		return toCandidateSong(*canonical), nil
	})
}

func hydrateTIDALCandidates[T any](ids []string, fetch func(string) (T, error)) ([]T, error) {
	results := make([]T, 0, min(len(ids), searchLimit))
	var firstErr error
	for _, id := range ids {
		candidate, err := fetch(id)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		results = append(results, candidate)
		if len(results) >= searchLimit {
			return results, nil
		}
	}
	if len(results) == 0 && firstErr != nil {
		return nil, firstErr
	}
	return results, nil
}

func resourceIDs(resources []apiResource) []string {
	ids := make([]string, 0, len(resources))
	seen := make(map[string]struct{}, len(resources))
	for _, resource := range resources {
		if resource.ID == "" {
			continue
		}
		if _, ok := seen[resource.ID]; ok {
			continue
		}
		seen[resource.ID] = struct{}{}
		ids = append(ids, resource.ID)
	}
	return ids
}

func uniqueStrings(values []string, seen map[string]struct{}) []string {
	unique := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	return unique
}

func (a *Adapter) fetchAlbumByID(ctx context.Context, albumID string, canonicalURL string, regionHint string) (*model.CanonicalAlbum, error) {
	var document apiDocument
	countryCode := a.countryCodeFor(regionHint)
	endpoint := fmt.Sprintf("%s/albums/%s?countryCode=%s&include=%s", a.apiBaseURL, url.PathEscape(albumID), url.QueryEscape(countryCode), url.QueryEscape("artists,items,coverArt"))
	if err := a.getAPIJSON(ctx, endpoint, &document); err != nil {
		return nil, err
	}
	resource := firstDataResource(document)
	if resource == nil {
		return nil, fmt.Errorf("%w: %s", errTIDALAlbumNotFound, albumID)
	}
	return toCanonicalAlbum(*resource, document.Included, canonicalURL, regionHint), nil
}

func (a *Adapter) fetchSongByID(ctx context.Context, trackID string, canonicalURL string, regionHint string) (*model.CanonicalSong, error) {
	var document apiDocument
	countryCode := a.countryCodeFor(regionHint)
	endpoint := fmt.Sprintf("%s/tracks/%s?countryCode=%s&include=%s", a.apiBaseURL, url.PathEscape(trackID), url.QueryEscape(countryCode), url.QueryEscape("artists,albums,coverArt"))
	if err := a.getAPIJSON(ctx, endpoint, &document); err != nil {
		return nil, err
	}
	resource := firstDataResource(document)
	if resource == nil {
		return nil, fmt.Errorf("%w: %s", errTIDALTrackNotFound, trackID)
	}
	return toCanonicalSong(*resource, document.Included, canonicalURL, regionHint), nil
}
