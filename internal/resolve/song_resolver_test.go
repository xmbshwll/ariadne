package resolve

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/score"
)

func TestSongResolverResolveSong(t *testing.T) {
	tests := []struct {
		name               string
		resolver           *SongResolver
		inputURL           string
		wantErr            error
		wantSourceService  model.ServiceName
		wantSourceTitle    string
		wantBestCandidates map[model.ServiceName]string
	}{
		{
			name:     "no source adapters",
			resolver: NewSongs(nil, nil, score.DefaultSongWeights()),
			inputURL: "https://open.spotify.com/track/1",
			wantErr:  ErrNoSourceAdapters,
		},
		{
			name: "unsupported url",
			resolver: NewSongs(
				[]SongSourceAdapter{stubSongSourceAdapter{}},
				nil,
				score.DefaultSongWeights(),
			),
			inputURL: "https://example.com/track/123",
			wantErr:  ErrUnsupportedURL,
		},
		{
			name: "collect song candidates and dedupe",
			resolver: NewSongs(
				[]SongSourceAdapter{stubSongSourceAdapter{}},
				[]SongTargetAdapter{stubSongTargetAdapter{}},
				score.DefaultSongWeights(),
			),
			inputURL:          "https://open.spotify.com/track/track-1",
			wantSourceService: model.ServiceSpotify,
			wantSourceTitle:   "Come Together",
			wantBestCandidates: map[model.ServiceName]string{
				model.ServiceAppleMusic: "song-1",
			},
		},
		{
			name: "nil source song",
			resolver: NewSongs(
				[]SongSourceAdapter{nilSongSourceAdapter{}},
				nil,
				score.DefaultSongWeights(),
			),
			inputURL: "https://open.spotify.com/track/track-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolution, err := tt.resolver.ResolveSong(context.Background(), tt.inputURL)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			if tt.name == "nil source song" {
				require.Error(t, err)
				assert.Nil(t, resolution)
				assert.EqualError(t, err, "fetch source song returned nil from spotify")
				assert.ErrorIs(t, err, errNilSourceSong)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tt.wantSourceService, resolution.Source.Service)
			assert.Equal(t, tt.wantSourceTitle, resolution.Source.Title)

			for service, wantID := range tt.wantBestCandidates {
				match := resolution.Matches[service]
				require.NotNil(t, match.Best)
				assert.Equal(t, wantID, match.Best.Candidate.CandidateID)
			}
		})
	}
}

type stubSongSourceAdapter struct{}

func (stubSongSourceAdapter) Service() model.ServiceName {
	return model.ServiceSpotify
}

func (stubSongSourceAdapter) ParseSongURL(raw string) (*model.ParsedAlbumURL, error) {
	if raw != "https://open.spotify.com/track/track-1" {
		return nil, errUnsupportedTestSource
	}
	return &model.ParsedAlbumURL{Service: model.ServiceSpotify, EntityType: "song", ID: "track-1", CanonicalURL: raw, RawURL: raw}, nil
}

func (stubSongSourceAdapter) FetchSong(_ context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalSong, error) {
	return &model.CanonicalSong{
		Service:              parsed.Service,
		SourceID:             parsed.ID,
		SourceURL:            parsed.CanonicalURL,
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
	}, nil
}

type nilSongSourceAdapter struct{}

func (nilSongSourceAdapter) Service() model.ServiceName {
	return model.ServiceSpotify
}

func (nilSongSourceAdapter) ParseSongURL(raw string) (*model.ParsedAlbumURL, error) {
	if raw != "https://open.spotify.com/track/track-1" {
		return nil, errUnsupportedTestSource
	}
	return &model.ParsedAlbumURL{Service: model.ServiceSpotify, EntityType: "song", ID: "track-1", CanonicalURL: raw, RawURL: raw}, nil
}

func (nilSongSourceAdapter) FetchSong(_ context.Context, _ model.ParsedAlbumURL) (*model.CanonicalSong, error) {
	//nolint:nilnil // Exercise resolver guard for adapters that incorrectly return (nil, nil).
	return nil, nil
}

type stubSongTargetAdapter struct{}

func (stubSongTargetAdapter) Service() model.ServiceName {
	return model.ServiceAppleMusic
}

func (stubSongTargetAdapter) SearchSongByISRC(_ context.Context, isrc string) ([]model.CandidateSong, error) {
	if isrc == "" {
		return nil, nil
	}
	return []model.CandidateSong{
		{CandidateID: "song-1", MatchURL: "https://music.apple.com/us/song/1", CanonicalSong: model.CanonicalSong{Service: model.ServiceAppleMusic, SourceID: "song-1", SourceURL: "https://music.apple.com/us/song/1", Title: "Come Together", NormalizedTitle: "come together", Artists: []string{"The Beatles"}, NormalizedArtists: []string{"the beatles"}, DurationMS: 258947, ISRC: isrc, TrackNumber: 1, AlbumTitle: "Abbey Road (Remastered)", AlbumNormalizedTitle: "abbey road remastered", ReleaseDate: "1969-09-26", EditionHints: []string{"remastered"}}},
	}, nil
}

func (stubSongTargetAdapter) SearchSongByMetadata(_ context.Context, song model.CanonicalSong) ([]model.CandidateSong, error) {
	if song.Title == "" {
		return nil, nil
	}
	return []model.CandidateSong{
		{CandidateID: "song-1", MatchURL: "https://music.apple.com/us/song/1", CanonicalSong: model.CanonicalSong{Service: model.ServiceAppleMusic, SourceID: "song-1", SourceURL: "https://music.apple.com/us/song/1", Title: "Come Together", NormalizedTitle: "come together", Artists: []string{"The Beatles"}, NormalizedArtists: []string{"the beatles"}, DurationMS: 258947, ISRC: "GBAYE0601690", TrackNumber: 1, AlbumTitle: "Abbey Road (Remastered)", AlbumNormalizedTitle: "abbey road remastered", ReleaseDate: "1969-09-26", EditionHints: []string{"remastered"}}},
		{CandidateID: "song-2", MatchURL: "https://music.apple.com/us/song/2", CanonicalSong: model.CanonicalSong{Service: model.ServiceAppleMusic, SourceID: "song-2", SourceURL: "https://music.apple.com/us/song/2", Title: "Come Together - Live", NormalizedTitle: "come together live", Artists: []string{"Tribute Band"}, NormalizedArtists: []string{"tribute band"}, DurationMS: 310000, ISRC: "OTHER0001", TrackNumber: 8, AlbumTitle: "Abbey Road Live", AlbumNormalizedTitle: "abbey road live", ReleaseDate: "2020-01-01", EditionHints: []string{"live"}}},
	}, nil
}
