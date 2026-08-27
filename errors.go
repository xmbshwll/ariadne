package ariadne

import (
	"errors"

	"github.com/xmbshwll/ariadne/internal/adapters"
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
	// ErrNoSourceAdapters indicates that no source adapter was registered for the Entity Shape
	// of the input. Auto mode treats it as "not a song URL" and falls back to albums.
	ErrNoSourceAdapters = resolve.ErrNoSourceAdapters
	// ErrResolverNotInitialized indicates that a public Resolver receiver or inner resolver was nil.
	ErrResolverNotInitialized = errors.New("resolver not initialized")
	// ErrRuntimeDeferred indicates that a recognized URL can be parsed, but runtime hydration remains intentionally deferred.
	ErrRuntimeDeferred = adapters.ErrRuntimeDeferred
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
	// ErrSourceAdapterReturnedNilParsedURL indicates that a source adapter answered a parse with a
	// nil parsed URL and no error, which breaks the adapter contract.
	ErrSourceAdapterReturnedNilParsedURL = resolve.ErrSourceAdapterReturnedNilParsedURL
	// ErrSourceAdapterReturnedNilAlbum indicates that an album source adapter returned a nil album
	// and no error, which breaks the adapter contract.
	ErrSourceAdapterReturnedNilAlbum = resolve.ErrSourceAdapterReturnedNilAlbum
	// ErrSourceAdapterReturnedNilSong indicates that a song source adapter returned a nil song and
	// no error, which breaks the adapter contract.
	ErrSourceAdapterReturnedNilSong = resolve.ErrSourceAdapterReturnedNilSong
)
