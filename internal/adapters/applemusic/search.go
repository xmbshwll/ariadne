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
	targetSearch := adapterutil.MetadataQuerySearch[lookupItem, model.CandidateAlbum]{
		Queries: queries,
		Limit:   searchLimit,
		Search: func(ctx context.Context, query string) ([]lookupItem, error) {
			searchURL := a.metadataSearchURL(query, "album", storefront)
			var payload lookupResponse
			if err := a.getJSON(ctx, searchURL, &payload); err != nil {
				return nil, fmt.Errorf("search apple music metadata %q: %w", query, err)
			}
			return payload.Results, nil
		},
		ItemID: appleMusicAlbumSearchItemID,
		BuildCandidate: func(ctx context.Context, item lookupItem) (model.CandidateAlbum, error) {
			canonical, err := a.fetchAlbumByID(
				ctx,
				strconv.FormatInt(item.CollectionID, 10),
				canonicalCollectionURL(item.CollectionViewURL, ""),
				storefront,
			)
			if err != nil {
				return model.CandidateAlbum{}, fmt.Errorf("hydrate apple music album %d: %w", item.CollectionID, err)
			}
			return toCandidateAlbum(*canonical), nil
		},
		ContinueAfterSearchError: continueAppleMusicMetadataSearchAfterError,
	}
	results, err := targetSearch.Collect(ctx)
	if err != nil {
		return nil, fmt.Errorf("collect apple music metadata candidates: %w", err)
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
	targetSearch := adapterutil.MetadataQuerySearch[lookupItem, model.CandidateSong]{
		Queries: queries,
		Limit:   searchLimit,
		Search: func(ctx context.Context, query string) ([]lookupItem, error) {
			searchURL := a.metadataSearchURL(query, entitySong, storefront)
			var payload lookupResponse
			if err := a.getJSON(ctx, searchURL, &payload); err != nil {
				return nil, fmt.Errorf("search apple music song metadata %q: %w", query, err)
			}
			return payload.Results, nil
		},
		ItemID: appleMusicSongSearchItemID,
		BuildCandidate: func(ctx context.Context, item lookupItem) (model.CandidateSong, error) {
			canonical, err := a.fetchSongByID(
				ctx,
				strconv.FormatInt(item.TrackID, 10),
				canonicalTrackURL(item.CollectionViewURL, item.TrackID),
				storefront,
			)
			if err != nil {
				return model.CandidateSong{}, fmt.Errorf("hydrate apple music song %d: %w", item.TrackID, err)
			}
			return toCandidateSong(*canonical), nil
		},
		ContinueAfterSearchError: continueAppleMusicMetadataSearchAfterError,
	}
	results, err := targetSearch.Collect(ctx)
	if err != nil {
		return nil, fmt.Errorf("collect apple music song metadata candidates: %w", err)
	}
	return results, nil
}

func (a *Adapter) metadataSearchURL(query string, entity string, storefront string) string {
	return fmt.Sprintf(
		"%s/search?term=%s&entity=%s&limit=%d&country=%s",
		a.lookupBaseURL,
		url.QueryEscape(query),
		entity,
		searchLimit,
		url.QueryEscape(storefront),
	)
}

func appleMusicAlbumSearchItemID(item lookupItem) string {
	if item.WrapperType != "collection" || item.CollectionType != "Album" || item.CollectionID == 0 {
		return ""
	}
	return strconv.FormatInt(item.CollectionID, 10)
}

func appleMusicSongSearchItemID(item lookupItem) string {
	if item.WrapperType != wrapperTypeTrack || item.Kind != entitySong || item.TrackID == 0 {
		return ""
	}
	return strconv.FormatInt(item.TrackID, 10)
}

func continueAppleMusicMetadataSearchAfterError(collected int) bool {
	return collected > 0
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
