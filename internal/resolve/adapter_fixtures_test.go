package resolve_test

import (
	"context"

	"github.com/xmbshwll/ariadne/internal/adapters"
	"github.com/xmbshwll/ariadne/internal/mocks"

	"github.com/stretchr/testify/mock"

	"github.com/xmbshwll/ariadne/internal/model"
)

const (
	stubAlbumURL = "https://www.deezer.com/album/12047952"
	stubSongURL  = "https://open.spotify.com/track/track-1"
)

type albumSourceFixture struct {
	service  model.ServiceName
	inputURL string
	album    *model.CanonicalAlbum
	nilAlbum bool
}

type albumTargetFixture struct {
	service    model.ServiceName
	byUPC      func(context.Context, string) ([]model.CandidateAlbum, error)
	byISRC     func(context.Context, []string) ([]model.CandidateAlbum, error)
	byMetadata func(context.Context, model.CanonicalAlbum) ([]model.CandidateAlbum, error)
}

type songSourceFixture struct {
	service  model.ServiceName
	inputURL string
	song     *model.CanonicalSong
	nilSong  bool
}

type songTargetFixture struct {
	service    model.ServiceName
	byISRC     func(context.Context, string) ([]model.CandidateSong, error)
	byMetadata func(context.Context, model.CanonicalSong) ([]model.CandidateSong, error)
}

// newAdapterMock builds the one Adapter mock with the identity every fixture
// needs: the service name and the Capabilities the fixture really implements.
func newAdapterMock(service model.ServiceName, capabilities adapters.Capabilities) *mocks.MockAdapter {
	adapter := new(mocks.MockAdapter)
	adapter.EXPECT().Service().Return(service)
	adapter.EXPECT().Capabilities().Return(capabilities)
	return adapter
}

// albumTargetCapabilities is what an album Target Search fixture declares.
var albumTargetCapabilities = adapters.Capabilities{AlbumUPC: true, AlbumISRC: true, AlbumMetadata: true}

// songTargetCapabilities is what a song Target Search fixture declares.
var songTargetCapabilities = adapters.Capabilities{SongISRC: true, SongMetadata: true}

func newAlbumSourceFixture(fixture albumSourceFixture) adapters.Adapter {
	adapter := newAdapterMock(fixture.service, adapters.Capabilities{AlbumSource: true})
	adapter.EXPECT().ParseAlbumURL(mock.Anything).RunAndReturn(func(raw string) (*model.ParsedURL, error) {
		if raw != fixture.inputURL {
			return nil, errUnsupportedTestSource
		}
		return &model.ParsedURL{Service: fixture.service, EntityType: model.EntityTypeAlbum, ID: fixture.album.SourceID, CanonicalURL: raw, RawURL: raw}, nil
	})
	adapter.EXPECT().FetchAlbum(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, parsed model.ParsedURL) (*model.CanonicalAlbum, error) {
		if fixture.nilAlbum {
			return nil, nil //nolint:nilnil // Exercise source adapter nil album contract handling.
		}
		album := *fixture.album
		album.Service = parsed.Service
		album.SourceID = parsed.ID
		album.SourceURL = parsed.CanonicalURL
		return &album, nil
	})
	return adapter
}

func newAlbumTargetFixture(fixture albumTargetFixture) adapters.Adapter {
	adapter := newAdapterMock(fixture.service, albumTargetCapabilities)
	adapter.EXPECT().SearchAlbumByUPC(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, upc string) ([]model.CandidateAlbum, error) {
		if fixture.byUPC == nil {
			return nil, nil
		}
		return fixture.byUPC(ctx, upc)
	})
	adapter.EXPECT().SearchAlbumByISRC(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, isrcs []string) ([]model.CandidateAlbum, error) {
		if fixture.byISRC == nil {
			return nil, nil
		}
		return fixture.byISRC(ctx, isrcs)
	})
	adapter.EXPECT().SearchAlbumByMetadata(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, album model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
		if fixture.byMetadata == nil {
			return nil, nil
		}
		return fixture.byMetadata(ctx, album)
	})
	return adapter
}

func newSongSourceFixture(fixture songSourceFixture) adapters.Adapter {
	adapter := newAdapterMock(fixture.service, adapters.Capabilities{SongSource: true})
	adapter.EXPECT().ParseSongURL(mock.Anything).RunAndReturn(func(raw string) (*model.ParsedURL, error) {
		if raw != fixture.inputURL {
			return nil, errUnsupportedTestSource
		}
		return &model.ParsedURL{Service: fixture.service, EntityType: model.EntityTypeSong, ID: fixture.song.SourceID, CanonicalURL: raw, RawURL: raw}, nil
	})
	adapter.EXPECT().FetchSong(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, parsed model.ParsedURL) (*model.CanonicalSong, error) {
		if fixture.nilSong {
			return nil, nil //nolint:nilnil // Exercise source adapter nil song contract handling.
		}
		song := *fixture.song
		song.Service = parsed.Service
		song.SourceID = parsed.ID
		song.SourceURL = parsed.CanonicalURL
		return &song, nil
	})
	return adapter
}

