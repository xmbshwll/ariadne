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
	//nolint:wrapcheck // HTTP exchange spec supplies request/status/read context.
	return adapterutil.FetchBytes(ctx, adapterutil.BytesRequest{
		RequestSpec: adapterutil.RequestSpec{
			Client:       a.client,
			URL:          requestURL,
			UserAgent:    adapterutil.BrowserUserAgent,
			BuildError:   "build soundcloud request",
			ExecuteError: "execute soundcloud request",
			StatusError:  adapterutil.StatusError(errUnexpectedSoundCloudStatus),
		},
		ReadError: "read soundcloud response",
	})
}

func extractPlaylistHydration(body []byte, canonicalURL string) (*soundPlaylist, error) {
	entries, err := extractHydrationEntries(body)
	if err != nil {
		return nil, err
	}
	var firstDecodeErr error
	for _, entry := range entries {
		if entry.Hydratable != "playlist" {
			continue
		}
		var playlist soundPlaylist
		if err := json.Unmarshal(entry.Data, &playlist); err != nil {
			if firstDecodeErr == nil {
				firstDecodeErr = fmt.Errorf("decode soundcloud playlist hydration: %w", err)
			}
			continue
		}
		if playlist.PermalinkURL == "" {
			continue
		}
		if canonicalizeSoundCloudURL(playlist.PermalinkURL) == canonicalURL {
			return &playlist, nil
		}
	}
	if firstDecodeErr != nil {
		return nil, firstDecodeErr
	}
	return nil, errSoundCloudPlaylistNotFound
}

func extractTrackHydration(body []byte, canonicalURL string) (*soundTrack, error) {
	entries, err := extractHydrationEntries(body)
	if err != nil {
		return nil, err
	}
	var firstDecodeErr error
	for _, entry := range entries {
		if entry.Hydratable != "sound" {
			continue
		}
		var track soundTrack
		if err := json.Unmarshal(entry.Data, &track); err != nil {
			if firstDecodeErr == nil {
				firstDecodeErr = fmt.Errorf("decode soundcloud track hydration: %w", err)
			}
			continue
		}
		if track.PermalinkURL == "" {
			continue
		}
		if canonicalizeSoundCloudURL(track.PermalinkURL) == canonicalURL {
			return &track, nil
		}
	}
	if firstDecodeErr != nil {
		return nil, firstDecodeErr
	}
	return nil, errSoundCloudTrackNotFound
}

func extractHydrationEntries(body []byte) ([]hydrationEnvelope, error) {
	matches := hydrationPattern.FindSubmatch(body)
	if len(matches) != 2 {
		return nil, errSoundCloudHydrationNotFound
	}
	var entries []hydrationEnvelope
	if err := json.Unmarshal(matches[1], &entries); err != nil {
		return nil, fmt.Errorf("decode soundcloud hydration payload: %w", err)
	}
	return entries, nil
}
