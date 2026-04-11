package ariadne

import "context"

type mockSourceAdapter struct {
	service       ServiceName
	parseAlbumURL func(string) (*ParsedAlbumURL, error)
	fetchAlbum    func(context.Context, ParsedAlbumURL) (*CanonicalAlbum, error)
}

func (a mockSourceAdapter) Service() ServiceName {
	return a.service
}

func (a mockSourceAdapter) ParseAlbumURL(raw string) (*ParsedAlbumURL, error) {
	return a.parseAlbumURL(raw)
}

func (a mockSourceAdapter) FetchAlbum(ctx context.Context, parsed ParsedAlbumURL) (*CanonicalAlbum, error) {
	return a.fetchAlbum(ctx, parsed)
}

type mockTargetAdapter struct {
	service          ServiceName
	searchByUPC      func(context.Context, string) ([]CandidateAlbum, error)
	searchByISRC     func(context.Context, []string) ([]CandidateAlbum, error)
	searchByMetadata func(context.Context, CanonicalAlbum) ([]CandidateAlbum, error)
}

func (a mockTargetAdapter) Service() ServiceName {
	return a.service
}

func (a mockTargetAdapter) SearchByUPC(ctx context.Context, upc string) ([]CandidateAlbum, error) {
	return a.searchByUPC(ctx, upc)
}

func (a mockTargetAdapter) SearchByISRC(ctx context.Context, isrcs []string) ([]CandidateAlbum, error) {
	return a.searchByISRC(ctx, isrcs)
}

func (a mockTargetAdapter) SearchByMetadata(ctx context.Context, album CanonicalAlbum) ([]CandidateAlbum, error) {
	return a.searchByMetadata(ctx, album)
}

type mockSongSourceAdapter struct {
	service      ServiceName
	parseSongURL func(string) (*ParsedURL, error)
	fetchSong    func(context.Context, ParsedURL) (*CanonicalSong, error)
}

func (a mockSongSourceAdapter) Service() ServiceName {
	return a.service
}

func (a mockSongSourceAdapter) ParseSongURL(raw string) (*ParsedURL, error) {
	return a.parseSongURL(raw)
}

func (a mockSongSourceAdapter) FetchSong(ctx context.Context, parsed ParsedURL) (*CanonicalSong, error) {
	return a.fetchSong(ctx, parsed)
}

type mockSongTargetAdapter struct {
	service              ServiceName
	searchSongByISRC     func(context.Context, string) ([]CandidateSong, error)
	searchSongByMetadata func(context.Context, CanonicalSong) ([]CandidateSong, error)
}

func (a mockSongTargetAdapter) Service() ServiceName {
	return a.service
}

func (a mockSongTargetAdapter) SearchSongByISRC(ctx context.Context, isrc string) ([]CandidateSong, error) {
	return a.searchSongByISRC(ctx, isrc)
}

func (a mockSongTargetAdapter) SearchSongByMetadata(ctx context.Context, song CanonicalSong) ([]CandidateSong, error) {
	return a.searchSongByMetadata(ctx, song)
}

func newLibrarySourceAdapter() SourceAdapter {
	return mockSourceAdapter{
		service: ServiceDeezer,
		parseAlbumURL: func(raw string) (*ParsedAlbumURL, error) {
			if raw != testLibrarySourceURL {
				return nil, errUnsupportedLibrarySource
			}
			return &ParsedAlbumURL{Service: ServiceDeezer, EntityType: "album", ID: "src-1", CanonicalURL: raw, RawURL: raw}, nil
		},
		fetchAlbum: func(_ context.Context, parsed ParsedAlbumURL) (*CanonicalAlbum, error) {
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
		},
	}
}

func newNilParsedSourceAdapter() SourceAdapter {
	return mockSourceAdapter{
		service: ServiceDeezer,
		parseAlbumURL: func(raw string) (*ParsedAlbumURL, error) {
			if raw != testLibrarySourceURL {
				return nil, errUnsupportedLibrarySource
			}
			return nil, nil
		},
		fetchAlbum: func(_ context.Context, _ ParsedAlbumURL) (*CanonicalAlbum, error) {
			return nil, nil
		},
	}
}

func newNilAlbumSourceAdapter() SourceAdapter {
	return mockSourceAdapter{
		service: ServiceDeezer,
		parseAlbumURL: func(raw string) (*ParsedAlbumURL, error) {
			if raw != testLibrarySourceURL {
				return nil, errUnsupportedLibrarySource
			}
			return &ParsedAlbumURL{Service: ServiceDeezer, EntityType: "album", ID: "src-1", CanonicalURL: raw, RawURL: raw}, nil
		},
		fetchAlbum: func(_ context.Context, _ ParsedAlbumURL) (*CanonicalAlbum, error) {
			return nil, nil
		},
	}
}

func newLibraryTargetAdapter() TargetAdapter {
	return mockTargetAdapter{
		service: ServiceSpotify,
		searchByUPC: func(_ context.Context, upc string) ([]CandidateAlbum, error) {
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
		},
		searchByISRC:     func(_ context.Context, _ []string) ([]CandidateAlbum, error) { return nil, nil },
		searchByMetadata: func(_ context.Context, _ CanonicalAlbum) ([]CandidateAlbum, error) { return nil, nil },
	}
}

func newFailingLibraryTargetAdapter() TargetAdapter {
	return mockTargetAdapter{
		service:      ServiceSpotify,
		searchByUPC:  func(_ context.Context, _ string) ([]CandidateAlbum, error) { return nil, nil },
		searchByISRC: func(_ context.Context, _ []string) ([]CandidateAlbum, error) { return nil, nil },
		searchByMetadata: func(_ context.Context, _ CanonicalAlbum) ([]CandidateAlbum, error) {
			return nil, errLibraryTargetBoom
		},
	}
}

func newLibrarySongSourceAdapter() SongSourceAdapter {
	return mockSongSourceAdapter{
		service: ServiceSpotify,
		parseSongURL: func(raw string) (*ParsedURL, error) {
			if raw != "https://fixture.test/songs/1" {
				return nil, errUnsupportedLibrarySource
			}
			return &ParsedURL{Service: ServiceSpotify, EntityType: "song", ID: "song-1", CanonicalURL: raw, RawURL: raw}, nil
		},
		fetchSong: func(_ context.Context, parsed ParsedURL) (*CanonicalSong, error) {
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
		},
	}
}

func newNilSongSourceAdapter() SongSourceAdapter {
	return mockSongSourceAdapter{
		service: ServiceSpotify,
		parseSongURL: func(raw string) (*ParsedURL, error) {
			if raw != "https://fixture.test/songs/1" {
				return nil, errUnsupportedLibrarySource
			}
			return &ParsedURL{Service: ServiceSpotify, EntityType: "song", ID: "song-1", CanonicalURL: raw, RawURL: raw}, nil
		},
		fetchSong: func(_ context.Context, _ ParsedURL) (*CanonicalSong, error) {
			return nil, nil
		},
	}
}

func newLibrarySongTargetAdapter() SongTargetAdapter {
	return mockSongTargetAdapter{
		service: ServiceAppleMusic,
		searchSongByISRC: func(_ context.Context, isrc string) ([]CandidateSong, error) {
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
		},
		searchSongByMetadata: func(_ context.Context, _ CanonicalSong) ([]CandidateSong, error) { return nil, nil },
	}
}
