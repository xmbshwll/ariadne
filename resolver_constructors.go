package ariadne

import (
	"context"
	"errors"
	"net/http"

	"github.com/xmbshwll/ariadne/internal/httpx"
	"github.com/xmbshwll/ariadne/internal/resolve"
	"github.com/xmbshwll/ariadne/internal/score"
	"github.com/xmbshwll/ariadne/internal/wiring"
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

// Option customizes New. Most callers need no options.
type Option func(*resolverOptions)

type resolverOptions struct {
	client *http.Client
}

// WithHTTPClient makes New use a caller-provided HTTP client for the default adapter set.
func WithHTTPClient(client *http.Client) Option {
	return func(options *resolverOptions) {
		options.client = client
	}
}

// New builds a Resolver with the default adapter set from the Provider Catalog.
func New(config Config, opts ...Option) *Resolver {
	options := resolverOptions{}
	for _, opt := range opts {
		opt(&options)
	}
	config = normalizedConfig(config)
	client := options.client
	if client == nil {
		client = httpx.NewClient(config.HTTPTimeout)
	}
	adapters := wiring.DefaultResolverAdapters(client, internalConfig(config), config.TargetServices)
	return newResolver(
		adapters.AlbumSources,
		adapters.AlbumTargets,
		adapters.SongSources,
		adapters.SongTargets,
		config.ScoreWeights,
		config.SongScoreWeights,
	)
}

// AdapterSet carries caller-provided adapters and ranking weights for
// NewWithAdapters. Zero weights use the defaults.
type AdapterSet struct {
	AlbumSources []SourceAdapter
	AlbumTargets []TargetAdapter
	SongSources  []SongSourceAdapter
	SongTargets  []SongTargetAdapter
	Weights      ScoreWeights
	SongWeights  SongScoreWeights
}

// NewWithAdapters builds a Resolver from a caller-provided AdapterSet.
func NewWithAdapters(set AdapterSet) *Resolver {
	weights := set.Weights
	if weights == (ScoreWeights{}) {
		weights = DefaultScoreWeights()
	}
	songWeights := set.SongWeights
	if songWeights == (SongScoreWeights{}) {
		songWeights = DefaultSongScoreWeights()
	}
	return newResolver(
		set.AlbumSources,
		set.AlbumTargets,
		set.SongSources,
		set.SongTargets,
		weights,
		songWeights,
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
// A failing target service does not fail the resolution: its MatchResult carries
// the error in Err while other services resolve normally.
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
	return resolution, nil
}

// ResolveSong resolves one input song URL into a canonical source song plus per-service matches.
// A failing target service does not fail the resolution: its SongMatchResult
// carries the error in Err while other services resolve normally.
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
	return resolution, nil
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