func newSongTargetFixture(fixture songTargetFixture) adapters.Adapter {
	adapter := newAdapterMock(fixture.service, songTargetCapabilities)
	adapter.EXPECT().SearchSongByISRC(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, isrc string) ([]model.CandidateSong, error) {
		if fixture.byISRC == nil {
			return nil, nil
		}
		return fixture.byISRC(ctx, isrc)
	})
	adapter.EXPECT().SearchSongByMetadata(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, song model.CanonicalSong) ([]model.CandidateSong, error) {
		if fixture.byMetadata == nil {
			return nil, nil
		}
		return fixture.byMetadata(ctx, song)
	})
	return adapter
}

func newStubSourceAdapter() adapters.Adapter {
	return newAlbumSourceFixture(albumSourceFixture{
		service:  model.ServiceDeezer,
		inputURL: stubAlbumURL,
		album: &model.CanonicalAlbum{
			Title:           "Abbey Road (Remastered)",
			UPC:             "602547670342",
			TrackCount:      17,
			NormalizedTitle: "abbey road remastered",
			Artists:         []string{"The Beatles"},
			Tracks:          []model.CanonicalTrack{{ISRC: "GBAYE0601690", Title: "Come Together"}, {ISRC: "GBAYE0601691", Title: "Something"}},
		},
	})
}

func newStubTargetAdapter() adapters.Adapter {
	return newAlbumTargetFixture(albumTargetFixture{
		service: model.ServiceSpotify,
		byUPC: func(_ context.Context, upc string) ([]model.CandidateAlbum, error) {
			if upc == "" {
				return nil, nil
			}
			return []model.CandidateAlbum{stubAlbumOne(upc)}, nil
		},
		byISRC: func(_ context.Context, isrcs []string) ([]model.CandidateAlbum, error) {
			if len(isrcs) == 0 {
				return nil, nil
			}
			return []model.CandidateAlbum{stubAlbumOne("602547670342"), stubAlbumTwo()}, nil
		},
		byMetadata: func(_ context.Context, album model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
			if album.Title == "" {
				return nil, nil
			}
			return []model.CandidateAlbum{stubAlbumTwo()}, nil
		},
	})
}

func stubAlbumOne(upc string) model.CandidateAlbum {
	return model.CandidateAlbum{
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
	}
}

func stubAlbumTwo() model.CandidateAlbum {
	return model.CandidateAlbum{
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
	}
}

