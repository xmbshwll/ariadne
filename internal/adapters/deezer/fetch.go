package deezer

import (
	"context"
	"fmt"
	"strconv"

	"github.com/xmbshwll/ariadne/internal/httpx"
	"github.com/xmbshwll/ariadne/internal/model"
)

// FetchAlbum loads a Deezer album and its tracks, then converts them into the canonical model.
func (a *Adapter) FetchAlbum(ctx context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalAlbum, error) {
	if parsed.Service != model.ServiceDeezer {
		return nil, fmt.Errorf("%w: %s", errUnexpectedDeezerService, parsed.Service)
	}

	return a.fetchAlbumByID(ctx, parsed.ID)
}

// FetchSong loads a Deezer track and converts it into the canonical song model.
func (a *Adapter) FetchSong(ctx context.Context, parsed model.ParsedURL) (*model.CanonicalSong, error) {
	if parsed.Service != model.ServiceDeezer {
		return nil, fmt.Errorf("%w: %s", errUnexpectedDeezerService, parsed.Service)
	}
	return a.fetchSongByID(ctx, parsed.ID)
}

func (a *Adapter) fetchAlbumByID(ctx context.Context, albumID string) (*model.CanonicalAlbum, error) {
	parsed := model.ParsedAlbumURL{
		Service:      model.ServiceDeezer,
		EntityType:   model.EntityTypeAlbum,
		ID:           albumID,
		CanonicalURL: canonicalAlbumURLString(albumID),
		RawURL:       canonicalAlbumURLString(albumID),
	}
	return a.fetchAlbumByLookup(ctx, a.baseURL+"/album/"+albumID, parsed)
}

func (a *Adapter) fetchSongByID(ctx context.Context, trackID string) (*model.CanonicalSong, error) {
	track, err := a.fetchTrackLookup(ctx, a.baseURL+"/track/"+trackID)
	if err != nil {
		return nil, fmt.Errorf("fetch deezer song %s: %w", trackID, err)
	}
	return a.toCanonicalSong(*track), nil
}

func (a *Adapter) fetchTrackLookup(ctx context.Context, endpoint string) (*trackLookupResponse, error) {
	var track trackLookupResponse
	if err := a.getJSON(ctx, endpoint, &track); err != nil {
		return nil, err
	}
	if track.ID == 0 {
		return nil, errDeezerTrackNotFound
	}
	return &track, nil
}

func (a *Adapter) fetchAlbumByLookup(ctx context.Context, endpoint string, parsedOverride ...model.ParsedAlbumURL) (*model.CanonicalAlbum, error) {
	var album AlbumResponse
	if err := a.getJSON(ctx, endpoint, &album); err != nil {
		return nil, err
	}

	if album.ID == 0 {
		return nil, errDeezerAlbumNotFound
	}

	tracks := album.Tracks
	if len(tracks.Data) == 0 && album.TracklistURL != "" {
		if err := a.getJSON(ctx, album.TracklistURL, &tracks); err != nil {
			return nil, fmt.Errorf("fetch deezer album tracks %d: %w", album.ID, err)
		}
	}

	parsed := model.ParsedAlbumURL{
		Service:      model.ServiceDeezer,
		EntityType:   model.EntityTypeAlbum,
		ID:           strconv.Itoa(album.ID),
		CanonicalURL: canonicalAlbumURL(album.ID),
		RawURL:       canonicalAlbumURL(album.ID),
	}
	if len(parsedOverride) > 0 {
		parsed = parsedOverride[0]
	}

	return a.ToCanonicalAlbum(parsed, album, tracks), nil
}

func (a *Adapter) getJSON(ctx context.Context, endpoint string, target any) error {
	//nolint:wrapcheck // HTTP exchange spec supplies request/status/decode context.
	return httpx.GetJSON(ctx, httpx.JSONRequest{
		RequestSpec: httpx.RequestSpec{
			Client:       a.client,
			URL:          endpoint,
			UserAgent:    httpx.DefaultUserAgent,
			BuildError:   "build deezer request",
			ExecuteError: "execute deezer request",
			StatusError:  httpx.StatusError(errUnexpectedDeezerStatus),
		},
		DecodeError:       "decode deezer response",
		MalformedResponse: ErrMalformedDeezerResponse,
	}, target)
}
