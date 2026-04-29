package resolve

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/xmbshwll/ariadne/internal/model"
)

func newSourceAdapterMock(service model.ServiceName) *MockSourceAdapter {
	adapter := new(MockSourceAdapter)
	adapter.EXPECT().Service().Return(service)
	return adapter
}

func newTargetAdapterMock(service model.ServiceName) *MockTargetAdapter {
	adapter := new(MockTargetAdapter)
	adapter.EXPECT().Service().Return(service)
	return adapter
}

func newSongSourceAdapterMock(service model.ServiceName) *MockSongSourceAdapter {
	adapter := new(MockSongSourceAdapter)
	adapter.EXPECT().Service().Return(service)
	return adapter
}

func newSongTargetAdapterMock(service model.ServiceName) *MockSongTargetAdapter {
	adapter := new(MockSongTargetAdapter)
	adapter.EXPECT().Service().Return(service)
	return adapter
}

func newStubSourceAdapter() SourceAdapter {
	adapter := newSourceAdapterMock(model.ServiceDeezer)
	adapter.EXPECT().ParseAlbumURL(mock.Anything).RunAndReturn(func(raw string) (*model.ParsedURL, error) {
		if raw != "https://www.deezer.com/album/12047952" {
			return nil, errUnsupportedTestSource
		}
		return &model.ParsedURL{Service: model.ServiceDeezer, EntityType: "album", ID: "12047952", CanonicalURL: raw, RawURL: raw}, nil
	})
	adapter.EXPECT().FetchAlbum(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, parsed model.ParsedURL) (*model.CanonicalAlbum, error) {
		return &model.CanonicalAlbum{
			Service:         parsed.Service,
			SourceID:        parsed.ID,
			SourceURL:       parsed.CanonicalURL,
			Title:           "Abbey Road (Remastered)",
			UPC:             "602547670342",
			TrackCount:      17,
			NormalizedTitle: "abbey road remastered",
			Artists:         []string{"The Beatles"},
			Tracks:          []model.CanonicalTrack{{ISRC: "GBAYE0601690", Title: "Come Together"}, {ISRC: "GBAYE0601691", Title: "Something"}},
		}, nil
	})
	return adapter
}

func newStubTargetAdapter() TargetAdapter {
	adapter := newTargetAdapterMock(model.ServiceSpotify)
	adapter.EXPECT().SearchByUPC(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, upc string) ([]model.CandidateAlbum, error) {
		if upc == "" {
			return nil, nil
		}
		return []model.CandidateAlbum{
			{
				CandidateID: "album-1",
				MatchURL:    "https://open.spotify.com/album/1",
				CanonicalAlbum: model.CanonicalAlbum{
					Service:           model.ServiceSpotify,
					SourceID:          "album-1",
					SourceURL:         "https://open.spotify.com/album/1",
					Title:             "Abbey Road (Remastered)",
					NormalizedTitle:   "abbey road remastered",
					Artists:           []string{"The Beatles"},
					NormalizedArtists: []string{"the beatles"},
					UPC:               upc,
					TrackCount:        17,
					ReleaseDate:       "2015-12-24",
					Tracks:            []model.CanonicalTrack{{ISRC: "GBAYE0601690"}, {ISRC: "GBAYE0601691"}},
				},
			},
		}, nil
	})
	adapter.EXPECT().SearchByISRC(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, isrcs []string) ([]model.CandidateAlbum, error) {
		if len(isrcs) == 0 {
			return nil, nil
		}
		return []model.CandidateAlbum{
			{
				CandidateID: "album-1",
				MatchURL:    "https://open.spotify.com/album/1",
				CanonicalAlbum: model.CanonicalAlbum{
					Service:           model.ServiceSpotify,
					SourceID:          "album-1",
					SourceURL:         "https://open.spotify.com/album/1",
					Title:             "Abbey Road (Remastered)",
					NormalizedTitle:   "abbey road remastered",
					Artists:           []string{"The Beatles"},
					NormalizedArtists: []string{"the beatles"},
					UPC:               "602547670342",
					TrackCount:        17,
					ReleaseDate:       "2015-12-24",
					Tracks:            []model.CanonicalTrack{{ISRC: "GBAYE0601690"}, {ISRC: "GBAYE0601691"}},
				},
			},
			{
				CandidateID: "album-2",
				MatchURL:    "https://open.spotify.com/album/2",
				CanonicalAlbum: model.CanonicalAlbum{
					Service:           model.ServiceSpotify,
					SourceID:          "album-2",
					SourceURL:         "https://open.spotify.com/album/2",
					Title:             "Abbey Road",
					NormalizedTitle:   "abbey road",
					Artists:           []string{"The Beatles Complete On Ukulele"},
					NormalizedArtists: []string{"the beatles complete on ukulele"},
					TrackCount:        17,
					ReleaseDate:       "2020-01-01",
					Tracks:            []model.CanonicalTrack{{ISRC: "OTHER0001"}},
				},
			},
		}, nil
	})
	adapter.EXPECT().SearchByMetadata(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, album model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
		if album.Title == "" {
			return nil, nil
		}
		return []model.CandidateAlbum{
			{
				CandidateID: "album-2",
				MatchURL:    "https://open.spotify.com/album/2",
				CanonicalAlbum: model.CanonicalAlbum{
					Service:           model.ServiceSpotify,
					SourceID:          "album-2",
					SourceURL:         "https://open.spotify.com/album/2",
					Title:             "Abbey Road",
					NormalizedTitle:   "abbey road",
					Artists:           []string{"The Beatles Complete On Ukulele"},
					NormalizedArtists: []string{"the beatles complete on ukulele"},
					TrackCount:        17,
					ReleaseDate:       "2020-01-01",
					Tracks:            []model.CanonicalTrack{{ISRC: "OTHER0001"}},
				},
			},
		}, nil
	})
	return adapter
}

