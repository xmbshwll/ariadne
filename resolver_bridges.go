package ariadne

import "context"

type fatalAdapterParseError struct {
	error
}

func (fatalAdapterParseError) FatalParseFailure() bool {
	return true
}

func (e fatalAdapterParseError) Unwrap() error {
	return e.error
}

type sourceAdapterBridge struct {
	source SourceAdapter
}

func (b sourceAdapterBridge) Service() ServiceName {
	return b.source.Service()
}

func (b sourceAdapterBridge) ParseAlbumURL(raw string) (*ParsedAlbumURL, error) {
	parsed, err := b.source.ParseAlbumURL(raw)
	if err != nil {
		return nil, err
	}
	if parsed == nil {
		return nil, fatalAdapterParseError{ErrSourceAdapterReturnedNilParsedURL}
	}
	return parsed, nil
}

func (b sourceAdapterBridge) FetchAlbum(ctx context.Context, parsed ParsedAlbumURL) (*CanonicalAlbum, error) {
	album, err := b.source.FetchAlbum(ctx, parsed)
	if err != nil {
		return nil, err
	}
	if album == nil {
		return nil, ErrSourceAdapterReturnedNilAlbum
	}
	return album, nil
}

type songSourceAdapterBridge struct {
	source SongSourceAdapter
}

func (b songSourceAdapterBridge) Service() ServiceName {
	return b.source.Service()
}

func (b songSourceAdapterBridge) ParseSongURL(raw string) (*ParsedSongURL, error) {
	parsed, err := b.source.ParseSongURL(raw)
	if err != nil {
		return nil, err
	}
	if parsed == nil {
		return nil, fatalAdapterParseError{ErrSourceAdapterReturnedNilParsedURL}
	}
	return parsed, nil
}

func (b songSourceAdapterBridge) FetchSong(ctx context.Context, parsed ParsedSongURL) (*CanonicalSong, error) {
	song, err := b.source.FetchSong(ctx, parsed)
	if err != nil {
		return nil, err
	}
	if song == nil {
		return nil, ErrSourceAdapterReturnedNilSong
	}
	return song, nil
}

type targetAdapterBridge struct {
	target TargetAdapter
}

func (b targetAdapterBridge) Service() ServiceName {
	return b.target.Service()
}

func (b targetAdapterBridge) SearchByUPC(ctx context.Context, upc string) ([]CandidateAlbum, error) {
	return b.target.SearchByUPC(ctx, upc)
}

func (b targetAdapterBridge) SearchByISRC(ctx context.Context, isrcs []string) ([]CandidateAlbum, error) {
	return b.target.SearchByISRC(ctx, isrcs)
}

func (b targetAdapterBridge) SearchByMetadata(ctx context.Context, album CanonicalAlbum) ([]CandidateAlbum, error) {
	return b.target.SearchByMetadata(ctx, album)
}

type songTargetAdapterBridge struct {
	target SongTargetAdapter
}

func (b songTargetAdapterBridge) Service() ServiceName {
	return b.target.Service()
}

func (b songTargetAdapterBridge) SearchSongByISRC(ctx context.Context, isrc string) ([]CandidateSong, error) {
	return b.target.SearchSongByISRC(ctx, isrc)
}

func (b songTargetAdapterBridge) SearchSongByMetadata(ctx context.Context, song CanonicalSong) ([]CandidateSong, error) {
	return b.target.SearchSongByMetadata(ctx, song)
}
