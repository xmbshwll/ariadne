package ariadne_test

import (
	"context"

	ariadne "github.com/xmbshwll/ariadne"

	"github.com/stretchr/testify/mock"

	"github.com/xmbshwll/ariadne/internal/mocks"
	"github.com/xmbshwll/ariadne/internal/model"
)

func newLibrarySourceAdapter() ariadne.SourceAdapter {
	adapter := new(mocks.MockSourceAdapter)
	adapter.EXPECT().Service().Return(ariadne.ServiceDeezer)
	adapter.EXPECT().ParseAlbumURL(mock.Anything).RunAndReturn(func(raw string) (*ariadne.ParsedURL, error) {
		if raw != testLibrarySourceURL {
			return nil, errUnsupportedLibrarySource
		}
		return &ariadne.ParsedURL{Service: ariadne.ServiceDeezer, EntityType: model.EntityTypeAlbum, ID: "src-1", CanonicalURL: raw, RawURL: raw}, nil
	})
	adapter.EXPECT().FetchAlbum(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, parsed ariadne.ParsedURL) (*ariadne.CanonicalAlbum, error) {
		return &ariadne.CanonicalAlbum{
			Service:           parsed.Service,
			SourceID:          parsed.ID,
			SourceURL:         parsed.CanonicalURL,
			Title:             "Fixture Album",
			NormalizedTitle:   "fixture album",
			Artists:           []string{"Fixture Artist"},
			NormalizedArtists: []string{"fixture artist"},
			UPC:               "123456789012",
			TrackCount:        2,
			Tracks:            []ariadne.CanonicalTrack{{Title: "Alpha", NormalizedTitle: "alpha", ISRC: "ISRC001"}, {Title: "Beta", NormalizedTitle: "beta"}},
		}, nil
	})
	return adapter
}

func newNilParsedSourceAdapter() ariadne.SourceAdapter {
	adapter := new(mocks.MockSourceAdapter)
	adapter.EXPECT().Service().Return(ariadne.ServiceDeezer)
	adapter.EXPECT().ParseAlbumURL(mock.Anything).RunAndReturn(func(raw string) (*ariadne.ParsedURL, error) {
		if raw != testLibrarySourceURL {
			return nil, errUnsupportedLibrarySource
		}
		return nil, nil //nolint:nilnil // Exercise adapter contract validation for invalid nil parsed URLs.
	})
	return adapter
}

func newNilAlbumSourceAdapter() ariadne.SourceAdapter {
	adapter := new(mocks.MockSourceAdapter)
	adapter.EXPECT().Service().Return(ariadne.ServiceDeezer)
	adapter.EXPECT().ParseAlbumURL(mock.Anything).RunAndReturn(func(raw string) (*ariadne.ParsedURL, error) {
		if raw != testLibrarySourceURL {
			return nil, errUnsupportedLibrarySource
		}
		return &ariadne.ParsedURL{Service: ariadne.ServiceDeezer, EntityType: model.EntityTypeAlbum, ID: "src-1", CanonicalURL: raw, RawURL: raw}, nil
	})
	adapter.EXPECT().FetchAlbum(mock.Anything, mock.Anything).Return(nil, nil)
	return adapter
}

func newLibraryTargetAdapter() ariadne.TargetAdapter {
	adapter := new(mocks.MockAlbumTargetSearcher)
	adapter.EXPECT().Service().Return(ariadne.ServiceSpotify)
	adapter.EXPECT().SearchByUPC(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, upc string) ([]ariadne.CandidateAlbum, error) {
		if upc == "" {
			return nil, nil
		}
		return []ariadne.CandidateAlbum{{
			CanonicalAlbum: ariadne.CanonicalAlbum{
				Service:           ariadne.ServiceSpotify,
				SourceID:          "spotify-1",
				SourceURL:         "https://open.spotify.com/album/spotify-1",
				Title:             "Fixture Album",
				NormalizedTitle:   "fixture album",
				Artists:           []string{"Fixture Artist"},
				NormalizedArtists: []string{"fixture artist"},
				UPC:               upc,
				TrackCount:        2,
				Tracks:            []ariadne.CanonicalTrack{{Title: "Alpha", NormalizedTitle: "alpha", ISRC: "ISRC001"}, {Title: "Beta", NormalizedTitle: "beta"}},
			},
			CandidateID: "spotify-1",
			MatchURL:    "https://open.spotify.com/album/spotify-1",
		}}, nil
	})
	adapter.EXPECT().SearchByISRC(mock.Anything, mock.Anything).Return(nil, nil)
	adapter.EXPECT().SearchByMetadata(mock.Anything, mock.Anything).Return(nil, nil)
	return adapter
}

func newFailingLibraryTargetAdapter() ariadne.TargetAdapter {
	adapter := new(mocks.MockAlbumTargetSearcher)
	adapter.EXPECT().Service().Return(ariadne.ServiceSpotify)
	adapter.EXPECT().SearchByUPC(mock.Anything, mock.Anything).Return(nil, nil)
	adapter.EXPECT().SearchByISRC(mock.Anything, mock.Anything).Return(nil, nil)
	adapter.EXPECT().SearchByMetadata(mock.Anything, mock.Anything).Return(nil, errLibraryTargetBoom)
	return adapter
}

func newLibrarySongSourceAdapter() ariadne.SongSourceAdapter {
	adapter := new(mocks.MockSongSourceAdapter)
	adapter.EXPECT().Service().Return(ariadne.ServiceSpotify)
	adapter.EXPECT().ParseSongURL(mock.Anything).RunAndReturn(func(raw string) (*ariadne.ParsedURL, error) {
		if raw != "https://fixture.test/songs/1" {
			return nil, errUnsupportedLibrarySource
		}
		return &ariadne.ParsedURL{Service: ariadne.ServiceSpotify, EntityType: model.EntityTypeSong, ID: "song-1", CanonicalURL: raw, RawURL: raw}, nil
	})
	adapter.EXPECT().FetchSong(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, parsed ariadne.ParsedURL) (*ariadne.CanonicalSong, error) {
		return &ariadne.CanonicalSong{
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

func newNilSongSourceAdapter() ariadne.SongSourceAdapter {
	adapter := new(mocks.MockSongSourceAdapter)
	adapter.EXPECT().Service().Return(ariadne.ServiceSpotify)
	adapter.EXPECT().ParseSongURL(mock.Anything).RunAndReturn(func(raw string) (*ariadne.ParsedURL, error) {
		if raw != "https://fixture.test/songs/1" {
			return nil, errUnsupportedLibrarySource
		}
		return &ariadne.ParsedURL{Service: ariadne.ServiceSpotify, EntityType: model.EntityTypeSong, ID: "song-1", CanonicalURL: raw, RawURL: raw}, nil
	})
	adapter.EXPECT().FetchSong(mock.Anything, mock.Anything).Return(nil, nil)
	return adapter
}

func newLibrarySongTargetAdapter() ariadne.SongTargetAdapter {
	adapter := new(mocks.MockSongTargetSearcher)
	adapter.EXPECT().Service().Return(ariadne.ServiceAppleMusic)
	adapter.EXPECT().SearchSongByISRC(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, isrc string) ([]ariadne.CandidateSong, error) {
		if isrc == "" {
			return nil, nil
		}
		return []ariadne.CandidateSong{{
			CanonicalSong: ariadne.CanonicalSong{
				Service:              ariadne.ServiceAppleMusic,
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