func newBlockingTargetAdapter(service model.ServiceName, started chan<- struct{}, release <-chan struct{}) TargetAdapter {
	adapter := newTargetAdapterMock(service)
	adapter.EXPECT().SearchByUPC(mock.Anything, mock.Anything).Return(nil, nil)
	adapter.EXPECT().SearchByISRC(mock.Anything, mock.Anything).Return(nil, nil)
	adapter.EXPECT().SearchByMetadata(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, _ model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
		select {
		case started <- struct{}{}:
		default:
		}
		<-release
		return nil, nil
	})
	return adapter
}

func newSourceServiceTargetAdapter() TargetAdapter {
	return newTargetAdapterMock(model.ServiceDeezer)
}

func newFailingTargetAdapter() TargetAdapter {
	adapter := newTargetAdapterMock(model.ServiceSpotify)
	adapter.EXPECT().SearchByUPC(mock.Anything, mock.Anything).Return(nil, nil)
	adapter.EXPECT().SearchByISRC(mock.Anything, mock.Anything).Return(nil, nil)
	adapter.EXPECT().SearchByMetadata(mock.Anything, mock.Anything).Return(nil, errTargetSearchBoom)
	return adapter
}

func newFixtureSourceAdapter(albumsByURL map[string]model.CanonicalAlbum) SourceAdapter {
	adapter := newSourceAdapterMock("fixture")
	adapter.EXPECT().ParseAlbumURL(mock.Anything).RunAndReturn(func(raw string) (*model.ParsedURL, error) {
		album, ok := albumsByURL[raw]
		if !ok {
			return nil, errUnsupportedTestSource
		}
		return &model.ParsedURL{Service: album.Service, EntityType: "album", ID: album.SourceID, CanonicalURL: raw, RawURL: raw}, nil
	})
	adapter.EXPECT().FetchAlbum(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, parsed model.ParsedURL) (*model.CanonicalAlbum, error) {
		album, ok := albumsByURL[parsed.RawURL]
		if !ok {
			return nil, errTestSourceNotFound
		}
		albumCopy := album
		return &albumCopy, nil
	})
	return adapter
}

func newSingleAlbumSourceAdapter(inputURL string, album model.CanonicalAlbum) SourceAdapter {
	return newFixtureSourceAdapter(map[string]model.CanonicalAlbum{inputURL: album})
}

func newNilAlbumSourceAdapter() SourceAdapter {
	adapter := newSourceAdapterMock(model.ServiceDeezer)
	adapter.EXPECT().ParseAlbumURL(mock.Anything).RunAndReturn(func(raw string) (*model.ParsedURL, error) {
		if raw != "https://www.deezer.com/album/12047952" {
			return nil, errUnsupportedTestSource
		}
		return &model.ParsedURL{Service: model.ServiceDeezer, EntityType: "album", ID: "12047952", CanonicalURL: raw, RawURL: raw}, nil
	})
	adapter.EXPECT().FetchAlbum(mock.Anything, mock.Anything).Return(nil, nil)
	return adapter
}

func newFixtureTargetAdapter(service model.ServiceName, candidatesBySourceID map[string][]model.CandidateAlbum) TargetAdapter {
	adapter := newTargetAdapterMock(service)
	adapter.EXPECT().SearchByMetadata(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, album model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
		return append([]model.CandidateAlbum(nil), candidatesBySourceID[album.SourceID]...), nil
	})
	return adapter
}

