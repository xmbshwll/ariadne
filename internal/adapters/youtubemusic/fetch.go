package youtubemusic

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/xmbshwll/ariadne/internal/adapters/adapterutil"
	"github.com/xmbshwll/ariadne/internal/model"
)

const (
	maxYouTubeMusicResponseBytes      = 10 << 20
	maxYouTubeMusicErrorResponseBytes = 4096
	youTubeMusicFetchTimeout          = 15 * time.Second
)

var errYouTubeMusicResponseTooLarge = errors.New("youtube music response too large")

func (a *Adapter) FetchAlbum(ctx context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalAlbum, error) {
	if parsed.Service != model.ServiceYouTubeMusic {
		return nil, fmt.Errorf("%w: %s", errUnexpectedYouTubeMusicService, parsed.Service)
	}
	body, err := a.fetchPage(ctx, parsed.CanonicalURL)
	if err != nil {
		return nil, fmt.Errorf("fetch youtube music page: %w", err)
	}
	return extractAlbum(body, parsed.CanonicalURL)
}

func (a *Adapter) FetchSong(_ context.Context, parsed model.ParsedURL) (*model.CanonicalSong, error) {
	if parsed.Service != model.ServiceYouTubeMusic {
		return nil, fmt.Errorf("%w: %s", errUnexpectedYouTubeMusicService, parsed.Service)
	}
	//nolint:wrapcheck // Preserve the deferred-runtime sentinel for errors.Is callers.
	return nil, adapterutil.NewRuntimeDeferredError(model.ServiceYouTubeMusic, songRuntimeDeferred)
}

func (a *Adapter) fetchAlbumByBrowseID(ctx context.Context, browseID string) (*model.CanonicalAlbum, error) {
	browseURL := a.baseURL + "/browse/" + browseID
	body, err := a.fetchPage(ctx, browseURL)
	if err != nil {
		return nil, err
	}
	return extractAlbum(body, browseURL)
}

func (a *Adapter) fetchPage(ctx context.Context, requestURL string) ([]byte, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, youTubeMusicFetchTimeout)
		defer cancel()
	}

	//nolint:wrapcheck // HTTP exchange spec supplies request/status/read context.
	return adapterutil.FetchBytes(ctx, adapterutil.BytesRequest{
		RequestSpec: adapterutil.RequestSpec{
			Client:         a.client,
			URL:            requestURL,
			UserAgent:      adapterutil.BrowserUserAgent,
			BuildError:     "build youtube music request",
			ExecuteError:   "execute youtube music request",
			StatusError:    adapterutil.StatusError(errUnexpectedYouTubeMusicStatus),
			ErrorBodyLimit: maxYouTubeMusicErrorResponseBytes,
		},
		ReadError:     "read youtube music response",
		MaxBodyBytes:  maxYouTubeMusicResponseBytes,
		TooLargeError: errYouTubeMusicResponseTooLarge,
	})
}
