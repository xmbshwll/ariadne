package applemusic

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/normalize"
)

// SearchByMetadata searches Apple Music albums by title and artist metadata via the public search API.
func (a *Adapter) SearchByMetadata(ctx context.Context, album model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
	queries := metadataQueries(album)
	if len(queries) == 0 {
		return nil, nil
	}

	storefront := a.storefrontFor(album.RegionHint)
	results := make([]model.CandidateAlbum, 0, searchLimit)
	seen := make(map[int64]struct{}, searchLimit)
	var firstHydrationErr error

	for _, query := range queries {
		searchURL := fmt.Sprintf("%s/search?term=%s&entity=album&limit=%d&country=%s", a.lookupBaseURL, url.QueryEscape(query), searchLimit, url.QueryEscape(storefront))
		var payload lookupResponse
		if err := a.getJSON(ctx, searchURL, &payload); err != nil {
			return nil, fmt.Errorf("search apple music metadata %q: %w", query, err)
		}

		for _, item := range payload.Results {
			if item.WrapperType != "collection" || item.CollectionType != "Album" {
				continue
			}
			if _, ok := seen[item.CollectionID]; ok {
				continue
			}
			seen[item.CollectionID] = struct{}{}

			canonical, err := a.fetchAlbumByID(ctx, strconv.FormatInt(item.CollectionID, 10), canonicalCollectionURL(item.CollectionViewURL, ""), storefront)
			if err != nil {
				if firstHydrationErr == nil {
					firstHydrationErr = fmt.Errorf("hydrate apple music album %d: %w", item.CollectionID, err)
				}
				continue
			}
			results = append(results, toCandidateAlbum(*canonical))
			if len(results) >= searchLimit {
				return results, nil
			}
		}
	}
	if len(results) == 0 && firstHydrationErr != nil {
		return nil, firstHydrationErr
	}
	return results, nil
}

// SearchSongByMetadata searches Apple Music songs by title and artist metadata via the public search API.
func (a *Adapter) SearchSongByMetadata(ctx context.Context, song model.CanonicalSong) ([]model.CandidateSong, error) {
	queries := songMetadataQueries(song)
	if len(queries) == 0 {
		return nil, nil
	}

	storefront := a.storefrontFor(song.RegionHint)
	results := make([]model.CandidateSong, 0, searchLimit)
	seen := make(map[int64]struct{}, searchLimit)
	var firstHydrationErr error

	for _, query := range queries {
		searchURL := fmt.Sprintf("%s/search?term=%s&entity=%s&limit=%d&country=%s", a.lookupBaseURL, url.QueryEscape(query), entitySong, searchLimit, url.QueryEscape(storefront))
		var payload lookupResponse
		if err := a.getJSON(ctx, searchURL, &payload); err != nil {
			return nil, fmt.Errorf("search apple music song metadata %q: %w", query, err)
		}

		for _, item := range payload.Results {
			if item.WrapperType != wrapperTypeTrack || item.Kind != entitySong || item.TrackID == 0 {
				continue
			}
			if _, ok := seen[item.TrackID]; ok {
				continue
			}
			seen[item.TrackID] = struct{}{}

			canonical, err := a.fetchSongByID(ctx, strconv.FormatInt(item.TrackID, 10), canonicalTrackURL(item.CollectionViewURL, item.TrackID), storefront)
			if err != nil {
				if firstHydrationErr == nil {
					firstHydrationErr = fmt.Errorf("hydrate apple music song %d: %w", item.TrackID, err)
				}
				continue
			}
			results = append(results, toCandidateSong(*canonical))
			if len(results) >= searchLimit {
				return results, nil
			}
		}
	}
	if len(results) == 0 && firstHydrationErr != nil {
		return nil, firstHydrationErr
	}
	return results, nil
}

func metadataQueries(album model.CanonicalAlbum) []string {
	if strings.TrimSpace(album.Title) == "" {
		return nil
	}

	queries := make([]string, 0, 8)
	seen := make(map[string]struct{}, 8)
	appendUnique := func(query string) {
		query = strings.TrimSpace(query)
		if query == "" {
			return
		}
		key := normalize.Text(query)
		if key == "" {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		queries = append(queries, query)
	}

	for _, title := range normalize.SearchTitleVariants(album.Title) {
		for _, artist := range normalize.SearchArtistVariants(album.Artists) {
			appendUnique(strings.TrimSpace(strings.Join([]string{title, artist}, " ")))
		}
		appendUnique(title)
	}
	return queries
}

func songMetadataQueries(song model.CanonicalSong) []string {
	if strings.TrimSpace(song.Title) == "" {
		return nil
	}

	queries := make([]string, 0, 8)
	seen := make(map[string]struct{}, 8)
	appendUnique := func(query string) {
		query = strings.TrimSpace(query)
		if query == "" {
			return
		}
		key := normalize.Text(query)
		if key == "" {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		queries = append(queries, query)
	}

	for _, title := range normalize.SearchTitleVariants(song.Title) {
		for _, artist := range normalize.SearchArtistVariants(song.Artists) {
			appendUnique(strings.TrimSpace(strings.Join([]string{title, artist}, " ")))
		}
		appendUnique(title)
	}
	return queries
}