func newStubSongSourceAdapter() SongSourceAdapter {
	adapter := newSongSourceAdapterMock(model.ServiceSpotify)
	adapter.EXPECT().ParseSongURL(mock.Anything).RunAndReturn(func(raw string) (*model.ParsedURL, error) {
		if raw != "https://open.spotify.com/track/track-1" {
			return nil, errUnsupportedTestSource
		}
		return &model.ParsedURL{Service: model.ServiceSpotify, EntityType: "song", ID: "track-1", CanonicalURL: raw, RawURL: raw}, nil
	})
	adapter.EXPECT().FetchSong(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, parsed model.ParsedURL) (*model.CanonicalSong, error) {
		return &model.CanonicalSong{Service: parsed.Service, SourceID: parsed.ID, SourceURL: parsed.CanonicalURL, Title: "Come Together", NormalizedTitle: "come together", Artists: []string{"The Beatles"}, NormalizedArtists: []string{"the beatles"}, DurationMS: 259000, ISRC: "GBAYE0601690", TrackNumber: 1, AlbumTitle: "Abbey Road (Remastered)", AlbumNormalizedTitle: "abbey road remastered", ReleaseDate: "1969-09-26", EditionHints: []string{"remastered"}}, nil
	})
	return adapter
}

func newNilSongSourceAdapter() SongSourceAdapter {
	adapter := newSongSourceAdapterMock(model.ServiceSpotify)
	adapter.EXPECT().ParseSongURL(mock.Anything).RunAndReturn(func(raw string) (*model.ParsedURL, error) {
		if raw != "https://open.spotify.com/track/track-1" {
			return nil, errUnsupportedTestSource
		}
		return &model.ParsedURL{Service: model.ServiceSpotify, EntityType: "song", ID: "track-1", CanonicalURL: raw, RawURL: raw}, nil
	})
	adapter.EXPECT().FetchSong(mock.Anything, mock.Anything).Return(nil, nil)
	return adapter
}

func newStubSongTargetAdapter() SongTargetAdapter {
	adapter := newSongTargetAdapterMock(model.ServiceAppleMusic)
	adapter.EXPECT().SearchSongByISRC(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, isrc string) ([]model.CandidateSong, error) {
		if isrc == "" {
			return nil, nil
		}
		return []model.CandidateSong{{CandidateID: "song-1", MatchURL: "https://music.apple.com/us/song/1", CanonicalSong: model.CanonicalSong{Service: model.ServiceAppleMusic, SourceID: "song-1", SourceURL: "https://music.apple.com/us/song/1", Title: "Come Together", NormalizedTitle: "come together", Artists: []string{"The Beatles"}, NormalizedArtists: []string{"the beatles"}, DurationMS: 258947, ISRC: isrc, TrackNumber: 1, AlbumTitle: "Abbey Road (Remastered)", AlbumNormalizedTitle: "abbey road remastered", ReleaseDate: "1969-09-26", EditionHints: []string{"remastered"}}}}, nil
	})
	adapter.EXPECT().SearchSongByMetadata(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, song model.CanonicalSong) ([]model.CandidateSong, error) {
		if song.Title == "" {
			return nil, nil
		}
		return []model.CandidateSong{
			{CandidateID: "song-1", MatchURL: "https://music.apple.com/us/song/1", CanonicalSong: model.CanonicalSong{Service: model.ServiceAppleMusic, SourceID: "song-1", SourceURL: "https://music.apple.com/us/song/1", Title: "Come Together", NormalizedTitle: "come together", Artists: []string{"The Beatles"}, NormalizedArtists: []string{"the beatles"}, DurationMS: 258947, ISRC: "GBAYE0601690", TrackNumber: 1, AlbumTitle: "Abbey Road (Remastered)", AlbumNormalizedTitle: "abbey road remastered", ReleaseDate: "1969-09-26", EditionHints: []string{"remastered"}}},
			{CandidateID: "song-2", MatchURL: "https://music.apple.com/us/song/2", CanonicalSong: model.CanonicalSong{Service: model.ServiceAppleMusic, SourceID: "song-2", SourceURL: "https://music.apple.com/us/song/2", Title: "Come Together - Live", NormalizedTitle: "come together live", Artists: []string{"Tribute Band"}, NormalizedArtists: []string{"tribute band"}, DurationMS: 310000, ISRC: "OTHER0001", TrackNumber: 8, AlbumTitle: "Abbey Road Live", AlbumNormalizedTitle: "abbey road live", ReleaseDate: "2020-01-01", EditionHints: []string{"live"}}},
		}, nil
	})
	return adapter
}

func newSourceServiceSongTargetAdapter() SongTargetAdapter {
	return newSongTargetAdapterMock(model.ServiceSpotify)
}

func newFailingSongTargetAdapter() SongTargetAdapter {
	adapter := newSongTargetAdapterMock(model.ServiceAppleMusic)
	adapter.EXPECT().SearchSongByISRC(mock.Anything, mock.Anything).Return(nil, nil)
	adapter.EXPECT().SearchSongByMetadata(mock.Anything, mock.Anything).Return(nil, errTargetSearchBoom)
	return adapter
}
