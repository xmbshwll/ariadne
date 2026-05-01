package bandcamp

import (
	"context"
	"errors"
	"fmt"

	"github.com/xmbshwll/ariadne/internal/adapters/adapterutil"
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/parse"
)

const maxBandcampResponseBytes = 10 << 20

var errBandcampResponseTooLarge = errors.New("bandcamp response too large")

// FetchAlbum loads a Bandcamp album page and extracts canonical metadata from schema.org JSON-LD.
func (a *Adapter) FetchAlbum(ctx context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalAlbum, error) {
	if parsed.Service != model.ServiceBandcamp {
		return nil, fmt.Errorf("%w: %s", errUnexpectedBandcampService, parsed.Service)
	}
	return a.fetchAlbumPage(ctx, parsed.CanonicalURL)
}

// FetchSong loads a Bandcamp track page and extracts canonical metadata from schema.org JSON-LD.
func (a *Adapter) FetchSong(ctx context.Context, parsed model.ParsedURL) (*model.CanonicalSong, error) {
	if parsed.Service != model.ServiceBandcamp {
		return nil, fmt.Errorf("%w: %s", errUnexpectedBandcampService, parsed.Service)
	}
	return a.fetchSongPage(ctx, parsed.CanonicalURL)
}

func (a *Adapter) fetchAlbumPage(ctx context.Context, rawURL string) (*model.CanonicalAlbum, error) {
	return fetchCanonicalPage(a, ctx, rawURL, "album", parse.BandcampAlbumURL, toCanonicalAlbum)
}

func (a *Adapter) fetchSongPage(ctx context.Context, rawURL string) (*model.CanonicalSong, error) {
	return fetchCanonicalPage(a, ctx, rawURL, "song", parse.BandcampSongURL, toCanonicalSong)
}

func fetchCanonicalPage[Canonical any](adapter *Adapter, ctx context.Context, rawURL, entity string, parseURL func(string) (*model.ParsedURL, error), toCanonical func(model.ParsedURL, *schemaAlbum) *Canonical) (*Canonical, error) {
	parsed, err := parseURL(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse bandcamp %s url: %w", entity, err)
	}

	body, err := adapter.fetchPage(ctx, parsed.CanonicalURL)
	if err != nil {
		return nil, fmt.Errorf("fetch bandcamp %s page: %w", entity, err)
	}

	schema, err := extractSchema(body)
	if err != nil {
		return nil, fmt.Errorf("extract bandcamp schema %s: %w", entity, err)
	}
	return toCanonical(*parsed, schema), nil
}

func (a *Adapter) fetchPage(ctx context.Context, requestURL string) ([]byte, error) {
	request := adapterutil.PageRequest{
		BytesRequest: adapterutil.BytesRequest{
			RequestSpec: adapterutil.RequestSpec{
				Client:       a.client,
				URL:          requestURL,
				UserAgent:    adapterutil.DefaultUserAgent,
				BuildError:   "build bandcamp request",
				ExecuteError: "execute bandcamp request",
				StatusError:  adapterutil.StatusError(errUnexpectedBandcampStatus),
			},
			ReadError:     "read bandcamp response",
			MaxBodyBytes:  maxBandcampResponseBytes,
			TooLargeError: errBandcampResponseTooLarge,
		},
	}
	//nolint:wrapcheck // Page extraction spec supplies request/status/read context.
	return adapterutil.FetchPage(ctx, request)
}
