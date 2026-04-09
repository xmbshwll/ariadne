package ariadne

import (
	"context"
	"errors"
	"net/http"

	"github.com/xmbshwll/ariadne/internal/httpx"
	"github.com/xmbshwll/ariadne/internal/resolve"
)

// Resolver wraps the internal resolvers with a public library-facing API.
type Resolver struct {
	inner     *resolve.Resolver
	songInner *resolve.SongResolver
}

// New builds a Resolver with the default adapter set and a default HTTP client.
func New(config Config) *Resolver {
	config = normalizedConfig(config)
	return NewWithClient(httpx.NewClient(config.HTTPTimeout), config)
}

// NewWithClient builds a Resolver with the default adapter set and a caller-provided HTTP client.
func NewWithClient(client *http.Client, config Config) *Resolver {
	if client == nil {
		client = http.DefaultClient
	}
	config = normalizedConfig(config)
	adapters := newDefaultAdapters(client, config)
	return &Resolver{
		inner: resolve.New(
			defaultSourceAdapters(adapters),
			defaultTargetAdapters(adapters, config),
			toInternalScoreWeights(config.ScoreWeights),
		),
		songInner: resolve.NewSongs(
			defaultSongSourceAdapters(adapters),
			defaultSongTargetAdapters(adapters, config),
			toInternalSongScoreWeights(config.SongScoreWeights),
		),
	}
}

// NewWithAdapters builds a Resolver from caller-provided album source and target adapters using the default ranking weights.
func NewWithAdapters(sources []SourceAdapter, targets []TargetAdapter) *Resolver {
	return NewWithAdaptersAndWeights(sources, targets, DefaultScoreWeights())
}

// NewWithAdaptersAndWeights builds a Resolver from caller-provided album source and target adapters and explicit ranking weights.
func NewWithAdaptersAndWeights(sources []SourceAdapter, targets []TargetAdapter, weights ScoreWeights) *Resolver {
	return NewWithEntityAdaptersAndWeights(sources, targets, nil, nil, weights, DefaultSongScoreWeights())
}

// NewWithEntityAdapters builds a Resolver from caller-provided album and song adapters using default ranking weights.
func NewWithEntityAdapters(albumSources []SourceAdapter, albumTargets []TargetAdapter, songSources []SongSourceAdapter, songTargets []SongTargetAdapter) *Resolver {
	return NewWithEntityAdaptersAndWeights(
		albumSources,
		albumTargets,
		songSources,
		songTargets,
		DefaultScoreWeights(),
		DefaultSongScoreWeights(),
	)
}

// NewWithEntityAdaptersAndWeights builds a Resolver from caller-provided album and song adapters and explicit ranking weights.
func NewWithEntityAdaptersAndWeights(albumSources []SourceAdapter, albumTargets []TargetAdapter, songSources []SongSourceAdapter, songTargets []SongTargetAdapter, albumWeights ScoreWeights, songWeights SongScoreWeights) *Resolver {
	return &Resolver{
		inner: resolve.New(
			wrapSourceAdapters(albumSources),
			wrapTargetAdapters(albumTargets),
			toInternalScoreWeights(albumWeights),
		),
		songInner: resolve.NewSongs(
			wrapSongSourceAdapters(songSources),
			wrapSongTargetAdapters(songTargets),
			toInternalSongScoreWeights(songWeights),
		),
	}
}

// ResolveAlbum resolves one input album URL into a canonical source album plus per-service matches.
//
// Callers should use errors.Is on the returned error when branching on public
// resolver failure modes. The stable exported sentinels are:
//   - ErrUnsupportedURL when no registered source adapter recognizes inputURL
//   - ErrNoSourceAdapters when the resolver was built without any source adapters
//   - ErrAmazonMusicDeferred when an Amazon Music URL is recognized but runtime
//     resolution is intentionally deferred
//   - ErrAppleMusicCredentialsNotConfigured when an Apple Music official API
//     operation requires developer token credentials
//   - ErrSpotifyCredentialsNotConfigured when a Spotify Web API operation
//     requires app credentials
//   - ErrTIDALCredentialsNotConfigured when a TIDAL source or target operation
//     requires credentials that are not configured
func (r *Resolver) ResolveAlbum(ctx context.Context, inputURL string) (*Resolution, error) {
	resolution, err := r.inner.ResolveAlbum(ctx, inputURL)
	if err != nil {
		//nolint:wrapcheck // Preserve the underlying resolver error for callers and CLI output.
		return nil, err
	}
	public := fromInternalResolution(*resolution)
	return &public, nil
}

// ResolveSong resolves one input song URL into a canonical source song plus per-service matches.
func (r *Resolver) ResolveSong(ctx context.Context, inputURL string) (*SongResolution, error) {
	resolution, err := r.songInner.ResolveSong(ctx, inputURL)
	if err != nil {
		//nolint:wrapcheck // Preserve the underlying resolver error for callers and CLI output.
		return nil, err
	}
	public := fromInternalSongResolution(*resolution)
	return &public, nil
}

// Resolve parses an input URL and dispatches to album or song resolution.
func (r *Resolver) Resolve(ctx context.Context, inputURL string) (*EntityResolution, error) {
	songResolution, err := r.ResolveSong(ctx, inputURL)
	if err == nil {
		return &EntityResolution{Parsed: songResolution.Parsed, Song: songResolution}, nil
	}
	if !errors.Is(err, ErrUnsupportedURL) && !errors.Is(err, ErrNoSourceAdapters) {
		return nil, err
	}

	albumResolution, albumErr := r.ResolveAlbum(ctx, inputURL)
	if albumErr != nil {
		return nil, albumErr
	}
	return &EntityResolution{Parsed: albumResolution.Parsed, Album: albumResolution}, nil
}
