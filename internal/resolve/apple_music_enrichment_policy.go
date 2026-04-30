package resolve

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/score"
)

const appleMusicCascadeMinimumScore = 100

type appleMusicEnrichmentPolicy struct {
	weights score.Weights
}

func newAppleMusicEnrichmentPolicy(weights score.Weights) appleMusicEnrichmentPolicy {
	return appleMusicEnrichmentPolicy{weights: weights}
}

func (p appleMusicEnrichmentPolicy) apply(
	ctx context.Context,
	targets []TargetAdapter,
	source model.CanonicalAlbum,
	matches map[model.ServiceName]MatchResult,
) error {
	appleMusicTargets := appleMusicTargets(targets)
	if len(appleMusicTargets) == 0 {
		return nil
	}

	enriched, ok := p.enrichedSource(source, matches)
	if !ok {
		return nil
	}

	var matchesMu sync.Mutex
	return resolveTargetsConcurrently(ctx, appleMusicTargets, func(groupCtx context.Context, target TargetAdapter) error {
		newResult, err := p.resolveTarget(groupCtx, target, enriched)
		if err != nil {
			return fmt.Errorf("collect candidates from %s: %w", target.Service(), err)
		}

		matchesMu.Lock()
		existing := matches[target.Service()]
		if p.shouldReplace(existing, newResult) {
			matches[target.Service()] = newResult
		}
		matchesMu.Unlock()
		return nil
	})
}

func (p appleMusicEnrichmentPolicy) enrichedSource(source model.CanonicalAlbum, matches map[model.ServiceName]MatchResult) (model.CanonicalAlbum, bool) {
	enriched := cloneAlbum(source)
	strongMatches := strongIntermediateAlbumMatches(matches)
	for _, match := range strongMatches {
		mergeAlbumIdentifiers(&enriched, match.Candidate)
	}
	return enriched, albumIdentifiersChanged(source, enriched)
}

func (p appleMusicEnrichmentPolicy) resolveTarget(ctx context.Context, target TargetAdapter, source model.CanonicalAlbum) (MatchResult, error) {
	candidates, err := collectAlbumTargetCandidates(ctx, target, source, p.weights)
	if err != nil {
		return MatchResult{}, err
	}
	ranking := score.RankAlbums(source, candidates, p.weights)
	return albumMatchResultFromRanking(target.Service(), ranking), nil
}

func (p appleMusicEnrichmentPolicy) shouldReplace(existing MatchResult, newResult MatchResult) bool {
	return newResult.Best != nil && (existing.Best == nil || newResult.Best.Score > existing.Best.Score)
}

func appleMusicTargets(targets []TargetAdapter) []TargetAdapter {
	filtered := make([]TargetAdapter, 0, len(targets))
	for _, target := range targets {
		if target.Service() != model.ServiceAppleMusic {
			continue
		}
		filtered = append(filtered, target)
	}
	return filtered
}

func strongIntermediateAlbumMatches(matches map[model.ServiceName]MatchResult) []ScoredMatch {
	strongMatches := make([]ScoredMatch, 0, len(matches))
	for service, match := range matches {
		if service == model.ServiceAppleMusic || match.Best == nil || match.Best.Score < appleMusicCascadeMinimumScore {
			continue
		}
		strongMatches = append(strongMatches, *match.Best)
	}
	sort.SliceStable(strongMatches, func(i, j int) bool {
		if strongMatches[i].Score == strongMatches[j].Score {
			leftService := string(strongMatches[i].Candidate.Service)
			rightService := string(strongMatches[j].Candidate.Service)
			if leftService == rightService {
				return strongMatches[i].Candidate.CandidateID < strongMatches[j].Candidate.CandidateID
			}
			return leftService < rightService
		}
		return strongMatches[i].Score > strongMatches[j].Score
	})
	return strongMatches
}

func mergeAlbumIdentifiers(album *model.CanonicalAlbum, candidate model.CandidateAlbum) {
	if album.UPC == "" && candidate.UPC != "" {
		album.UPC = candidate.UPC
	}
	mergeTrackISRCs(album, candidate.Tracks)
}

func mergeTrackISRCs(album *model.CanonicalAlbum, tracks []model.CanonicalTrack) {
	if len(tracks) == 0 {
		return
	}
	if len(album.Tracks) == 0 {
		album.Tracks = make([]model.CanonicalTrack, len(tracks))
	}
	if len(album.Tracks) != len(tracks) {
		return
	}
	for i := range album.Tracks {
		if album.Tracks[i].ISRC != "" || tracks[i].ISRC == "" {
			continue
		}
		album.Tracks[i].ISRC = tracks[i].ISRC
	}
}

func albumIdentifiersChanged(source model.CanonicalAlbum, enriched model.CanonicalAlbum) bool {
	if source.UPC != enriched.UPC {
		return true
	}
	sourceISRCs := collectISRCs(source)
	enrichedISRCs := collectISRCs(enriched)
	if len(sourceISRCs) != len(enrichedISRCs) {
		return true
	}
	for i := range sourceISRCs {
		if !strings.EqualFold(sourceISRCs[i], enrichedISRCs[i]) {
			return true
		}
	}
	return false
}

func cloneAlbum(album model.CanonicalAlbum) model.CanonicalAlbum {
	clone := album
	clone.Artists = append([]string(nil), album.Artists...)
	clone.NormalizedArtists = append([]string(nil), album.NormalizedArtists...)
	clone.EditionHints = append([]string(nil), album.EditionHints...)
	if album.Tracks != nil {
		clone.Tracks = make([]model.CanonicalTrack, 0, len(album.Tracks))
		for _, track := range album.Tracks {
			trackCopy := track
			trackCopy.Artists = append([]string(nil), track.Artists...)
			clone.Tracks = append(clone.Tracks, trackCopy)
		}
	}
	return clone
}

func filterAppleMusicMetadataFallbackCandidates(
	targetService model.ServiceName,
	source model.CanonicalAlbum,
	candidates []model.CandidateAlbum,
	weights score.Weights,
) []model.CandidateAlbum {
	if targetService != model.ServiceAppleMusic || len(candidates) == 0 {
		return candidates
	}

	filtered := make([]model.CandidateAlbum, 0, len(candidates))
	for _, candidate := range candidates {
		ranking := score.RankAlbums(source, []model.CandidateAlbum{candidate}, weights)
		if len(ranking.Ranked) == 0 {
			continue
		}
		ranked := ranking.Ranked[0]
		if ranked.Score <= 0 || !ranked.Evidence.HasTitleOrArtist() {
			continue
		}
		filtered = append(filtered, candidate)
	}
	return filtered
}
