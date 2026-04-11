package resolve

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/score"
)

// SongSourceAdapter fetches canonical song metadata from a parsed source URL.
type SongSourceAdapter interface {
	Service() model.ServiceName
	ParseSongURL(raw string) (*model.ParsedURL, error)
	FetchSong(ctx context.Context, parsed model.ParsedURL) (*model.CanonicalSong, error)
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
	Parsed   model.ParsedURL
	Source   model.CanonicalSong
	Matches  map[model.ServiceName]SongMatchResult
}

var errNilSourceSong = errors.New("fetch source song returned nil")

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

	sourceSong, err := r.fetchSourceSong(ctx, sourceAdapter, *parsed)
	if err != nil {
		return nil, err
	}
	source := *sourceSong

	targets := excludeTargetService(r.targets, source.Service)
	resolution := &SongResolution{
		InputURL: inputURL,
		Parsed:   *parsed,
		Source:   source,
		Matches:  make(map[model.ServiceName]SongMatchResult, len(targets)),
	}

	var matchesMu sync.Mutex
	if err := resolveTargetsConcurrently(ctx, targets, func(groupCtx context.Context, target SongTargetAdapter) error {
		candidates, err := r.collectCandidates(groupCtx, target, source)
		if err != nil {
			return fmt.Errorf("collect song candidates from %s: %w", target.Service(), err)
		}
		ranking := score.RankSongs(source, candidates, r.weights)

		matchesMu.Lock()
		resolution.Matches[target.Service()] = songMatchResultFromRanking(target.Service(), ranking)
		matchesMu.Unlock()
		return nil
	}); err != nil {
		return nil, fmt.Errorf("resolve song target searches: %w", err)
	}
	return resolution, nil
}

func (r *SongResolver) fetchSourceSong(ctx context.Context, sourceAdapter SongSourceAdapter, parsed model.ParsedURL) (*model.CanonicalSong, error) {
	sourceSong, err := sourceAdapter.FetchSong(ctx, parsed)
	if err != nil {
		return nil, fmt.Errorf("fetch source song with %s: %w", sourceAdapter.Service(), err)
	}
	if sourceSong == nil {
		return nil, fmt.Errorf("%w from %s", errNilSourceSong, sourceAdapter.Service())
	}
	return sourceSong, nil
}

func (r *SongResolver) parseSource(inputURL string) (SongSourceAdapter, *model.ParsedURL, error) {
	return parseSourceAdapter(
		r.sources,
		inputURL,
		func(source SongSourceAdapter, raw string) (*model.ParsedURL, error) {
			return source.ParseSongURL(raw)
		},
	)
}

func (r *SongResolver) collectCandidates(ctx context.Context, target SongTargetAdapter, source model.CanonicalSong) ([]model.CandidateSong, error) {
	return collectCandidateLayers(ctx, songCandidateKey,
		candidateLayer[model.CandidateSong]{
			enabled: source.ISRC != "",
			search: func(ctx context.Context) ([]model.CandidateSong, error) {
				return target.SearchSongByISRC(ctx, source.ISRC)
			},
		},
		candidateLayer[model.CandidateSong]{
			enabled: true,
			search: func(ctx context.Context) ([]model.CandidateSong, error) {
				return target.SearchSongByMetadata(ctx, source)
			},
		},
	)
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
