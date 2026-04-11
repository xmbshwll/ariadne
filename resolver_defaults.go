package ariadne

import (
	"context"
	"net/http"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/resolve"
)

func defaultSourceAdapters(client *http.Client, config Config) []resolve.SourceAdapter {
	sets := buildDefaultServiceAdapters(client, config)
	sources := make([]resolve.SourceAdapter, 0, len(defaultServiceOrder.albumSources))
	for _, service := range defaultServiceOrder.albumSources {
		adapter := sets[service].albumSource
		if adapter == nil {
			continue
		}
		sources = append(sources, adapter)
	}
	return sources
}

func defaultTargetAdapters(client *http.Client, config Config) []resolve.TargetAdapter {
	sets := buildDefaultServiceAdapters(client, config)
	targets := make([]resolve.TargetAdapter, 0, len(defaultServiceOrder.albumTargets))
	for _, service := range defaultServiceOrder.albumTargets {
		adapter := sets[service].albumTarget
		if adapter == nil {
			continue
		}
		targets = append(targets, adapter)
	}
	return filterAdaptersByServiceName(targets, config.TargetServices)
}

func allowedTargetServices(services []ServiceName) map[ServiceName]struct{} {
	if len(services) == 0 {
		return nil
	}

	allowed := make(map[ServiceName]struct{}, len(services))
	for _, service := range services {
		allowed[service] = struct{}{}
	}
	return allowed
}

func defaultSongSourceAdapters(client *http.Client, config Config) []resolve.SongSourceAdapter {
	sets := buildDefaultServiceAdapters(client, config)
	sources := make([]resolve.SongSourceAdapter, 0, len(defaultServiceOrder.songSources))
	for _, service := range defaultServiceOrder.songSources {
		adapter := sets[service].songSource
		if adapter == nil {
			continue
		}
		sources = append(sources, adapter)
	}
	return sources
}

func defaultSongTargetAdapters(client *http.Client, config Config) []resolve.SongTargetAdapter {
	sets := buildDefaultServiceAdapters(client, config)
	targets := make([]resolve.SongTargetAdapter, 0, len(defaultServiceOrder.songTargets))
	for _, service := range defaultServiceOrder.songTargets {
		adapter := sets[service].songTarget
		if adapter == nil {
			continue
		}
		targets = append(targets, adapter)
	}
	return filterAdaptersByServiceName(targets, config.TargetServices)
}

func filterAdaptersByServiceName[T interface{ Service() model.ServiceName }](adapters []T, services []ServiceName) []T {
	allowed := allowedTargetServices(services)
	if len(allowed) == 0 {
		return adapters
	}

	filtered := make([]T, 0, len(adapters))
	for _, adapter := range adapters {
		if _, ok := allowed[fromInternalServiceName(adapter.Service())]; !ok {
			continue
		}
		filtered = append(filtered, adapter)
	}
	return filtered
}

func wrapSourceAdapters(sources []SourceAdapter) []resolve.SourceAdapter {
	wrapped := make([]resolve.SourceAdapter, 0, len(sources))
	for _, source := range sources {
		wrapped = append(wrapped, sourceAdapterBridge{source: source})
	}
	return wrapped
}

func wrapSongSourceAdapters(sources []SongSourceAdapter) []resolve.SongSourceAdapter {
	wrapped := make([]resolve.SongSourceAdapter, 0, len(sources))
	for _, source := range sources {
		wrapped = append(wrapped, songSourceAdapterBridge{source: source})
	}
	return wrapped
}

func wrapTargetAdapters(targets []TargetAdapter) []resolve.TargetAdapter {
	wrapped := make([]resolve.TargetAdapter, 0, len(targets))
	for _, target := range targets {
		wrapped = append(wrapped, targetAdapterBridge{target: target})
	}
	return wrapped
}

func wrapSongTargetAdapters(targets []SongTargetAdapter) []resolve.SongTargetAdapter {
	wrapped := make([]resolve.SongTargetAdapter, 0, len(targets))
	for _, target := range targets {
		wrapped = append(wrapped, songTargetAdapterBridge{target: target})
	}
	return wrapped
}

type sourceAdapterBridge struct {
	source SourceAdapter
}

func (b sourceAdapterBridge) Service() model.ServiceName {
	return toInternalServiceName(b.source.Service())
}

