package ariadne

import (
	"context"

	"github.com/stretchr/testify/mock"
)

func newLibrarySourceAdapter() SourceAdapter {
	adapter := new(MockSourceAdapter)
	adapter.EXPECT().Service().Return(ServiceDeezer)
	adapter.EXPECT().ParseAlbumURL(mock.Anything).RunAndReturn(func(raw string) (*ParsedURL, error) {
		if raw != testLibrarySourceURL {
			return nil, errUnsupportedLibrarySource
		}
		return &ParsedURL{Service: ServiceDeezer, EntityType: "album", ID: "src-1", CanonicalURL: raw, RawURL: raw}, nil
	})
	adapter.EXPECT().FetchAlbum(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, parsed ParsedURL) (*CanonicalAlbum, error) {
		return &CanonicalAlbum{
			Service:           parsed.Service,
			SourceID:          parsed.ID,
			SourceURL:         parsed.CanonicalURL,
			Title:             "Fixture Album",
			NormalizedTitle:   "fixture album",
			Artists:           []string{"Fixture Artist"},
			NormalizedArtists: []string{"fixture artist"},
			UPC:               "123456789012",
			TrackCount:        2,
			Tracks:            []CanonicalTrack{{Title: "Alpha", NormalizedTitle: "alpha", ISRC: "ISRC001"}, {Title: "Beta", NormalizedTitle: "beta"}},
		}, nil
	})
	return adapter
}

func newNilParsedSourceAdapter() SourceAdapter {
	adapter := new(MockSourceAdapter)
	adapter.EXPECT().Service().Return(ServiceDeezer)
	adapter.EXPECT().ParseAlbumURL(mock.Anything).RunAndReturn(func(raw string) (*ParsedURL, error) {
		if raw != testLibrarySourceURL {
			return nil, errUnsupportedLibrarySource
		}
		return nil, nil
	})
	adapter.EXPECT().FetchAlbum(mock.Anything, mock.Anything).Return(nil, nil)
	return adapter
}

func newNilAlbumSourceAdapter() SourceAdapter {
	adapter := new(MockSourceAdapter)
	adapter.EXPECT().Service().Return(ServiceDeezer)
	adapter.EXPECT().ParseAlbumURL(mock.Anything).RunAndReturn(func(raw string) (*ParsedURL, error) {
		if raw != testLibrarySourceURL {
			return nil, errUnsupportedLibrarySource
		}
		return &ParsedURL{Service: ServiceDeezer, EntityType: "album", ID: "src-1", CanonicalURL: raw, RawURL: raw}, nil
	})
	adapter.EXPECT().FetchAlbum(mock.Anything, mock.Anything).Return(nil, nil)
	return adapter
}

func newLibraryTargetAdapter() TargetAdapter {
	adapter := new(MockTargetAdapter)
	adapter.EXPECT().Service().Return(ServiceSpotify)
	adapter.EXPECT().SearchByUPC(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, upc string) ([]CandidateAlbum, error) {
		if upc == "" {
			return nil, nil
		}
		return []CandidateAlbum{{
			CanonicalAlbum: CanonicalAlbum{
				Service:           ServiceSpotify,
				SourceID:          "spotify-1",
				SourceURL:         "https://open.spotify.com/album/spotify-1",
				Title:             "Fixture Album",
				NormalizedTitle:   "fixture album",
				Artists:           []string{"Fixture Artist"},
				NormalizedArtists: []string{"fixture artist"},
				UPC:               upc,
				TrackCount:        2,
				Tracks:            []CanonicalTrack{{Title: "Alpha", NormalizedTitle: "alpha", ISRC: "ISRC001"}, {Title: "Beta", NormalizedTitle: "beta"}},
			},
			CandidateID: "spotify-1",
			MatchURL:    "https://open.spotify.com/album/spotify-1",
		}}, nil
	})
	adapter.EXPECT().SearchByISRC(mock.Anything, mock.Anything).Return(nil, nil)
	adapter.EXPECT().SearchByMetadata(mock.Anything, mock.Anything).Return(nil, nil)
	return adapter
}

