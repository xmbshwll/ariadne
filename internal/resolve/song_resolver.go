package resolve

import (
	"context"
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
	policy songEntityResolutionPolicy
}

type songEntityResolutionPolicy struct {
	sourceAdapters []SongSourceAdapter
	targetAdapters []SongTargetAdapter
	weights        score.SongWeights
}

func newSongEntityResolutionPolicy(sources []SongSourceAdapter, targets []SongTargetAdapter, weights score.SongWeights) songEntityResolutionPolicy {
	return songEntityResolutionPolicy{
		sourceAdapters: append([]SongSourceAdapter(nil), sources...),
		targetAdapters: append([]SongTargetAdapter(nil), targets...),
		weights:        weights,
	}
}

// NewSongs creates a song resolver from registered source and target adapters.
// Adapters that implement no song search interfaces produce no target search layers.
func NewSongs(sources []SongSourceAdapter, targets []SongTargetAdapter, weights score.SongWeights) *SongResolver {
	return &SongResolver{policy: newSongEntityResolutionPolicy(sources, targets, weights)}
}

// ResolveSong parses an input song URL, fetches the canonical source song,
// then collects and ranks candidates from every target adapter except the source service.
func (r *SongResolver) ResolveSong(ctx context.Context, inputURL string) (*SongResolution, error) {
	return r.policy.resolve(ctx, inputURL)
}

func (p songEntityResolutionPolicy) resolve(ctx context.Context, inputURL string) (*SongResolution, error) {
	source, err := p.resolveSourceInput(ctx, inputURL)
	if err != nil {
		return nil, err
	}

	targets := excludeTargetService(p.targetAdapters, source.Entity.Service)
	matches, err := p.resolveTargetMatches(ctx, targets, source.Entity)
	if err != nil {
		return nil, err
	}

	return &SongResolution{
		InputURL: inputURL,
		Parsed:   source.Parsed,
		Source:   source.Entity,
		Matches:  matches,
	}, nil
}

// songSourceInput pairs the recognized Source Input with its hydrated song.
type songSourceInput struct {
	Parsed model.ParsedURL
	Entity model.CanonicalSong
}

func (p songEntityResolutionPolicy) resolveSourceInput(ctx context.Context, inputURL string) (songSourceInput, error) {
	return resolveSongSourceInput(ctx, p.sourceAdapters, inputURL)
}

func resolveSongSourceInput(ctx context.Context, sources []SongSourceAdapter, inputURL string) (songSourceInput, error) {
	parsed, adapter, err := recognizeSourceInput(sources, inputURL, func(source SongSourceAdapter) (*model.ParsedURL, error) {
		return source.ParseSongURL(inputURL)
	})
	if err != nil {
		return songSourceInput{}, err
	}

	song, err := hydrateSourceInput(ctx, adapter, "song", errNilSourceSong,
		func(ctx context.Context) (*model.CanonicalSong, error) {
			return adapter.FetchSong(ctx, *parsed)
		})
	if err != nil {
		return songSourceInput{}, err
	}

	return songSourceInput{Parsed: *parsed, Entity: *song}, nil
}

func (p songEntityResolutionPolicy) resolveTargetMatches(ctx context.Context, targets []SongTargetAdapter, source model.CanonicalSong) (map[model.ServiceName]SongMatchResult, error) {
	matches := make(map[model.ServiceName]SongMatchResult, len(targets))
	var matchesMu sync.Mutex

	err := resolveTargetsConcurrently(ctx, targets, func(groupCtx context.Context, target SongTargetAdapter) error {
		candidates, err := collectSongTargetCandidates(groupCtx, target, source)
		if err != nil {
			return fmt.Errorf("collect song candidates from %s: %w", target.Service(), err)
		}
		ranking := score.RankSongs(source, candidates, p.weights)

		matchesMu.Lock()
		matches[target.Service()] = songMatchResultFromRanking(target.Service(), ranking)
		matchesMu.Unlock()
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("resolve song target searches: %w", err)
	}
	return matches, nil
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
