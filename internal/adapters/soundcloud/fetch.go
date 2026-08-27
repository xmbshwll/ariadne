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
	playlist, err := ExtractPlaylistHydration(body, parsed.CanonicalURL)
	if err != nil {
		return nil, fmt.Errorf("extract soundcloud playlist hydration: %w", err)
	}
	a.maybeCacheClientIDFromPage(body)
	return ToCanonicalAlbum(*playlist), nil
}

func (a *Adapter) FetchSong(ctx context.Context, parsed model.ParsedURL) (*model.CanonicalSong, error) {
	if parsed.Service != model.ServiceSoundCloud {
		return nil, fmt.Errorf("%w: %s", errUnexpectedSoundCloudService, parsed.Service)
	}
	body, err := a.fetchPage(ctx, parsed.CanonicalURL)
	if err != nil {
		return nil, fmt.Errorf("fetch soundcloud page: %w", err)
	}
	track, err := ExtractTrackHydration(body, parsed.CanonicalURL)
	if err != nil {
		return nil, fmt.Errorf("extract soundcloud track hydration: %w", err)
	}
	a.maybeCacheClientIDFromPage(body)
	return ToCanonicalSong(*track), nil
}

func (a *Adapter) fetchPage(ctx context.Context, requestURL string) ([]byte, error) {
	//nolint:wrapcheck // Page fetcher supplies request/status/read context.
	return adapterutil.PageFetcher{
		Client:       a.client,
		UserAgent:    adapterutil.BrowserUserAgent,
		BuildError:   "build soundcloud request",
		ExecuteError: "execute soundcloud request",
		StatusError:  adapterutil.StatusError(errUnexpectedSoundCloudStatus),
		ReadError:    "read soundcloud response",
	}.Fetch(ctx, requestURL)
}

func ExtractPlaylistHydration(body []byte, canonicalURL string) (*soundPlaylist, error) {
	return extractHydrationEntity(
		body,
		canonicalURL,
		"playlist",
		ErrSoundCloudPlaylistNotFound,
		"decode soundcloud playlist hydration",
		func(playlist soundPlaylist) string {
			return playlist.PermalinkURL
		},
	)
}

func ExtractTrackHydration(body []byte, canonicalURL string) (*SoundTrack, error) {
	return extractHydrationEntity(
		body,
		canonicalURL,
		"sound",
		ErrSoundCloudTrackNotFound,
		"decode soundcloud track hydration",
		func(track SoundTrack) string {
			return track.PermalinkURL
		},
	)
}

func extractHydrationEntity[T any](
	body []byte,
	canonicalURL string,
	hydratable string,
	notFound error,
	decodeError string,
	permalinkURL func(T) string,
) (*T, error) {
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
	return adapterutil.DecodeJSONBlock[[]hydrationEnvelope](
		body,
		hydrationPattern,
		errSoundCloudHydrationNotFound,
		"decode soundcloud hydration payload",
		nil,
	)
}
