package amazonmusic

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/xmbshwll/ariadne/internal/adapters"
	"github.com/xmbshwll/ariadne/internal/adapters/base"
	"github.com/xmbshwll/ariadne/internal/model"
)

const runtimeDeferredReason = "no viable public metadata fetch or search path exists"

var (
	// ErrDeferredRuntimeAdapter indicates that an Amazon Music URL parsed successfully, but runtime hydration is intentionally deferred.
	ErrDeferredRuntimeAdapter  = adapters.RuntimeDeferredService(model.ServiceAmazonMusic)
	errUnexpectedAmazonService = errors.New("unexpected amazon music service")
)

type Adapter struct {
	base.Unsupported
}

func New(_ *http.Client) *Adapter {
	return &Adapter{Unsupported: base.Unsupported{ServiceName: model.ServiceAmazonMusic}}
}

func (a *Adapter) ParseAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	return ParseAlbumURL(raw)
}

func (a *Adapter) ParseSongURL(raw string) (*model.ParsedURL, error) {
	return ParseSongURL(raw)
}

func (a *Adapter) FetchAlbum(_ context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalAlbum, error) {
	if parsed.Service != model.ServiceAmazonMusic {
		return nil, fmt.Errorf("%w: %s", errUnexpectedAmazonService, parsed.Service)
	}
	//nolint:wrapcheck // Preserve the deferred-runtime sentinel for errors.Is callers.
	return nil, adapters.NewRuntimeDeferredError(model.ServiceAmazonMusic, runtimeDeferredReason)
}

func (a *Adapter) FetchSong(_ context.Context, parsed model.ParsedURL) (*model.CanonicalSong, error) {
	if parsed.Service != model.ServiceAmazonMusic {
		return nil, fmt.Errorf("%w: %s", errUnexpectedAmazonService, parsed.Service)
	}
	//nolint:wrapcheck // Preserve the deferred-runtime sentinel for errors.Is callers.
	return nil, adapters.NewRuntimeDeferredError(model.ServiceAmazonMusic, runtimeDeferredReason)
}
