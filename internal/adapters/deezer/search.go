package deezer

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/xmbshwll/ariadne/internal/adapters/adapterutil"
	"github.com/xmbshwll/ariadne/internal/model"
)

// SearchByUPC resolves a Deezer album directly from a UPC when Deezer exposes the lookup path.
func (a *Adapter) SearchByUPC(ctx context.Context, upc string) ([]model.CandidateAlbum, error) {
	upc = strings.TrimSpace(upc)
	if upc == "" {
		return nil, nil
	}

	canonical, err := a.fetchAlbumByLookup(ctx, a.baseURL+"/album/upc:"+url.PathEscape(upc))
	if err != nil {
		return nil, fmt.Errorf("search deezer by upc %s: %w", upc, err)
	}

	return []model.CandidateAlbum{toCandidateAlbum(*canonical)}, nil
}

// SearchByISRC resolves Deezer albums from one or more track ISRCs.
func (a *Adapter) SearchByISRC(ctx context.Context, isrcs []string) ([]model.CandidateAlbum, error) {
	isrcs = adapterutil.TrimmedNonEmptyStrings(isrcs)
	if len(isrcs) == 0 {
		return nil, nil
	}

	results := make([]model.CandidateAlbum, 0, len(isrcs))
	seenAlbumIDs := make(map[int]struct{}, len(isrcs))
	var firstErr error

	for _, isrc := range isrcs {
		var track trackLookupResponse
		endpoint := a.baseURL + "/track/isrc:" + url.PathEscape(isrc)
		if err := a.getJSON(ctx, endpoint, &track); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("search deezer by isrc %s: %w", isrc, err)
			}
			continue
		}
		if track.Album.ID == 0 {
			continue
		}
		albumID := track.Album.ID
		if _, ok := seenAlbumIDs[albumID]; ok {
			continue
		}
		seenAlbumIDs[albumID] = struct{}{}

		candidate, err := a.hydrateAlbumCandidate(ctx, albumID)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("hydrate deezer album %d from isrc %s: %w", albumID, isrc, err)
			}
			continue
		}
		results = append(results, candidate)
		if len(results) >= identifierSearchLimit {
			break
		}
	}
	if len(results) == 0 && firstErr != nil {
		return nil, firstErr
	}

	return results, nil
}

// SearchByMetadata searches Deezer albums using album title and artist metadata.
func (a *Adapter) SearchByMetadata(ctx context.Context, album model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
	query := metadataQuery(album)
	if query == "" {
		return nil, nil
	}

	var searchResults albumSearchResponse
	endpoint := a.baseURL + "/search/album?q=" + url.QueryEscape(query)
	if err := a.getJSON(ctx, endpoint, &searchResults); err != nil {
		return nil, fmt.Errorf("search deezer by metadata %q: %w", query, err)
	}

	results, err := adapterutil.CollectCandidates(
		searchResults.Data,
		metadataSearchLimit,
		deezerAlbumSearchCandidateID,
		func(candidate albumResponse) (model.CandidateAlbum, error) {
			return a.hydrateDeezerAlbumSearchCandidate(ctx, candidate)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("collect deezer album candidates: %w", err)
	}
	return results, nil
}

// SearchSongByISRC resolves Deezer songs from an ISRC.
func (a *Adapter) SearchSongByISRC(ctx context.Context, isrc string) ([]model.CandidateSong, error) {
	isrc = strings.TrimSpace(isrc)
	if isrc == "" {
		return nil, nil
	}

	track, err := a.fetchTrackLookup(ctx, a.baseURL+"/track/isrc:"+url.PathEscape(isrc))
	if err != nil {
		if errors.Is(err, errDeezerTrackNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("search deezer song by isrc %s: %w", isrc, err)
	}
	return []model.CandidateSong{toCandidateSong(*a.toCanonicalSong(*track))}, nil
}

// SearchSongByMetadata searches Deezer tracks using song title and artist metadata.
func (a *Adapter) SearchSongByMetadata(ctx context.Context, song model.CanonicalSong) ([]model.CandidateSong, error) {
	query := songMetadataQuery(song)
	if query == "" {
		return nil, nil
	}

	var searchResults tracksResponse
	endpoint := a.baseURL + "/search/track?q=" + url.QueryEscape(query)
	if err := a.getJSON(ctx, endpoint, &searchResults); err != nil {
		return nil, fmt.Errorf("search deezer song by metadata %q: %w", query, err)
	}

	results, err := adapterutil.CollectCandidates(
		searchResults.Data,
		metadataSearchLimit,
		deezerSongSearchCandidateID,
		func(candidate trackResponse) (model.CandidateSong, error) {
			return a.hydrateDeezerSongSearchCandidate(ctx, candidate)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("collect deezer song candidates: %w", err)
	}
	return results, nil
}

func deezerAlbumSearchCandidateID(candidate albumResponse) string {
	return strconv.Itoa(candidate.ID)
}

func deezerSongSearchCandidateID(candidate trackResponse) string {
	return strconv.Itoa(candidate.ID)
}

func (a *Adapter) hydrateDeezerAlbumSearchCandidate(ctx context.Context, candidate albumResponse) (model.CandidateAlbum, error) {
	hydrated, err := a.hydrateAlbumCandidate(ctx, candidate.ID)
	if err != nil {
		return model.CandidateAlbum{}, fmt.Errorf("hydrate deezer candidate %d: %w", candidate.ID, err)
	}
	return hydrated, nil
}

func (a *Adapter) hydrateDeezerSongSearchCandidate(ctx context.Context, candidate trackResponse) (model.CandidateSong, error) {
	hydrated, err := a.hydrateSongCandidate(ctx, candidate.ID)
	if err != nil {
		return model.CandidateSong{}, fmt.Errorf("hydrate deezer song candidate %d: %w", candidate.ID, err)
	}
	return hydrated, nil
}

func (a *Adapter) hydrateAlbumCandidate(ctx context.Context, albumID int) (model.CandidateAlbum, error) {
	canonical, err := a.fetchAlbumByID(ctx, strconv.Itoa(albumID))
	if err != nil {
		return model.CandidateAlbum{}, err
	}
	return toCandidateAlbum(*canonical), nil
}

func (a *Adapter) hydrateSongCandidate(ctx context.Context, trackID int) (model.CandidateSong, error) {
	canonical, err := a.fetchSongByID(ctx, strconv.Itoa(trackID))
	if err != nil {
		return model.CandidateSong{}, err
	}
	return toCandidateSong(*canonical), nil
}