func (b sourceAdapterBridge) ParseAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	parsed, err := b.source.ParseAlbumURL(raw)
	if err != nil || parsed == nil {
		if err != nil {
			//nolint:wrapcheck // Preserve adapter parse errors without adding another wrapper layer.
			return nil, err
		}
		return nil, errSourceAdapterReturnedNilParsed
	}
	internal := toInternalParsedAlbumURL(*parsed)
	return &internal, nil
}

func (b sourceAdapterBridge) FetchAlbum(ctx context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalAlbum, error) {
	album, err := b.source.FetchAlbum(ctx, fromInternalParsedAlbumURL(parsed))
	if err != nil || album == nil {
		if err != nil {
			//nolint:wrapcheck // Preserve adapter fetch errors without adding another wrapper layer.
			return nil, err
		}
		return nil, errSourceAdapterReturnedNilAlbum
	}
	internal := toInternalCanonicalAlbum(*album)
	return &internal, nil
}

type songSourceAdapterBridge struct {
	source SongSourceAdapter
}

func (b songSourceAdapterBridge) Service() model.ServiceName {
	return toInternalServiceName(b.source.Service())
}

func (b songSourceAdapterBridge) ParseSongURL(raw string) (*model.ParsedAlbumURL, error) {
	parsed, err := b.source.ParseSongURL(raw)
	if err != nil || parsed == nil {
		if err != nil {
			//nolint:wrapcheck // Preserve adapter parse errors without adding another wrapper layer.
			return nil, err
		}
		return nil, errSourceAdapterReturnedNilParsed
	}
	internal := toInternalParsedAlbumURL(*parsed)
	return &internal, nil
}

func (b songSourceAdapterBridge) FetchSong(ctx context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalSong, error) {
	song, err := b.source.FetchSong(ctx, fromInternalParsedAlbumURL(parsed))
	if err != nil || song == nil {
		if err != nil {
			//nolint:wrapcheck // Preserve adapter fetch errors without adding another wrapper layer.
			return nil, err
		}
		return nil, errSourceAdapterReturnedNilSong
	}
	internal := toInternalCanonicalSong(*song)
	return &internal, nil
}

type targetAdapterBridge struct {
	target TargetAdapter
}

func (b targetAdapterBridge) Service() model.ServiceName {
	return toInternalServiceName(b.target.Service())
}

func (b targetAdapterBridge) SearchByUPC(ctx context.Context, upc string) ([]model.CandidateAlbum, error) {
	albums, err := b.target.SearchByUPC(ctx, upc)
	if err != nil {
		//nolint:wrapcheck // Preserve target adapter errors without adding another wrapper layer.
		return nil, err
	}
	return toInternalCandidateAlbums(albums), nil
}

func (b targetAdapterBridge) SearchByISRC(ctx context.Context, isrcs []string) ([]model.CandidateAlbum, error) {
	albums, err := b.target.SearchByISRC(ctx, append([]string(nil), isrcs...))
	if err != nil {
		//nolint:wrapcheck // Preserve target adapter errors without adding another wrapper layer.
		return nil, err
	}
	return toInternalCandidateAlbums(albums), nil
}

func (b targetAdapterBridge) SearchByMetadata(ctx context.Context, album model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
	albums, err := b.target.SearchByMetadata(ctx, fromInternalCanonicalAlbum(album))
	if err != nil {
		//nolint:wrapcheck // Preserve target adapter errors without adding another wrapper layer.
		return nil, err
	}
	return toInternalCandidateAlbums(albums), nil
}

type songTargetAdapterBridge struct {
	target SongTargetAdapter
}

func (b songTargetAdapterBridge) Service() model.ServiceName {
	return toInternalServiceName(b.target.Service())
}

func (b songTargetAdapterBridge) SearchSongByISRC(ctx context.Context, isrc string) ([]model.CandidateSong, error) {
	songs, err := b.target.SearchSongByISRC(ctx, isrc)
	if err != nil {
		//nolint:wrapcheck // Preserve target adapter errors without adding another wrapper layer.
		return nil, err
	}
	return toInternalCandidateSongs(songs), nil
}

func (b songTargetAdapterBridge) SearchSongByMetadata(ctx context.Context, song model.CanonicalSong) ([]model.CandidateSong, error) {
	songs, err := b.target.SearchSongByMetadata(ctx, fromInternalCanonicalSong(song))
	if err != nil {
		//nolint:wrapcheck // Preserve target adapter errors without adding another wrapper layer.
		return nil, err
	}
	return toInternalCandidateSongs(songs), nil
}
