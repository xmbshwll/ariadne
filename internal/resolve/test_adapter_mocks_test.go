package resolve

import (
	"context"

	"github.com/xmbshwll/ariadne/internal/model"
)

type mockSourceAdapter struct {
	service       model.ServiceName
	parseAlbumURL func(string) (*model.ParsedAlbumURL, error)
	fetchAlbum    func(context.Context, model.ParsedAlbumURL) (*model.CanonicalAlbum, error)
}

func (a mockSourceAdapter) Service() model.ServiceName { return a.service }
func (a mockSourceAdapter) ParseAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	return a.parseAlbumURL(raw)
}
func (a mockSourceAdapter) FetchAlbum(ctx context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalAlbum, error) {
	return a.fetchAlbum(ctx, parsed)
}

type mockTargetAdapter struct {
	service          model.ServiceName
	searchByUPC      func(context.Context, string) ([]model.CandidateAlbum, error)
	searchByISRC     func(context.Context, []string) ([]model.CandidateAlbum, error)
	searchByMetadata func(context.Context, model.CanonicalAlbum) ([]model.CandidateAlbum, error)
}

func (a mockTargetAdapter) Service() model.ServiceName { return a.service }
func (a mockTargetAdapter) SearchByUPC(ctx context.Context, upc string) ([]model.CandidateAlbum, error) {
	return a.searchByUPC(ctx, upc)
}
func (a mockTargetAdapter) SearchByISRC(ctx context.Context, isrcs []string) ([]model.CandidateAlbum, error) {
	return a.searchByISRC(ctx, isrcs)
}
func (a mockTargetAdapter) SearchByMetadata(ctx context.Context, album model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
	return a.searchByMetadata(ctx, album)
}

type mockSongSourceAdapter struct {
	service      model.ServiceName
	parseSongURL func(string) (*model.ParsedURL, error)
	fetchSong    func(context.Context, model.ParsedURL) (*model.CanonicalSong, error)
}

func (a mockSongSourceAdapter) Service() model.ServiceName { return a.service }
func (a mockSongSourceAdapter) ParseSongURL(raw string) (*model.ParsedURL, error) {
	return a.parseSongURL(raw)
}
func (a mockSongSourceAdapter) FetchSong(ctx context.Context, parsed model.ParsedURL) (*model.CanonicalSong, error) {
	return a.fetchSong(ctx, parsed)
}

type mockSongTargetAdapter struct {
	service              model.ServiceName
	searchSongByISRC     func(context.Context, string) ([]model.CandidateSong, error)
	searchSongByMetadata func(context.Context, model.CanonicalSong) ([]model.CandidateSong, error)
}

func (a mockSongTargetAdapter) Service() model.ServiceName { return a.service }
func (a mockSongTargetAdapter) SearchSongByISRC(ctx context.Context, isrc string) ([]model.CandidateSong, error) {
	return a.searchSongByISRC(ctx, isrc)
}
func (a mockSongTargetAdapter) SearchSongByMetadata(ctx context.Context, song model.CanonicalSong) ([]model.CandidateSong, error) {
	return a.searchSongByMetadata(ctx, song)
}

func newStubSourceAdapter() SourceAdapter {
	return mockSourceAdapter{
		service: model.ServiceDeezer,
		parseAlbumURL: func(raw string) (*model.ParsedAlbumURL, error) {
			if raw != "https://www.deezer.com/album/12047952" {
				return nil, errUnsupportedTestSource
			}
			return &model.ParsedAlbumURL{Service: model.ServiceDeezer, EntityType: "album", ID: "12047952", CanonicalURL: raw, RawURL: raw}, nil
		},
		fetchAlbum: func(_ context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalAlbum, error) {
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
		},
	}
}

func newStubTargetAdapter() TargetAdapter {
	return mockTargetAdapter{
		service: model.ServiceSpotify,
		searchByUPC: func(_ context.Context, upc string) ([]model.CandidateAlbum, error) {
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
		},
		searchByISRC: func(_ context.Context, isrcs []string) ([]model.CandidateAlbum, error) {
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
		},
		searchByMetadata: func(_ context.Context, album model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
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
		},
	}
}

