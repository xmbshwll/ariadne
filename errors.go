package ariadne

import (
	"errors"

	"github.com/xmbshwll/ariadne/internal/adapters/adapterutil"
	amazonmusicadapter "github.com/xmbshwll/ariadne/internal/adapters/amazonmusic"
	applemusicadapter "github.com/xmbshwll/ariadne/internal/adapters/applemusic"
	spotifyadapter "github.com/xmbshwll/ariadne/internal/adapters/spotify"
	tidaladapter "github.com/xmbshwll/ariadne/internal/adapters/tidal"
	youtubemusicadapter "github.com/xmbshwll/ariadne/internal/adapters/youtubemusic"
	"github.com/xmbshwll/ariadne/internal/resolve"
)

var (
	// ErrUnsupportedURL indicates that no registered source adapter recognized the input URL.
	ErrUnsupportedURL = resolve.ErrUnsupportedURL
	// ErrNoSourceAdapters indicates that a resolver was created without source adapters.
	ErrNoSourceAdapters = resolve.ErrNoSourceAdapters
	// ErrResolverNotInitialized indicates that a public Resolver receiver or inner resolver was nil.
	ErrResolverNotInitialized = errors.New("resolver not initialized")
	// ErrRuntimeDeferred indicates that a recognized URL can be parsed, but runtime hydration remains intentionally deferred.
	ErrRuntimeDeferred = adapterutil.ErrRuntimeDeferred
	// ErrAmazonMusicDeferred indicates that Amazon Music URLs are recognized, but runtime resolution remains intentionally deferred.
	ErrAmazonMusicDeferred = amazonmusicadapter.ErrDeferredRuntimeAdapter
	// ErrYouTubeMusicDeferred indicates that YouTube Music song URLs are recognized, but runtime song hydration remains intentionally deferred.
	ErrYouTubeMusicDeferred = youtubemusicadapter.ErrDeferredRuntimeAdapter
	// ErrAppleMusicCredentialsNotConfigured indicates that an Apple Music official API operation requires developer token credentials.
	ErrAppleMusicCredentialsNotConfigured = applemusicadapter.ErrCredentialsNotConfigured
	// ErrSpotifyCredentialsNotConfigured indicates that a Spotify Web API operation requires app credentials.
	ErrSpotifyCredentialsNotConfigured = spotifyadapter.ErrCredentialsNotConfigured
	// ErrTIDALCredentialsNotConfigured indicates that a TIDAL operation requires app credentials that were not configured.
	ErrTIDALCredentialsNotConfigured = tidaladapter.ErrCredentialsNotConfigured
	// ErrSourceAdapterReturnedNilParsedURL indicates that a caller-provided source adapter returned a nil parsed URL instead of either a parsed value or an error.
	ErrSourceAdapterReturnedNilParsedURL = errors.New("source adapter returned nil parsed url")
	// ErrSourceAdapterReturnedNilAlbum indicates that a caller-provided album source adapter returned a nil album without an error.
	ErrSourceAdapterReturnedNilAlbum = errors.New("source adapter returned nil album")
	// ErrSourceAdapterReturnedNilSong indicates that a caller-provided song source adapter returned a nil song without an error.
	ErrSourceAdapterReturnedNilSong = errors.New("source adapter returned nil song")
)