func newFailingLibraryTargetAdapter() TargetAdapter {
	adapter := new(MockTargetAdapter)
	adapter.EXPECT().Service().Return(ServiceSpotify)
	adapter.EXPECT().SearchByUPC(mock.Anything, mock.Anything).Return(nil, nil)
	adapter.EXPECT().SearchByISRC(mock.Anything, mock.Anything).Return(nil, nil)
	adapter.EXPECT().SearchByMetadata(mock.Anything, mock.Anything).Return(nil, errLibraryTargetBoom)
	return adapter
}

func newLibrarySongSourceAdapter() SongSourceAdapter {
	adapter := new(MockSongSourceAdapter)
	adapter.EXPECT().Service().Return(ServiceSpotify)
	adapter.EXPECT().ParseSongURL(mock.Anything).RunAndReturn(func(raw string) (*ParsedURL, error) {
		if raw != "https://fixture.test/songs/1" {
			return nil, errUnsupportedLibrarySource
		}
		return &ParsedURL{Service: ServiceSpotify, EntityType: "song", ID: "song-1", CanonicalURL: raw, RawURL: raw}, nil
	})
	adapter.EXPECT().FetchSong(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, parsed ParsedURL) (*CanonicalSong, error) {
		return &CanonicalSong{
			Service:              parsed.Service,
			SourceID:             parsed.ID,
			SourceURL:            parsed.CanonicalURL,
			Title:                "Fixture Song",
			NormalizedTitle:      "fixture song",
			Artists:              []string{"Fixture Artist"},
			NormalizedArtists:    []string{"fixture artist"},
			DurationMS:           180000,
			ISRC:                 "ISRCSONG001",
			TrackNumber:          1,
			AlbumTitle:           "Fixture Album",
			AlbumNormalizedTitle: "fixture album",
		}, nil
	})
	return adapter
}

func newNilSongSourceAdapter() SongSourceAdapter {
	adapter := new(MockSongSourceAdapter)
	adapter.EXPECT().Service().Return(ServiceSpotify)
	adapter.EXPECT().ParseSongURL(mock.Anything).RunAndReturn(func(raw string) (*ParsedURL, error) {
		if raw != "https://fixture.test/songs/1" {
			return nil, errUnsupportedLibrarySource
		}
		return &ParsedURL{Service: ServiceSpotify, EntityType: "song", ID: "song-1", CanonicalURL: raw, RawURL: raw}, nil
	})
	adapter.EXPECT().FetchSong(mock.Anything, mock.Anything).Return(nil, nil)
	return adapter
}

func newLibrarySongTargetAdapter() SongTargetAdapter {
	adapter := new(MockSongTargetAdapter)
	adapter.EXPECT().Service().Return(ServiceAppleMusic)
	adapter.EXPECT().SearchSongByISRC(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, isrc string) ([]CandidateSong, error) {
		if isrc == "" {
			return nil, nil
		}
		return []CandidateSong{{
			CanonicalSong: CanonicalSong{
				Service:              ServiceAppleMusic,
				SourceID:             "apple-song-1",
				SourceURL:            "https://music.apple.com/us/song/apple-song-1",
				Title:                "Fixture Song",
				NormalizedTitle:      "fixture song",
				Artists:              []string{"Fixture Artist"},
				NormalizedArtists:    []string{"fixture artist"},
				DurationMS:           180100,
				ISRC:                 isrc,
				TrackNumber:          1,
				AlbumTitle:           "Fixture Album",
				AlbumNormalizedTitle: "fixture album",
			},
			CandidateID: "apple-song-1",
			MatchURL:    "https://music.apple.com/us/song/apple-song-1",
		}}, nil
	})
	adapter.EXPECT().SearchSongByMetadata(mock.Anything, mock.Anything).Return(nil, nil)
	return adapter
}
