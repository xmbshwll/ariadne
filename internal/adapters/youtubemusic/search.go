package youtubemusic

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/xmbshwll/ariadne/internal/adapters/adapterutil"
	"github.com/xmbshwll/ariadne/internal/model"
)

func (a *Adapter) SearchByMetadata(ctx context.Context, album model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
	query := metadataQuery(album)
	if query == "" {
		return nil, nil
	}

	searchURL := fmt.Sprintf("%s/search?q=%s", a.baseURL, url.QueryEscape(query))
	body, err := a.fetchPage(ctx, searchURL)
	if err != nil {
		return nil, fmt.Errorf("fetch youtube music search page: %w", err)
	}

	candidates := extractSearchCandidates(body)
	results, err := adapterutil.CollectCandidatesWithContext(
		ctx,
		candidates,
		searchLimit,
		youTubeMusicSearchCandidateID,
		a.hydrateYouTubeMusicAlbumSearchCandidate,
	)
	if err != nil {
		return nil, fmt.Errorf("collect youtube music candidates: %w", err)
	}
	return results, nil
}

func youTubeMusicSearchCandidateID(candidate searchCandidate) string {
	return candidate.BrowseID
}

func (a *Adapter) hydrateYouTubeMusicAlbumSearchCandidate(ctx context.Context, candidate searchCandidate) (model.CandidateAlbum, error) {
	canonical, err := a.fetchAlbumByBrowseID(ctx, candidate.BrowseID)
	if err != nil {
		return model.CandidateAlbum{}, fmt.Errorf("hydrate youtube music album %s: %w", candidate.BrowseID, err)
	}
	return toCandidateAlbum(*canonical), nil
}

func metadataQuery(album model.CanonicalAlbum) string {
	parts := make([]string, 0, 2)
	if album.Title != "" {
		parts = append(parts, album.Title)
	}
	if len(album.Artists) > 0 {
		parts = append(parts, album.Artists[0])
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}
