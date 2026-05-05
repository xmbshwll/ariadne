package resolve

import (
	"context"
	"fmt"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/score"
)

// SongSourceAdapter fetches canonical song metadata from a parsed source URL.
type SongSourceAdapter interface {
	Service() model.ServiceName
	ParseSongURL(raw string) (*model.ParsedURL, error)
	FetchSong(ctx context.Context, parsed model.ParsedURL) (*model.CanonicalSong, error)
}

// SongTargetAdapter identifies one song target Music Service.
type SongTargetAdapter interface {
	Service() model.ServiceName
}

// SongISRCSearcher searches song targets by ISRC.
type SongISRCSearcher interface {
	SearchSongByISRC(ctx context.Context, isrc string) ([]model.CandidateSong, error)
}

// SongMetadataSearcher searches song targets by canonical metadata.
type SongMetadataSearcher interface {
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

// SongResolver coordinates song Source Input, Runtime Hydration, and Target Search.
type SongResolver struct {
	sources []SongSourceAdapter
	targets []SongTargetAdapter
	weights score.SongWeights
}

// NewSongs creates a song resolver from registered source and target adapters.
// Adapters that implement neither SongISRCSearcher nor SongMetadataSearcher
// cannot participate in target search and will cause a panic.
func NewSongs(sources []SongSourceAdapter, targets []SongTargetAdapter, weights score.SongWeights) *SongResolver {
	for i, target := range targets {
		if !searchableSongTarget(target) {
			panic(fmt.Sprintf("song target adapter at index %d (%s) implements neither SongISRCSearcher nor SongMetadataSearcher", i, target.Service()))
		}
	}
	return &SongResolver{
		sources: append([]SongSourceAdapter(nil), sources...),
		targets: append([]SongTargetAdapter(nil), targets...),
		weights: weights,
	}
}

func searchableSongTarget(target SongTargetAdapter) bool {
	if _, ok := target.(SongISRCSearcher); ok {
		return true
	}
	_, ok := target.(SongMetadataSearcher)
	return ok
}

// ResolveSong parses an input song URL, fetches the canonical source song,
// then collects and ranks candidates from every target adapter except the source service.
func (r *SongResolver) ResolveSong(ctx context.Context, inputURL string) (*SongResolution, error) {
	source, err := resolveSongSourceInput(ctx, r.sources, inputURL)
	if err != nil {
		return nil, err
	}

	targets := excludeTargetService(r.targets, source.Entity.Service)
	matches, err := r.resolveSongTargets(ctx, targets, source.Entity)
	if err != nil {
		return nil, fmt.Errorf("resolve song target searches: %w", err)
	}

	return &SongResolution{
		InputURL: inputURL,
		Parsed:   source.Parsed,
		Source:   source.Entity,
		Matches:  matches,
	}, nil
}

func resolveSongSourceInput(ctx context.Context, sources []SongSourceAdapter, inputURL string) (sourceInput[model.ParsedURL, model.CanonicalSong], error) {
	return resolveSourceInput(
		ctx,
		sources,
		inputURL,
		func(source SongSourceAdapter, raw string) (*model.ParsedURL, error) {
			return source.ParseSongURL(raw)
		},
		func(ctx context.Context, source SongSourceAdapter, parsed model.ParsedURL) (*model.CanonicalSong, error) {
			return source.FetchSong(ctx, parsed)
		},
		"song",
		errNilSourceSong,
	)
}

func (r *SongResolver) resolveSongTargets(ctx context.Context, targets []SongTargetAdapter, source model.CanonicalSong) (map[model.ServiceName]SongMatchResult, error) {
	return resolveTargetMatches(
		ctx,
		targets,
		source,
		collectSongTargetCandidates,
		func(source model.CanonicalSong, candidates []model.CandidateSong) score.SongRanking {
			return score.RankSongs(source, candidates, r.weights)
		},
		songMatchResultFromRanking,
		"song candidates",
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