func newBlockingTargetAdapter(service model.ServiceName, started chan<- struct{}, release <-chan struct{}) TargetAdapter {
	return mockTargetAdapter{
		service:      service,
		searchByUPC:  func(_ context.Context, _ string) ([]model.CandidateAlbum, error) { return nil, nil },
		searchByISRC: func(_ context.Context, _ []string) ([]model.CandidateAlbum, error) { return nil, nil },
		searchByMetadata: func(_ context.Context, _ model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
			select {
			case started <- struct{}{}:
			default:
			}
			<-release
			return nil, nil
		},
	}
}

func newSourceServiceTargetAdapter(called *bool) TargetAdapter {
	return mockTargetAdapter{
		service:      model.ServiceDeezer,
		searchByUPC:  func(_ context.Context, _ string) ([]model.CandidateAlbum, error) { *called = true; return nil, nil },
		searchByISRC: func(_ context.Context, _ []string) ([]model.CandidateAlbum, error) { *called = true; return nil, nil },
		searchByMetadata: func(_ context.Context, _ model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
			*called = true
			return nil, nil
		},
	}
}

func newFailingTargetAdapter() TargetAdapter {
	return mockTargetAdapter{
		service:      model.ServiceSpotify,
		searchByUPC:  func(_ context.Context, _ string) ([]model.CandidateAlbum, error) { return nil, nil },
		searchByISRC: func(_ context.Context, _ []string) ([]model.CandidateAlbum, error) { return nil, nil },
		searchByMetadata: func(_ context.Context, _ model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
			return nil, errTargetSearchBoom
		},
	}
}

func newFixtureSourceAdapter(albumsByURL map[string]model.CanonicalAlbum) SourceAdapter {
	return mockSourceAdapter{
		service: "fixture",
		parseAlbumURL: func(raw string) (*model.ParsedAlbumURL, error) {
			album, ok := albumsByURL[raw]
			if !ok {
				return nil, errUnsupportedTestSource
			}
			return &model.ParsedAlbumURL{Service: album.Service, EntityType: "album", ID: album.SourceID, CanonicalURL: raw, RawURL: raw}, nil
		},
		fetchAlbum: func(_ context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalAlbum, error) {
			album, ok := albumsByURL[parsed.RawURL]
			if !ok {
				return nil, errTestSourceNotFound
			}
			albumCopy := album
			return &albumCopy, nil
		},
	}
}

func newSingleAlbumSourceAdapter(inputURL string, album model.CanonicalAlbum) SourceAdapter {
	return newFixtureSourceAdapter(map[string]model.CanonicalAlbum{inputURL: album})
}

func newFixtureTargetAdapter(service model.ServiceName, candidatesBySourceID map[string][]model.CandidateAlbum) TargetAdapter {
	return mockTargetAdapter{
		service:      service,
		searchByUPC:  func(_ context.Context, _ string) ([]model.CandidateAlbum, error) { return nil, nil },
		searchByISRC: func(_ context.Context, _ []string) ([]model.CandidateAlbum, error) { return nil, nil },
		searchByMetadata: func(_ context.Context, album model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
			return append([]model.CandidateAlbum(nil), candidatesBySourceID[album.SourceID]...), nil
		},
	}
}

func newStubSongSourceAdapter() SongSourceAdapter {
	return mockSongSourceAdapter{
		service: model.ServiceSpotify,
		parseSongURL: func(raw string) (*model.ParsedURL, error) {
			if raw != "https://open.spotify.com/track/track-1" {
				return nil, errUnsupportedTestSource
			}
			return &model.ParsedURL{Service: model.ServiceSpotify, EntityType: "song", ID: "track-1", CanonicalURL: raw, RawURL: raw}, nil
		},
		fetchSong: func(_ context.Context, parsed model.ParsedURL) (*model.CanonicalSong, error) {
			return &model.CanonicalSong{Service: parsed.Service, SourceID: parsed.ID, SourceURL: parsed.CanonicalURL, Title: "Come Together", NormalizedTitle: "come together", Artists: []string{"The Beatles"}, NormalizedArtists: []string{"the beatles"}, DurationMS: 259000, ISRC: "GBAYE0601690", TrackNumber: 1, AlbumTitle: "Abbey Road (Remastered)", AlbumNormalizedTitle: "abbey road remastered", ReleaseDate: "1969-09-26", EditionHints: []string{"remastered"}}, nil
		},
	}
}

