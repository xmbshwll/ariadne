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

type (
	// SongScoredMatch is one scored song candidate.
	SongScoredMatch = ScoredMatchOf[model.CandidateSong]
	// SongMatchResult is the song resolver output for one target service.
	SongMatchResult = MatchResultOf[model.CandidateSong]
	// SongResolution is the song resolver output.
	SongResolution = ResolutionOf[model.ParsedURL, model.CanonicalSong, model.CandidateSong]
)

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
// then collects and ranks candidates from every target adapter except the source
// service. A failing target does not abort the resolution: its SongMatchResult
// carries the error in Err while other targets resolve normally.
func (r *SongResolver) ResolveSong(ctx context.Context, inputURL string) (*SongResolution, error) {
	return r.policy.resolve(ctx, inputURL)
}

func (p songEntityResolutionPolicy) resolve(ctx context.Context, inputURL string) (*SongResolution, error) {
	source, err := p.resolveSourceInput(ctx, inputURL)
	if err != nil {
		return nil, err
	}

	targets := excludeTargetService(p.targetAdapters, source.Entity.Service)
	matches := p.resolveTargetMatches(ctx, targets, source.Entity)

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

func (p songEntityResolutionPolicy) resolveTargetMatches(ctx context.Context, targets []SongTargetAdapter, source model.CanonicalSong) map[model.ServiceName]SongMatchResult {
	matches := make(map[model.ServiceName]SongMatchResult, len(targets))
	var matchesMu sync.Mutex

	resolveTargetsConcurrently(ctx, targets, func(targetCtx context.Context, target SongTargetAdapter) {
		var result SongMatchResult
		candidates, err := collectSongTargetCandidates(targetCtx, target, source)
		if err != nil {
			result = SongMatchResult{Service: target.Service(), Err: fmt.Errorf("collect song candidates: %w", err)}
		} else {
			result = songMatchResultFromRanking(target.Service(), score.RankSongs(source, candidates, p.weights))
		}

		matchesMu.Lock()
		matches[target.Service()] = result
		matchesMu.Unlock()
	})
	return matches
}

func songCandidateKey(candidate model.CandidateSong) string {
	if candidate.CandidateID != "" {
		return string(candidate.Service) + ":id:" + candidate.CandidateID
	}
	return string(candidate.Service) + ":url:" + candidate.MatchURL
}

func songMatchResultFromRanking(service model.ServiceName, ranking score.Ranking[model.CandidateSong]) SongMatchResult {
	return matchResultFromRanking(service, ranking, func(candidate model.CandidateSong) string { return candidate.MatchURL })
}
