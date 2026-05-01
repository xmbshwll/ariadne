package soundcloud

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/xmbshwll/ariadne/internal/adapters/adapterutil"
	"github.com/xmbshwll/ariadne/internal/model"
)

func (a *Adapter) FetchAlbum(ctx context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalAlbum, error) {
	if parsed.Service != model.ServiceSoundCloud {
		return nil, fmt.Errorf("%w: %s", errUnexpectedSoundCloudService, parsed.Service)
	}
	body, err := a.fetchPage(ctx, parsed.CanonicalURL)
	if err != nil {
		return nil, fmt.Errorf("fetch soundcloud page: %w", err)
	}
	playlist, err := extractPlaylistHydration(body, parsed.CanonicalURL)
	if err != nil {
		return nil, fmt.Errorf("extract soundcloud playlist hydration: %w", err)
	}
	a.maybeCacheClientIDFromPage(body)
	return toCanonicalAlbum(*playlist), nil
}

func (a *Adapter) FetchSong(ctx context.Context, parsed model.ParsedURL) (*model.CanonicalSong, error) {
	if parsed.Service != model.ServiceSoundCloud {
		return nil, fmt.Errorf("%w: %s", errUnexpectedSoundCloudService, parsed.Service)
	}
	body, err := a.fetchPage(ctx, parsed.CanonicalURL)
	if err != nil {
		return nil, fmt.Errorf("fetch soundcloud page: %w", err)
	}
	track, err := extractTrackHydration(body, parsed.CanonicalURL)
	if err != nil {
		return nil, fmt.Errorf("extract soundcloud track hydration: %w", err)
	}
	a.maybeCacheClientIDFromPage(body)
	return toCanonicalSong(*track), nil
}

func (a *Adapter) fetchPage(ctx context.Context, requestURL string) ([]byte, error) {
	request := adapterutil.PageRequest{
		BytesRequest: adapterutil.BytesRequest{
			RequestSpec: adapterutil.RequestSpec{
				Client:       a.client,
				URL:          requestURL,
				UserAgent:    adapterutil.BrowserUserAgent,
				BuildError:   "build soundcloud request",
				ExecuteError: "execute soundcloud request",
				StatusError:  adapterutil.StatusError(errUnexpectedSoundCloudStatus),
			},
			ReadError: "read soundcloud response",
		},
	}
	//nolint:wrapcheck // Page extraction spec supplies request/status/read context.
	return adapterutil.FetchPage(ctx, request)
}

func extractPlaylistHydration(body []byte, canonicalURL string) (*soundPlaylist, error) {
	return extractHydrationEntity(body, canonicalURL, "playlist", errSoundCloudPlaylistNotFound, "decode soundcloud playlist hydration", func(playlist soundPlaylist) string {
		return playlist.PermalinkURL
	})
}

func extractTrackHydration(body []byte, canonicalURL string) (*soundTrack, error) {
	return extractHydrationEntity(body, canonicalURL, "sound", errSoundCloudTrackNotFound, "decode soundcloud track hydration", func(track soundTrack) string {
		return track.PermalinkURL
	})
}

func extractHydrationEntity[T any](body []byte, canonicalURL string, hydratable string, notFound error, decodeError string, permalinkURL func(T) string) (*T, error) {
	entries, err := extractHydrationEntries(body)
	if err != nil {
		return nil, err
	}
	var firstDecodeErr error
	for _, entry := range entries {
		if entry.Hydratable != hydratable {
			continue
		}
		var entity T
		if err := json.Unmarshal(entry.Data, &entity); err != nil {
			if firstDecodeErr == nil {
				firstDecodeErr = fmt.Errorf("%s: %w", decodeError, err)
			}
			continue
		}
		url := permalinkURL(entity)
		if url == "" {
			continue
		}
		if canonicalizeSoundCloudURL(url) == canonicalURL {
			return &entity, nil
		}
	}
	if firstDecodeErr != nil {
		return nil, firstDecodeErr
	}
	return nil, notFound
}

func extractHydrationEntries(body []byte) ([]hydrationEnvelope, error) {
	return adapterutil.DecodeJSONBlock[[]hydrationEnvelope](body, hydrationPattern, errSoundCloudHydrationNotFound, "decode soundcloud hydration payload", nil)
}