func newNilSongSourceAdapter() SongSourceAdapter {
	return mockSongSourceAdapter{
		service: model.ServiceSpotify,
		parseSongURL: func(raw string) (*model.ParsedURL, error) {
			if raw != "https://open.spotify.com/track/track-1" {
				return nil, errUnsupportedTestSource
			}
			return &model.ParsedURL{Service: model.ServiceSpotify, EntityType: "song", ID: "track-1", CanonicalURL: raw, RawURL: raw}, nil
		},
		fetchSong: func(_ context.Context, _ model.ParsedURL) (*model.CanonicalSong, error) {
			return nil, nil
		},
	}
}

func newStubSongTargetAdapter() SongTargetAdapter {
	return mockSongTargetAdapter{
		service: model.ServiceAppleMusic,
		searchSongByISRC: func(_ context.Context, isrc string) ([]model.CandidateSong, error) {
			if isrc == "" {
				return nil, nil
			}
			return []model.CandidateSong{{CandidateID: "song-1", MatchURL: "https://music.apple.com/us/song/1", CanonicalSong: model.CanonicalSong{Service: model.ServiceAppleMusic, SourceID: "song-1", SourceURL: "https://music.apple.com/us/song/1", Title: "Come Together", NormalizedTitle: "come together", Artists: []string{"The Beatles"}, NormalizedArtists: []string{"the beatles"}, DurationMS: 258947, ISRC: isrc, TrackNumber: 1, AlbumTitle: "Abbey Road (Remastered)", AlbumNormalizedTitle: "abbey road remastered", ReleaseDate: "1969-09-26", EditionHints: []string{"remastered"}}}}, nil
		},
		searchSongByMetadata: func(_ context.Context, song model.CanonicalSong) ([]model.CandidateSong, error) {
			if song.Title == "" {
				return nil, nil
			}
			return []model.CandidateSong{
				{CandidateID: "song-1", MatchURL: "https://music.apple.com/us/song/1", CanonicalSong: model.CanonicalSong{Service: model.ServiceAppleMusic, SourceID: "song-1", SourceURL: "https://music.apple.com/us/song/1", Title: "Come Together", NormalizedTitle: "come together", Artists: []string{"The Beatles"}, NormalizedArtists: []string{"the beatles"}, DurationMS: 258947, ISRC: "GBAYE0601690", TrackNumber: 1, AlbumTitle: "Abbey Road (Remastered)", AlbumNormalizedTitle: "abbey road remastered", ReleaseDate: "1969-09-26", EditionHints: []string{"remastered"}}},
				{CandidateID: "song-2", MatchURL: "https://music.apple.com/us/song/2", CanonicalSong: model.CanonicalSong{Service: model.ServiceAppleMusic, SourceID: "song-2", SourceURL: "https://music.apple.com/us/song/2", Title: "Come Together - Live", NormalizedTitle: "come together live", Artists: []string{"Tribute Band"}, NormalizedArtists: []string{"tribute band"}, DurationMS: 310000, ISRC: "OTHER0001", TrackNumber: 8, AlbumTitle: "Abbey Road Live", AlbumNormalizedTitle: "abbey road live", ReleaseDate: "2020-01-01", EditionHints: []string{"live"}}},
			}, nil
		},
	}
}

func newSourceServiceSongTargetAdapter(called *bool) SongTargetAdapter {
	return mockSongTargetAdapter{
		service:          model.ServiceSpotify,
		searchSongByISRC: func(_ context.Context, _ string) ([]model.CandidateSong, error) { *called = true; return nil, nil },
		searchSongByMetadata: func(_ context.Context, _ model.CanonicalSong) ([]model.CandidateSong, error) {
			*called = true
			return nil, nil
		},
	}
}

func newFailingSongTargetAdapter() SongTargetAdapter {
	return mockSongTargetAdapter{
		service:          model.ServiceAppleMusic,
		searchSongByISRC: func(_ context.Context, _ string) ([]model.CandidateSong, error) { return nil, nil },
		searchSongByMetadata: func(_ context.Context, _ model.CanonicalSong) ([]model.CandidateSong, error) {
			return nil, errTargetSearchBoom
		},
	}
}
