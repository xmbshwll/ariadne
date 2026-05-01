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

// SongResolver coordinates source parsing, source fetching, and layered target search for songs.
type SongResolver struct {
	core entityResolver[SongSourceAdapter, SongTargetAdapter, model.ParsedURL, model.CanonicalSong, model.CandidateSong, score.SongRanking, SongMatchResult]
}

// NewSongs creates a song resolver from registered source and target adapters.
// Adapters that implement neither SongISRCSearcher nor SongMetadataSearcher
// cannot participate in target search and will cause a panic.
func NewSongs(sources []SongSourceAdapter, targets []SongTargetAdapter, weights score.SongWeights) *SongResolver {
	for i, t := range targets {
		if _, okISRC := t.(SongISRCSearcher); !okISRC {
			if _, okMeta := t.(SongMetadataSearcher); !okMeta {
				panic(fmt.Sprintf("song target adapter at index %d (%s) implements neither SongISRCSearcher nor SongMetadataSearcher", i, t.Service()))
			}
		}
	}
	return &SongResolver{
		core: newEntityResolver(sources, targets, songResolutionPipeline(weights)),
	}
}

// ResolveSong parses an input song URL, fetches the canonical source song,
// then collects and ranks candidates from every target adapter except the source service.
func (r *SongResolver) ResolveSong(ctx context.Context, inputURL string) (*SongResolution, error) {
	result, err := r.core.resolve(ctx, inputURL)
	if err != nil {
		return nil, err
	}

	return &SongResolution{
		InputURL: result.InputURL,
		Parsed:   result.Parsed,
		Source:   result.Source,
		Matches:  result.Matches,
	}, nil
}

func songResolutionPipeline(weights score.SongWeights) entityResolutionPipeline[SongSourceAdapter, SongTargetAdapter, model.ParsedURL, model.CanonicalSong, model.CandidateSong, score.SongRanking, SongMatchResult] {
	return entityResolutionPipeline[SongSourceAdapter, SongTargetAdapter, model.ParsedURL, model.CanonicalSong, model.CandidateSong, score.SongRanking, SongMatchResult]{
		parse: func(source SongSourceAdapter, raw string) (*model.ParsedURL, error) {
			return source.ParseSongURL(raw)
		},
		hydrate: func(ctx context.Context, source SongSourceAdapter, parsed model.ParsedURL) (*model.CanonicalSong, error) {
			return source.FetchSong(ctx, parsed)
		},
		sourceService: func(source model.CanonicalSong) model.ServiceName {
			return source.Service
		},
		collect: collectSongTargetCandidates,
		rank: func(source model.CanonicalSong, candidates []model.CandidateSong) score.SongRanking {
			return score.RankSongs(source, candidates, weights)
		},
		result:         songMatchResultFromRanking,
		entityLabel:    "song",
		nilEntityErr:   errNilSourceSong,
		candidateLabel: "song candidates",
		targetErrLabel: "resolve song target searches",
	}
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
