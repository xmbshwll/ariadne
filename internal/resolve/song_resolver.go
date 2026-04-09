package resolve

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/score"
)

// SongSourceAdapter fetches canonical song metadata from a parsed source URL.
type SongSourceAdapter interface {
	Service() model.ServiceName
	ParseSongURL(raw string) (*model.ParsedAlbumURL, error)
	FetchSong(ctx context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalSong, error)
}

// SongTargetAdapter searches one target service for matching songs.
type SongTargetAdapter interface {
	Service() model.ServiceName
	SearchSongByISRC(ctx context.Context, isrc string) ([]model.CandidateSong, error)
	SearchSongByMetadata(ctx context.Context, song model.CanonicalSong) ([]model.CandidateSong, error)
}

// SongScoredMatch is one scored song candidate exposed by the resolver.
type SongScoredMatch struct {
	URL       string
	Score     int
	Reasons   []string
	Candidate model.CandidateSong
}

// SongMatchResult is the resolver output for one target service.
type SongMatchResult struct {
	Service    model.ServiceName
	Best       *SongScoredMatch
	Alternates []SongScoredMatch
}

// SongResolution contains the source song and ranked target matches collected by the resolver.
type SongResolution struct {
	InputURL string
	Parsed   model.ParsedAlbumURL
	Source   model.CanonicalSong
	Matches  map[model.ServiceName]SongMatchResult
}

// SongResolver coordinates source parsing, source fetching, and layered target search for songs.
type SongResolver struct {
	sources []SongSourceAdapter
	targets []SongTargetAdapter
	weights score.SongWeights
}

// NewSongs creates a song resolver from registered source and target adapters.
func NewSongs(sources []SongSourceAdapter, targets []SongTargetAdapter, weights score.SongWeights) *SongResolver {
	return &SongResolver{
		sources: append([]SongSourceAdapter(nil), sources...),
		targets: append([]SongTargetAdapter(nil), targets...),
		weights: weights,
	}
}

// ResolveSong parses an input song URL, fetches the canonical source song,
// then collects and ranks candidates from every target adapter except the source service.
func (r *SongResolver) ResolveSong(ctx context.Context, inputURL string) (*SongResolution, error) {
	if len(r.sources) == 0 {
		return nil, ErrNoSourceAdapters
	}

	sourceAdapter, parsed, err := r.parseSource(inputURL)
	if err != nil {
		return nil, err
	}

	sourceSong, err := sourceAdapter.FetchSong(ctx, *parsed)
	if err != nil {
		return nil, fmt.Errorf("fetch source song with %s: %w", sourceAdapter.Service(), err)
	}

	targets := r.targetSearchesFor(sourceSong.Service)
	resolution := &SongResolution{
		InputURL: inputURL,
		Parsed:   *parsed,
		Source:   *sourceSong,
		Matches:  make(map[model.ServiceName]SongMatchResult, len(targets)),
	}

	group, groupCtx := errgroup.WithContext(ctx)
	var matchesMu sync.Mutex

	for _, target := range targets {
		group.Go(func() error {
			candidates, err := r.collectCandidates(groupCtx, target, *sourceSong)
			if err != nil {
				return fmt.Errorf("collect song candidates from %s: %w", target.Service(), err)
			}
			ranking := score.RankSongs(*sourceSong, candidates, r.weights)

			matchesMu.Lock()
			resolution.Matches[target.Service()] = songMatchResultFromRanking(target.Service(), ranking)
			matchesMu.Unlock()
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return nil, fmt.Errorf("resolve song target searches: %w", err)
	}
	return resolution, nil
}

func (r *SongResolver) targetSearchesFor(sourceService model.ServiceName) []SongTargetAdapter {
	targets := make([]SongTargetAdapter, 0, len(r.targets))
	for _, target := range r.targets {
		if target.Service() == sourceService {
			continue
		}
		targets = append(targets, target)
	}
	return targets
}

func (r *SongResolver) parseSource(inputURL string) (SongSourceAdapter, *model.ParsedAlbumURL, error) {
	for _, source := range r.sources {
		parsed, err := source.ParseSongURL(inputURL)
		if err != nil || parsed == nil {
			continue
		}
		return source, parsed, nil
	}
	return nil, nil, fmt.Errorf("%w: %s", ErrUnsupportedURL, inputURL)
}

func (r *SongResolver) collectCandidates(ctx context.Context, target SongTargetAdapter, source model.CanonicalSong) ([]model.CandidateSong, error) {
	combined := make([]model.CandidateSong, 0)
	seen := make(map[string]struct{})

	appendUnique := func(candidates []model.CandidateSong) {
		for _, candidate := range candidates {
			key := songCandidateKey(candidate)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			combined = append(combined, candidate)
		}
	}

	if source.ISRC != "" {
		candidates, err := target.SearchSongByISRC(ctx, source.ISRC)
		if err != nil {
			return nil, err
		}
		appendUnique(candidates)
	}

	metadataCandidates, err := target.SearchSongByMetadata(ctx, source)
	if err != nil {
		return nil, err
	}
	appendUnique(metadataCandidates)

	return combined, nil
}

func songCandidateKey(candidate model.CandidateSong) string {
	if candidate.CandidateID != "" {
		return string(candidate.Service) + ":id:" + candidate.CandidateID
	}
	return string(candidate.Service) + ":url:" + candidate.MatchURL
}

func songMatchResultFromRanking(service model.ServiceName, ranking score.SongRanking) SongMatchResult {
	result := SongMatchResult{Service: service, Alternates: make([]SongScoredMatch, 0)}
	if ranking.Best == nil {
		return result
	}

	best := toSongScoredMatch(*ranking.Best)
	result.Best = &best
	for _, ranked := range ranking.Ranked[1:] {
		result.Alternates = append(result.Alternates, toSongScoredMatch(ranked))
	}
	return result
}

func toSongScoredMatch(ranked score.RankedSongCandidate) SongScoredMatch {
	return SongScoredMatch{
		URL:       ranked.Candidate.MatchURL,
		Score:     ranked.Score,
		Reasons:   append([]string(nil), ranked.Reasons...),
		Candidate: ranked.Candidate,
	}
}
