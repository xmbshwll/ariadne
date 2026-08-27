package resolve

import (
	"context"

	"github.com/xmbshwll/ariadne/internal/adapters"
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/score"
)

type (
	// SongScoredMatch is one scored song candidate.
	SongScoredMatch = ScoredMatchOf[model.CandidateSong]
	// SongMatchResult is the song resolver output for one target service.
	SongMatchResult = MatchResultOf[model.CandidateSong]
	// SongResolution is the song resolver output.
	SongResolution = ResolutionOf[model.ParsedURL, model.CanonicalSong, model.CandidateSong]
)

const songEntityLabel = "song"

// SongResolver coordinates song Source Input, Runtime Hydration, and Target Search.
type SongResolver struct {
	policy songEntityResolutionPolicy
}

// songEntityResolutionPolicy is the Entity Resolution pipeline configured for songs.
type songEntityResolutionPolicy = entityResolution[model.ParsedURL, model.CanonicalSong, model.CandidateSong]

func newSongEntityResolutionPolicy(sources []adapters.Adapter, targets []adapters.Adapter, weights score.SongWeights) songEntityResolutionPolicy {
	return songEntityResolutionPolicy{
		targetAdapters:    append([]adapters.Adapter(nil), targets...),
		collectCandidates: collectSongTargetCandidates,
		rank: func(song model.CanonicalSong, candidates []model.CandidateSong) score.Ranking[model.CandidateSong] {
			return score.RankSongs(song, candidates, weights)
		},
		entityService:  func(song model.CanonicalSong) model.ServiceName { return song.Service },
		candidateURL:   func(candidate model.CandidateSong) string { return candidate.MatchURL },
		collectFailure: "collect song candidates",
		resolveSourceInput: func(ctx context.Context, inputURL string) (model.ParsedURL, model.CanonicalSong, error) {
			return resolveEntitySourceInput(ctx, sources, inputURL, songEntityLabel, ErrNilSourceSong,
				func(source adapters.Adapter, rawURL string) (*model.ParsedURL, error) {
					return source.ParseSongURL(rawURL)
				},
				func(ctx context.Context, source adapters.Adapter, parsed *model.ParsedURL) (*model.CanonicalSong, error) {
					return source.FetchSong(ctx, *parsed)
				})
		},
	}
}

// NewSongs creates a song resolver from registered source and target adapters.
// Adapters whose Capabilities report no song Target Search contribute no layers.
func NewSongs(sources []adapters.Adapter, targets []adapters.Adapter, weights score.SongWeights) *SongResolver {
	return &SongResolver{policy: newSongEntityResolutionPolicy(sources, targets, weights)}
}

// ResolveSong parses an input song URL, fetches the canonical source song,
// then collects and ranks candidates from every target adapter except the source
// service. A failing target does not abort the resolution: its SongMatchResult
// carries the error in Err while other targets resolve normally.
func (r *SongResolver) ResolveSong(ctx context.Context, inputURL string) (*SongResolution, error) {
	return r.policy.resolve(ctx, inputURL)
}
