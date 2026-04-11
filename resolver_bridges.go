package ariadne

import (
	"context"

	"github.com/xmbshwll/ariadne/internal/model"
)

type sourceAdapterBridge struct {
	source SourceAdapter
}

func (b sourceAdapterBridge) Service() model.ServiceName {
	return toInternalServiceName(b.source.Service())
}

func (b sourceAdapterBridge) ParseAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	parsed, err := b.source.ParseAlbumURL(raw)
	if err != nil {
		//nolint:wrapcheck // Preserve adapter parse errors without adding another wrapper layer.
		return nil, err
	}
	if parsed == nil {
		return nil, errSourceAdapterReturnedNilParsed
	}
	internal := toInternalParsedAlbumURL(*parsed)
	return &internal, nil
}

func (b sourceAdapterBridge) FetchAlbum(ctx context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalAlbum, error) {
	album, err := b.source.FetchAlbum(ctx, fromInternalParsedAlbumURL(parsed))
	if err != nil {
		//nolint:wrapcheck // Preserve adapter fetch errors without adding another wrapper layer.
		return nil, err
	}
	if album == nil {
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
	if err != nil {
		//nolint:wrapcheck // Preserve adapter parse errors without adding another wrapper layer.
		return nil, err
	}
	if parsed == nil {
		return nil, errSourceAdapterReturnedNilParsed
	}
	internal := toInternalParsedAlbumURL(*parsed)
	return &internal, nil
}

func (b songSourceAdapterBridge) FetchSong(ctx context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalSong, error) {
	song, err := b.source.FetchSong(ctx, fromInternalParsedAlbumURL(parsed))
	if err != nil {
		//nolint:wrapcheck // Preserve adapter fetch errors without adding another wrapper layer.
		return nil, err
	}
	if song == nil {
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
