package ariadne

import (
	"context"
	"errors"
	"net/http"

	"github.com/xmbshwll/ariadne/internal/httpx"
	"github.com/xmbshwll/ariadne/internal/resolve"
	"github.com/xmbshwll/ariadne/internal/score"
)

// Resolver wraps the internal resolvers with a public library-facing API.
type Resolver struct {
	inner     *resolve.Resolver
	songInner *resolve.SongResolver
}

func (r *Resolver) albumResolver() (*resolve.Resolver, error) {
	if r == nil || r.inner == nil {
		return nil, ErrResolverNotInitialized
	}
	return r.inner, nil
}

func (r *Resolver) songResolver() (*resolve.SongResolver, error) {
	if r == nil || r.songInner == nil {
		return nil, ErrResolverNotInitialized
	}
	return r.songInner, nil
}

// New builds a Resolver with the default adapter set and a default HTTP client.
func New(config Config) *Resolver {
	return NewWithClient(nil, config)
}

// NewWithClient builds a Resolver with the default adapter set and a caller-provided HTTP client.
func NewWithClient(client *http.Client, config Config) *Resolver {
	config = normalizedConfig(config)
	if client == nil {
		client = httpx.NewClient(config.HTTPTimeout)
	}
	adapterSets := buildDefaultServiceAdapters(client, config)
	return newResolver(
		defaultSourceAdapters(adapterSets),
		defaultTargetAdapters(adapterSets, config.TargetServices),
		defaultSongSourceAdapters(adapterSets),
		defaultSongTargetAdapters(adapterSets, config.TargetServices),
		toInternalScoreWeights(config.ScoreWeights),
		toInternalSongScoreWeights(config.SongScoreWeights),
	)
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
	return newResolver(
		wrapSourceAdapters(albumSources),
		wrapTargetAdapters(albumTargets),
		wrapSongSourceAdapters(songSources),
		wrapSongTargetAdapters(songTargets),
		toInternalScoreWeights(albumWeights),
		toInternalSongScoreWeights(songWeights),
	)
}

func newResolver(
	albumSources []resolve.SourceAdapter,
	albumTargets []resolve.TargetAdapter,
	songSources []resolve.SongSourceAdapter,
	songTargets []resolve.SongTargetAdapter,
	albumWeights score.Weights,
	songWeights score.SongWeights,
) *Resolver {
	return &Resolver{
		inner:     resolve.New(albumSources, albumTargets, albumWeights),
		songInner: resolve.NewSongs(songSources, songTargets, songWeights),
	}
}

// ResolveAlbum resolves one input album URL into a canonical source album plus per-service matches.
//
// Callers should use errors.Is on the returned error when branching on public
// resolver failure modes. The stable exported sentinels are:
//   - ErrResolverNotInitialized when ResolveAlbum is called on a nil Resolver
//     or one whose albumResolver guard detects a missing inner resolver
//   - ErrUnsupportedURL when no registered source adapter recognizes inputURL
//   - ErrNoSourceAdapters when the resolver was built without any source adapters
//   - ErrRuntimeDeferred when a recognized URL can parse, but runtime hydration
//     is intentionally deferred
//   - ErrAmazonMusicDeferred when an Amazon Music URL is recognized but runtime
//     resolution is intentionally deferred
//   - ErrAppleMusicCredentialsNotConfigured when an Apple Music official API
//     operation requires developer token credentials
//   - ErrSpotifyCredentialsNotConfigured when a Spotify Web API operation
//     requires app credentials
//   - ErrTIDALCredentialsNotConfigured when a TIDAL source or target operation
//     requires credentials that are not configured
//   - ErrSourceAdapterReturnedNilParsedURL or ErrSourceAdapterReturnedNilAlbum
//     when a caller-provided custom source adapter violates the adapter contract
func (r *Resolver) ResolveAlbum(ctx context.Context, inputURL string) (*Resolution, error) {
	resolver, err := r.albumResolver()
	if err != nil {
		return nil, err
	}

	resolution, err := resolver.ResolveAlbum(ctx, inputURL)
	if err != nil {
		//nolint:wrapcheck // Preserve the underlying resolver error for callers and CLI output.
		return nil, err
	}
	public := fromInternalResolution(*resolution)
	return &public, nil
}

// ResolveSong resolves one input song URL into a canonical source song plus per-service matches.
//
// Callers should use errors.Is on the returned error when branching on
// ResolveSong failure modes. The stable exported sentinels are:
//   - ErrResolverNotInitialized when ResolveSong is called on a nil Resolver
//     or one whose songResolver guard detects a missing inner resolver
//   - ErrUnsupportedURL when no registered source adapter recognizes inputURL
//   - ErrNoSourceAdapters when the resolver was built without any source adapters
//   - ErrRuntimeDeferred when a recognized URL can parse, but runtime hydration
//     is intentionally deferred
//   - ErrAmazonMusicDeferred when an Amazon Music URL is recognized but runtime
//     resolution is intentionally deferred
//   - ErrYouTubeMusicDeferred when a YouTube Music song URL is recognized but
//     runtime song hydration is intentionally deferred
//   - ErrAppleMusicCredentialsNotConfigured when an Apple Music official API
//     operation requires developer token credentials
//   - ErrSpotifyCredentialsNotConfigured when a Spotify Web API operation
//     requires app credentials
//   - ErrTIDALCredentialsNotConfigured when a TIDAL source or target operation
//     requires credentials that are not configured
//   - ErrSourceAdapterReturnedNilParsedURL or ErrSourceAdapterReturnedNilSong
//     when a caller-provided custom song source adapter violates the adapter contract
func (r *Resolver) ResolveSong(ctx context.Context, inputURL string) (*SongResolution, error) {
	resolver, err := r.songResolver()
	if err != nil {
		return nil, err
	}

	resolution, err := resolver.ResolveSong(ctx, inputURL)
	if err != nil {
		//nolint:wrapcheck // Preserve the underlying resolver error for callers and CLI output.
		return nil, err
	}
	public := fromInternalSongResolution(*resolution)
	return &public, nil
}

// Resolve tries ResolveSong first and returns an EntityResolution containing
// either Song or Album. Non-fallback ResolveSong failures, such as credential
// errors, are returned immediately. Resolve falls back to ResolveAlbum only
// when ResolveSong returns ErrUnsupportedURL or ErrNoSourceAdapters.
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
