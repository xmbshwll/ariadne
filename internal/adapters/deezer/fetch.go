package deezer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

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
		EntityType:   "album",
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
	var album albumResponse
	if err := a.getJSON(ctx, endpoint, &album); err != nil {
		return nil, err
	}

	if album.ID == 0 || album.TracklistURL == "" {
		return nil, errDeezerAlbumNotFound
	}

	var tracks tracksResponse
	if err := a.getJSON(ctx, album.TracklistURL, &tracks); err != nil {
		return nil, fmt.Errorf("fetch deezer album tracks %d: %w", album.ID, err)
	}

	parsed := model.ParsedAlbumURL{
		Service:      model.ServiceDeezer,
		EntityType:   "album",
		ID:           strconv.Itoa(album.ID),
		CanonicalURL: canonicalAlbumURL(album.ID),
		RawURL:       canonicalAlbumURL(album.ID),
	}
	if len(parsedOverride) > 0 {
		parsed = parsedOverride[0]
	}

	return a.toCanonicalAlbum(parsed, album, tracks), nil
}

func (a *Adapter) getJSON(ctx context.Context, endpoint string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		closeErr := resp.Body.Close()
		if closeErr != nil {
			return fmt.Errorf("unexpected status %d and close body: %w", resp.StatusCode, closeErr)
		}
		return fmt.Errorf("%w: %d", errUnexpectedDeezerStatus, resp.StatusCode)
	}

	decodeErr := json.NewDecoder(resp.Body).Decode(target)
	closeErr := resp.Body.Close()
	if decodeErr != nil {
		if closeErr != nil {
			return errors.Join(
				fmt.Errorf("decode response: %w", errors.Join(errMalformedDeezerResponse, decodeErr)),
				fmt.Errorf("close response body: %w", closeErr),
			)
		}
		return fmt.Errorf("decode response: %w", errors.Join(errMalformedDeezerResponse, decodeErr))
	}
	if closeErr != nil {
		return fmt.Errorf("close response body: %w", closeErr)
	}
	return nil
}
