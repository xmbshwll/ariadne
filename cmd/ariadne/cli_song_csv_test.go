package main

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne"
)

func TestRunResolveForcedSongCSVOutput(t *testing.T) {
	withResolverFactory(t, func(_ ariadne.Config) *ariadne.Resolver {
		return ariadne.NewWithEntityAdapters(
			nil,
			nil,
			[]ariadne.SongSourceAdapter{newFixtureSongSourceAdapterForCLI(map[string]ariadne.CanonicalSong{
				"https://fixture.test/songs/1": {
					Service:     ariadne.ServiceSpotify,
					SourceID:    "song-1",
					SourceURL:   "https://fixture.test/songs/1",
					Title:       "Fixture Song",
					Artists:     []string{"Fixture Artist"},
					DurationMS:  180000,
					ISRC:        "ISRCSONG001",
					AlbumTitle:  "Fixture Album",
					TrackNumber: 1,
				},
			})},
			[]ariadne.SongTargetAdapter{newFixtureSongTargetAdapterForCLI(ariadne.ServiceAppleMusic, []ariadne.CandidateSong{{
				CanonicalSong: ariadne.CanonicalSong{
					Service:     ariadne.ServiceAppleMusic,
					SourceID:    "apple-song-1",
					SourceURL:   "https://music.apple.com/us/album/fixture-album/2?i=3",
					Title:       "Fixture Song",
					Artists:     []string{"Fixture Artist"},
					DurationMS:  180050,
					ISRC:        "ISRCSONG001",
					AlbumTitle:  "Fixture Album",
					TrackNumber: 1,
				},
				CandidateID: "apple-song-1",
				MatchURL:    "https://music.apple.com/us/album/fixture-album/2?i=3",
			}}, nil)},
		)
	})

	var stdout bytes.Buffer
	err := runResolve([]string{"--song", "--format=csv", "https://fixture.test/songs/1"}, &stdout)
	require.NoError(t, err)

	records, err := csv.NewReader(strings.NewReader(stdout.String())).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 3)
	assert.Equal(t, []string{"service", "url"}, records[0])
	assert.Equal(t, []string{"appleMusic", "https://music.apple.com/us/album/fixture-album/2?i=3"}, records[1])
	assert.Equal(t, []string{"spotify", "https://fixture.test/songs/1"}, records[2])
}

func TestRunResolveForcedSongPropagatesMetadataErrors(t *testing.T) {
	withResolverFactory(t, func(_ ariadne.Config) *ariadne.Resolver {
		return ariadne.NewWithEntityAdapters(
			nil,
			nil,
			[]ariadne.SongSourceAdapter{newFixtureSongSourceAdapterForCLI(map[string]ariadne.CanonicalSong{
				"https://fixture.test/songs/1": {
					Service:     ariadne.ServiceSpotify,
					SourceID:    "song-1",
					SourceURL:   "https://fixture.test/songs/1",
					Title:       "Fixture Song",
					Artists:     []string{"Fixture Artist"},
					DurationMS:  180000,
					AlbumTitle:  "Fixture Album",
					TrackNumber: 1,
				},
			})},
			[]ariadne.SongTargetAdapter{newFixtureSongTargetAdapterForCLI(ariadne.ServiceTIDAL, nil, errCLIResolveBoom)},
		)
	})

	var stdout bytes.Buffer
	err := runResolve([]string{"--song", "https://fixture.test/songs/1"}, &stdout)
	require.Error(t, err)
	assert.ErrorIs(t, err, errCLIResolveBoom)
}

func TestRunResolveForcedSongVerboseCSVOutput(t *testing.T) {
	withResolverFactory(t, func(_ ariadne.Config) *ariadne.Resolver {
		return ariadne.NewWithEntityAdapters(
			nil,
			nil,
			[]ariadne.SongSourceAdapter{newFixtureSongSourceAdapterForCLI(map[string]ariadne.CanonicalSong{
				"https://fixture.test/songs/1": {
					Service:     ariadne.ServiceSpotify,
					SourceID:    "song-1",
					SourceURL:   "https://fixture.test/songs/1",
					RegionHint:  "us",
					Title:       "Fixture Song",
					Artists:     []string{"Fixture Artist"},
					DurationMS:  180000,
					ISRC:        "ISRCSONG001",
					AlbumTitle:  "Fixture Album",
					TrackNumber: 1,
					ReleaseDate: "2024-02-03",
				},
			})},
			[]ariadne.SongTargetAdapter{newFixtureSongTargetAdapterForCLI(ariadne.ServiceAppleMusic, []ariadne.CandidateSong{{
				CanonicalSong: ariadne.CanonicalSong{
					Service:     ariadne.ServiceAppleMusic,
					SourceID:    "apple-song-1",
					SourceURL:   "https://music.apple.com/us/album/fixture-album/2?i=3",
					RegionHint:  "us",
					Title:       "Fixture Song",
					Artists:     []string{"Fixture Artist"},
					DurationMS:  180050,
					ISRC:        "ISRCSONG001",
					AlbumTitle:  "Fixture Album",
					TrackNumber: 1,
					ReleaseDate: "2024-02-03",
				},
				CandidateID: "apple-song-1",
				MatchURL:    "https://music.apple.com/us/album/fixture-album/2?i=3",
			}}, nil)},
		)
	})

	var stdout bytes.Buffer
	err := runResolve([]string{"--song", "--verbose", "--format=csv", "https://fixture.test/songs/1"}, &stdout)
	require.NoError(t, err)

	records, err := csv.NewReader(strings.NewReader(stdout.String())).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 3)
	assert.Equal(t, []string{"input_url", "service", "kind", "url", "found", "summary", "score", "song_id", "region_hint", "title", "artists", "duration_ms", "isrc", "album_title", "track_number", "release_date", "reasons"}, records[0])
	assert.Equal(t, []string{"https://fixture.test/songs/1", "spotify", "source", "https://fixture.test/songs/1", "true", "source", "", "song-1", "us", "Fixture Song", "Fixture Artist", "180000", "ISRCSONG001", "Fixture Album", "1", "2024-02-03", ""}, records[1])
	assert.Equal(t, "https://fixture.test/songs/1", records[2][0])
	assert.Equal(t, "appleMusic", records[2][1])
	assert.Equal(t, "best", records[2][2])
	assert.Equal(t, "https://music.apple.com/us/album/fixture-album/2?i=3", records[2][3])
	assert.Equal(t, "true", records[2][4])
	assert.NotEmpty(t, records[2][5])
	assert.NotEmpty(t, records[2][6])
	assert.Equal(t, "apple-song-1", records[2][7])
	assert.Equal(t, "us", records[2][8])
	assert.Equal(t, "Fixture Song", records[2][9])
	assert.Equal(t, "Fixture Artist", records[2][10])
	assert.Equal(t, "180050", records[2][11])
	assert.Equal(t, "ISRCSONG001", records[2][12])
	assert.Equal(t, "Fixture Album", records[2][13])
	assert.Equal(t, "1", records[2][14])
	assert.Equal(t, "2024-02-03", records[2][15])
	assert.NotEmpty(t, records[2][16])
}
