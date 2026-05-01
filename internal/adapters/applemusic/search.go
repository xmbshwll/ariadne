package applemusic

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/xmbshwll/ariadne/internal/adapters/adapterutil"
	"github.com/xmbshwll/ariadne/internal/model"
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
			if err := continueAppleMusicSearchAfterQueryError(results, func() error {
				return fmt.Errorf("search apple music metadata %q: %w", query, err)
			}); err != nil {
				return nil, err
			}
			continue
		}

		for _, item := range payload.Results {
			if item.WrapperType != "collection" || item.CollectionType != "Album" || item.CollectionID == 0 {
				continue
			}
			if _, ok := seen[item.CollectionID]; ok {
				continue
			}
			seen[item.CollectionID] = struct{}{}

			canonical, err := a.fetchAlbumByID(ctx, strconv.FormatInt(item.CollectionID, 10), canonicalCollectionURL(item.CollectionViewURL, ""), storefront)
			if err != nil {
				recordFirstAppleMusicHydrationError(&firstHydrationErr, func() error {
					return fmt.Errorf("hydrate apple music album %d: %w", item.CollectionID, err)
				})
				continue
			}
			results = append(results, toCandidateAlbum(*canonical))
			if len(results) >= searchLimit {
				return results, nil
			}
		}
	}
	return finishAppleMusicMetadataSearch(results, firstHydrationErr)
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
			if err := continueAppleMusicSearchAfterQueryError(results, func() error {
				return fmt.Errorf("search apple music song metadata %q: %w", query, err)
			}); err != nil {
				return nil, err
			}
			continue
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
				recordFirstAppleMusicHydrationError(&firstHydrationErr, func() error {
					return fmt.Errorf("hydrate apple music song %d: %w", item.TrackID, err)
				})
				continue
			}
			results = append(results, toCandidateSong(*canonical))
			if len(results) >= searchLimit {
				return results, nil
			}
		}
	}
	return finishAppleMusicMetadataSearch(results, firstHydrationErr)
}

func continueAppleMusicSearchAfterQueryError[T any](results []T, makeErr func() error) error {
	if len(results) == 0 {
		return makeErr()
	}
	return nil
}

func recordFirstAppleMusicHydrationError(firstErr *error, makeErr func() error) {
	if *firstErr != nil {
		return
	}
	*firstErr = makeErr()
}

func finishAppleMusicMetadataSearch[T any](results []T, firstErr error) ([]T, error) {
	if len(results) == 0 && firstErr != nil {
		return nil, firstErr
	}
	return results, nil
}

func metadataQueries(album model.CanonicalAlbum) []string {
	return buildMetadataQueries(album.Title, album.Artists)
}

func songMetadataQueries(song model.CanonicalSong) []string {
	return buildMetadataQueries(song.Title, song.Artists)
}

func buildMetadataQueries(title string, artists []string) []string {
	return adapterutil.MetadataQueries(title, artists)
}