func newBlockingTargetAdapter(service model.ServiceName, started chan<- struct{}, release <-chan struct{}) adapters.Adapter {
	return newAlbumTargetFixture(albumTargetFixture{
		service: service,
		byMetadata: func(ctx context.Context, _ model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
			select {
			case started <- struct{}{}:
			default:
			}
			select {
			case <-release:
				return nil, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
	})
}

func newSourceServiceTargetAdapter() adapters.Adapter {
	return newAdapterMock(model.ServiceDeezer, albumTargetCapabilities)
}

func newFailingTargetAdapter() adapters.Adapter {
	return newAlbumTargetFixture(albumTargetFixture{
		service: model.ServiceBandcamp,
		byMetadata: func(context.Context, model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
			return nil, errTargetSearchBoom
		},
	})
}

func newFixtureSourceAdapter(albumsByURL map[string]model.CanonicalAlbum) adapters.Adapter {
	adapter := newAdapterMock("fixture", adapters.Capabilities{AlbumSource: true})
	adapter.EXPECT().ParseAlbumURL(mock.Anything).RunAndReturn(func(raw string) (*model.ParsedURL, error) {
		album, ok := albumsByURL[raw]
		if !ok {
			return nil, errUnsupportedTestSource
		}
		return &model.ParsedURL{Service: album.Service, EntityType: model.EntityTypeAlbum, ID: album.SourceID, CanonicalURL: raw, RawURL: raw}, nil
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

func newSingleAlbumSourceAdapter(inputURL string, album model.CanonicalAlbum) adapters.Adapter {
	return newFixtureSourceAdapter(map[string]model.CanonicalAlbum{inputURL: album})
}

func newNilAlbumSourceAdapter() adapters.Adapter {
	return newAlbumSourceFixture(albumSourceFixture{
		service:  model.ServiceDeezer,
		inputURL: stubAlbumURL,
		album:    &model.CanonicalAlbum{SourceID: "12047952"},
		nilAlbum: true,
	})
}

func newFixtureTargetAdapter(service model.ServiceName, candidatesBySourceID map[string][]model.CandidateAlbum) adapters.Adapter {
	return newAlbumTargetFixture(albumTargetFixture{
		service: service,
		byMetadata: func(_ context.Context, album model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
			return append([]model.CandidateAlbum(nil), candidatesBySourceID[album.SourceID]...), nil
		},
	})
}

func newStubSongSourceAdapter() adapters.Adapter {
	return newSongSourceFixture(songSourceFixture{
		service:  model.ServiceSpotify,
		inputURL: stubSongURL,
		song: &model.CanonicalSong{
			Title:                "Come Together",
			NormalizedTitle:      "come together",
			Artists:              []string{"The Beatles"},
			NormalizedArtists:    []string{"the beatles"},
			DurationMS:           259000,
			ISRC:                 "GBAYE0601690",
			TrackNumber:          1,
			AlbumTitle:           "Abbey Road (Remastered)",
			AlbumNormalizedTitle: "abbey road remastered",
			ReleaseDate:          "1969-09-26",
			EditionHints:         []string{"remastered"},
		},
	})
}

func newNilSongSourceAdapter() adapters.Adapter {
	return newSongSourceFixture(songSourceFixture{
		service:  model.ServiceSpotify,
		inputURL: stubSongURL,
		song:     &model.CanonicalSong{SourceID: "track-1"},
		nilSong:  true,
	})
}

func newStubSongTargetAdapter() adapters.Adapter {
	return newSongTargetFixture(songTargetFixture{
		service: model.ServiceAppleMusic,
		byISRC: func(_ context.Context, isrc string) ([]model.CandidateSong, error) {
			if isrc == "" {
				return nil, nil
			}
			return []model.CandidateSong{stubSongOne(isrc)}, nil
		},
		byMetadata: func(_ context.Context, song model.CanonicalSong) ([]model.CandidateSong, error) {
			if song.Title == "" {
				return nil, nil
			}
			return []model.CandidateSong{stubSongOne("GBAYE0601690"), stubSongTwo()}, nil
		},
	})
}

func stubSongOne(isrc string) model.CandidateSong {
	return model.CandidateSong{
		CandidateID: "song-1",
		MatchURL:    "https://music.apple.com/us/song/1",
		CanonicalSong: model.CanonicalSong{
			Service:              model.ServiceAppleMusic,
			SourceID:             "song-1",
			SourceURL:            "https://music.apple.com/us/song/1",
			Title:                "Come Together",
			NormalizedTitle:      "come together",
			Artists:              []string{"The Beatles"},
			NormalizedArtists:    []string{"the beatles"},
			DurationMS:           258947,
			ISRC:                 isrc,
			TrackNumber:          1,
			AlbumTitle:           "Abbey Road (Remastered)",
			AlbumNormalizedTitle: "abbey road remastered",
			ReleaseDate:          "1969-09-26",
			EditionHints:         []string{"remastered"},
		},
	}
}

func stubSongTwo() model.CandidateSong {
	return model.CandidateSong{
		CandidateID: "song-2",
		MatchURL:    "https://music.apple.com/us/song/2",
		CanonicalSong: model.CanonicalSong{
			Service:              model.ServiceAppleMusic,
			SourceID:             "song-2",
			SourceURL:            "https://music.apple.com/us/song/2",
			Title:                "Come Together - Live",
			NormalizedTitle:      "come together live",
			Artists:              []string{"Tribute Band"},
			NormalizedArtists:    []string{"tribute band"},
			DurationMS:           310000,
			ISRC:                 "OTHER0001",
			TrackNumber:          8,
			AlbumTitle:           "Abbey Road Live",
			AlbumNormalizedTitle: "abbey road live",
			ReleaseDate:          "2020-01-01",
			EditionHints:         []string{"live"},
		},
	}
}

func newSourceServiceSongTargetAdapter() *mocks.MockAdapter {
	return newAdapterMock(model.ServiceSpotify, songTargetCapabilities)
}

func newFailingSongTargetAdapter() adapters.Adapter {
	return newSongTargetFixture(songTargetFixture{
		service: model.ServiceDeezer,
		byMetadata: func(context.Context, model.CanonicalSong) ([]model.CandidateSong, error) {
			return nil, errTargetSearchBoom
		},
	})
}
