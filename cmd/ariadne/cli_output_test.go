package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne"
	"gopkg.in/yaml.v3"
)

func TestNewCLIResolution(t *testing.T) {
	resolution := ariadne.Resolution{
		InputURL: "https://open.spotify.com/album/abc",
		Source: ariadne.CanonicalAlbum{
			Service:      ariadne.ServiceSpotify,
			SourceID:     "abc",
			SourceURL:    "https://open.spotify.com/album/abc",
			RegionHint:   "us",
			Title:        "Album",
			Artists:      []string{"Artist"},
			ReleaseDate:  "2024-01-01",
			TrackCount:   10,
			ArtworkURL:   "https://image.test/art.jpg",
			EditionHints: []string{"remastered"},
		},
		Matches: map[ariadne.ServiceName]ariadne.MatchResult{
			ariadne.ServiceDeezer: {
				Service: ariadne.ServiceDeezer,
				Best: &ariadne.ScoredMatch{
					URL:     "https://www.deezer.com/album/1",
					Score:   140,
					Reasons: []string{"upc exact match", "title exact match"},
					Candidate: ariadne.CandidateAlbum{
						CandidateID: "1",
						CanonicalAlbum: ariadne.CanonicalAlbum{
							Title:       "Album",
							Artists:     []string{"Artist"},
							ReleaseDate: "2024-01-01",
							UPC:         "123",
						},
					},
				},
			},
			ariadne.ServiceAppleMusic: {
				Service: ariadne.ServiceAppleMusic,
				Best: &ariadne.ScoredMatch{
					URL:     "https://music.apple.com/us/album/album/2",
					Score:   95,
					Reasons: []string{"title exact match", "primary artist exact match"},
					Candidate: ariadne.CandidateAlbum{
						CandidateID: "2",
						CanonicalAlbum: ariadne.CanonicalAlbum{
							RegionHint:  "us",
							Title:       "Album",
							Artists:     []string{"Artist"},
							ReleaseDate: "2024-01-01",
						},
					},
				},
			},
			ariadne.ServiceTIDAL: {
				Service: ariadne.ServiceTIDAL,
				Best: &ariadne.ScoredMatch{
					URL:     "https://tidal.com/album/3",
					Score:   88,
					Reasons: []string{"upc exact match", "track isrc overlap"},
					Candidate: ariadne.CandidateAlbum{
						CandidateID: "3",
						CanonicalAlbum: ariadne.CanonicalAlbum{
							Title:       "Album",
							Artists:     []string{"Artist"},
							ReleaseDate: "2024-01-01",
							UPC:         "123",
						},
					},
				},
				Alternates: []ariadne.ScoredMatch{{
					URL:     "https://tidal.com/album/4",
					Score:   59,
					Reasons: []string{"title exact match"},
					Candidate: ariadne.CandidateAlbum{
						CandidateID: "4",
						CanonicalAlbum: ariadne.CanonicalAlbum{
							Title:   "Album (Deluxe)",
							Artists: []string{"Artist"},
						},
					},
				}},
			},
		},
	}

	output := newCLIResolution(resolution)
	assert.Equal(t, resolution.InputURL, output.InputURL)
	assert.Equal(t, "spotify", output.Source.Service)
	assert.Equal(t, "abc", output.Source.ID)
	assert.Equal(t, "us", output.Source.RegionHint)

	deezer, ok := output.Links["deezer"]
	require.True(t, ok)
	assert.True(t, deezer.Found)
	assert.Equal(t, "strong", deezer.Summary)
	require.NotNil(t, deezer.Best)
	assert.Equal(t, "https://www.deezer.com/album/1", deezer.Best.URL)
	assert.Equal(t, "1", deezer.Best.AlbumID)
	assert.Empty(t, deezer.Best.RegionHint)
	assert.Len(t, deezer.Best.Reasons, 2)

	appleMusic, ok := output.Links["appleMusic"]
	require.True(t, ok)
	require.NotNil(t, appleMusic.Best)
	assert.Equal(t, "us", appleMusic.Best.RegionHint)

	tidal, ok := output.Links["tidal"]
	require.True(t, ok)
	assert.True(t, tidal.Found)
	assert.Equal(t, "probable", tidal.Summary)
	require.NotNil(t, tidal.Best)
	assert.Equal(t, "https://tidal.com/album/3", tidal.Best.URL)
	assert.Equal(t, "3", tidal.Best.AlbumID)
	require.Len(t, tidal.Alternates, 1)
	assert.Equal(t, "4", tidal.Alternates[0].AlbumID)
}

func TestCLIOutputYAMLUsesSnakeCaseTags(t *testing.T) {
	albumOutput := cliResolution{
		InputURL: "https://open.spotify.com/album/abc",
		Source: cliAlbum{
			Service:     "spotify",
			ID:          "abc",
			URL:         "https://open.spotify.com/album/abc",
			RegionHint:  "us",
			Title:       "Album",
			Artists:     []string{"Artist"},
			ReleaseDate: "2024-01-01",
		},
		Links: map[string]cliMatchResult{
			"deezer": {
				Found:   true,
				Summary: "strong",
				Best: &cliMatch{
					URL:         "https://www.deezer.com/album/1",
					Score:       120,
					AlbumID:     "1",
					RegionHint:  "us",
					ReleaseDate: "2024-01-01",
				},
			},
		},
	}

	albumYAML, err := yaml.Marshal(albumOutput)
	require.NoError(t, err)
	albumYAMLText := string(albumYAML)
	assert.Contains(t, albumYAMLText, "input_url:")
	assert.Contains(t, albumYAMLText, "region_hint:")
	assert.Contains(t, albumYAMLText, "album_id:")
	assert.Contains(t, albumYAMLText, "release_date:")
	assert.NotContains(t, albumYAMLText, "inputurl:")
	assert.NotContains(t, albumYAMLText, "albumid:")

	songOutput := cliSongResolution{
		InputURL: "https://open.spotify.com/track/song-1",
		Source: cliSong{
			Service:     "spotify",
			ID:          "song-1",
			URL:         "https://open.spotify.com/track/song-1",
			Title:       "Song",
			Artists:     []string{"Artist"},
			DurationMS:  180000,
			AlbumTitle:  "Album",
			TrackNumber: 1,
		},
		Links: map[string]cliSongMatchResult{
			"appleMusic": {
				Found:   true,
				Summary: "strong",
				Best: &cliSongMatch{
					URL:         "https://music.apple.com/us/album/album/2?i=3",
					Score:       115,
					SongID:      "apple-song-1",
					DurationMS:  180050,
					AlbumTitle:  "Album",
					TrackNumber: 1,
				},
			},
		},
	}

	songYAML, err := yaml.Marshal(songOutput)
	require.NoError(t, err)
	songYAMLText := string(songYAML)
	assert.Contains(t, songYAMLText, "input_url:")
	assert.Contains(t, songYAMLText, "song_id:")
	assert.Contains(t, songYAMLText, "duration_ms:")
	assert.Contains(t, songYAMLText, "album_title:")
	assert.Contains(t, songYAMLText, "track_number:")
	assert.NotContains(t, songYAMLText, "songid:")
	assert.NotContains(t, songYAMLText, "durationms:")
}

func TestNewCLISongResolution(t *testing.T) {
	resolution := ariadne.SongResolution{
		InputURL: "https://open.spotify.com/track/song-1",
		Source: ariadne.CanonicalSong{
			Service:     ariadne.ServiceSpotify,
			SourceID:    "song-1",
			SourceURL:   "https://open.spotify.com/track/song-1",
			RegionHint:  "us",
			Title:       "Song",
			Artists:     []string{"Artist"},
			DurationMS:  180000,
			ISRC:        "ISRC001",
			TrackNumber: 1,
			AlbumID:     "album-1",
			AlbumTitle:  "Album",
		},
		Matches: map[ariadne.ServiceName]ariadne.SongMatchResult{
			ariadne.ServiceAppleMusic: {
				Service: ariadne.ServiceAppleMusic,
				Best: &ariadne.SongScoredMatch{
					URL:     "https://music.apple.com/us/album/album/2?i=3",
					Score:   115,
					Reasons: []string{"isrc exact match", "title exact match"},
					Candidate: ariadne.CandidateSong{
						CandidateID: "apple-song-1",
						CanonicalSong: ariadne.CanonicalSong{
							RegionHint:  "us",
							Title:       "Song",
							Artists:     []string{"Artist"},
							DurationMS:  180050,
							ISRC:        "ISRC001",
							AlbumTitle:  "Album",
							TrackNumber: 1,
						},
					},
				},
			},
		},
	}

	output := newCLISongResolution(resolution)
	assert.Equal(t, resolution.InputURL, output.InputURL)
	assert.Equal(t, "spotify", output.Source.Service)
	assert.Equal(t, "ISRC001", output.Source.ISRC)
	assert.Equal(t, "Album", output.Source.AlbumTitle)
	appleMusic, ok := output.Links["appleMusic"]
	require.True(t, ok)
	require.NotNil(t, appleMusic.Best)
	assert.Equal(t, "apple-song-1", appleMusic.Best.SongID)
	assert.Equal(t, 180050, appleMusic.Best.DurationMS)
}

func TestNewCLILinks(t *testing.T) {
	resolution := ariadne.Resolution{
		Source: ariadne.CanonicalAlbum{
			Service:   ariadne.ServiceDeezer,
			SourceURL: "https://www.deezer.com/album/source",
		},
		Matches: map[ariadne.ServiceName]ariadne.MatchResult{
			ariadne.ServiceSpotify: {
				Best: &ariadne.ScoredMatch{URL: "https://open.spotify.com/album/spotify-1"},
			},
			ariadne.ServiceAppleMusic: {
				Best: &ariadne.ScoredMatch{URL: "https://music.apple.com/us/album/album/2"},
			},
			ariadne.ServiceYouTubeMusic: {},
		},
	}

	output := newCLILinks(resolution)
	assert.Len(t, output, 3)
	assert.Equal(t, "https://www.deezer.com/album/source", output["deezer"])
	assert.Equal(t, "https://open.spotify.com/album/spotify-1", output["spotify"])
	assert.Equal(t, "https://music.apple.com/us/album/album/2", output["appleMusic"])
	_, ok := output["youtubeMusic"]
	assert.False(t, ok)
}

func TestFilterResolutionByStrength(t *testing.T) {
	resolution := ariadne.Resolution{
		Source: ariadne.CanonicalAlbum{Service: ariadne.ServiceDeezer, SourceURL: "https://www.deezer.com/album/source"},
		Matches: map[ariadne.ServiceName]ariadne.MatchResult{
			ariadne.ServiceSpotify: {
				Best: &ariadne.ScoredMatch{URL: "https://open.spotify.com/album/strong", Score: 120},
				Alternates: []ariadne.ScoredMatch{
					{URL: "https://open.spotify.com/album/weak", Score: 45},
					{URL: "https://open.spotify.com/album/probable", Score: 80},
				},
			},
			ariadne.ServiceAppleMusic: {Best: &ariadne.ScoredMatch{URL: "https://music.apple.com/us/album/weak", Score: 55}},
		},
	}

	filtered := filterResolutionByStrength(resolution, ariadne.MatchStrengthProbable)
	assert.Len(t, filtered.Matches, 1)
	spotify, ok := filtered.Matches[ariadne.ServiceSpotify]
	assert.True(t, ok)
	assert.Len(t, spotify.Alternates, 1)
	assert.Equal(t, "https://open.spotify.com/album/probable", spotify.Alternates[0].URL)
	_, ok = filtered.Matches[ariadne.ServiceAppleMusic]
	assert.False(t, ok)
	assert.Len(t, resolution.Matches[ariadne.ServiceSpotify].Alternates, 2)
}

func TestVerboseCSVRowsIncludeAlternatesWithoutBest(t *testing.T) {
	albumRows := newVerboseCSVRows(ariadne.Resolution{
		InputURL: "https://fixture.test/album",
		Source:   ariadne.CanonicalAlbum{Service: ariadne.ServiceDeezer, SourceID: "src", SourceURL: "https://fixture.test/album"},
		Matches: map[ariadne.ServiceName]ariadne.MatchResult{
			ariadne.ServiceSpotify: {
				Alternates: []ariadne.ScoredMatch{{
					URL:       "https://open.spotify.com/album/alt",
					Score:     80,
					Candidate: ariadne.CandidateAlbum{CandidateID: "alt", CanonicalAlbum: ariadne.CanonicalAlbum{Title: "Alt Album"}},
				}},
			},
		},
	})
	assert.Len(t, albumRows, 3)
	assert.Equal(t, "alternate", albumRows[2][2])
	assert.Equal(t, "https://open.spotify.com/album/alt", albumRows[2][3])

	songRows := newVerboseSongCSVRows(ariadne.SongResolution{
		InputURL: "https://fixture.test/song",
		Source:   ariadne.CanonicalSong{Service: ariadne.ServiceSpotify, SourceID: "src", SourceURL: "https://fixture.test/song"},
		Matches: map[ariadne.ServiceName]ariadne.SongMatchResult{
			ariadne.ServiceAppleMusic: {
				Alternates: []ariadne.SongScoredMatch{{
					URL:       "https://music.apple.com/us/album/alt?i=1",
					Score:     82,
					Candidate: ariadne.CandidateSong{CandidateID: "alt-song", CanonicalSong: ariadne.CanonicalSong{Title: "Alt Song"}},
				}},
			},
		},
	})
	assert.Len(t, songRows, 3)
	assert.Equal(t, "alternate", songRows[2][2])
	assert.Equal(t, "https://music.apple.com/us/album/alt?i=1", songRows[2][3])
}

func TestFilterSongResolutionByStrengthPrunesAlternates(t *testing.T) {
	resolution := ariadne.SongResolution{
		Source: ariadne.CanonicalSong{Service: ariadne.ServiceSpotify, SourceURL: "https://open.spotify.com/track/source"},
		Matches: map[ariadne.ServiceName]ariadne.SongMatchResult{
			ariadne.ServiceAppleMusic: {
				Best: &ariadne.SongScoredMatch{URL: "https://music.apple.com/us/album/best?i=1", Score: 115},
				Alternates: []ariadne.SongScoredMatch{
					{URL: "https://music.apple.com/us/album/weak?i=2", Score: 45},
					{URL: "https://music.apple.com/us/album/strong?i=3", Score: 90},
				},
			},
			ariadne.ServiceDeezer: {
				Best: &ariadne.SongScoredMatch{URL: "https://www.deezer.com/track/too-weak", Score: 40},
				Alternates: []ariadne.SongScoredMatch{
					{URL: "https://www.deezer.com/track/alternate", Score: 82},
				},
			},
		},
	}

	filtered := filterSongResolutionByStrength(resolution, ariadne.MatchStrengthProbable)

	appleMusic, ok := filtered.Matches[ariadne.ServiceAppleMusic]
	require.True(t, ok)
	require.NotNil(t, appleMusic.Best)
	require.Len(t, appleMusic.Alternates, 1)
	assert.Equal(t, "https://music.apple.com/us/album/strong?i=3", appleMusic.Alternates[0].URL)

	deezer, ok := filtered.Matches[ariadne.ServiceDeezer]
	require.True(t, ok)
	require.NotNil(t, deezer.Best)
	assert.Equal(t, "https://www.deezer.com/track/alternate", deezer.Best.URL)
	assert.Empty(t, deezer.Alternates)

	require.NotNil(t, resolution.Matches[ariadne.ServiceAppleMusic].Best)
	assert.Len(t, resolution.Matches[ariadne.ServiceAppleMusic].Alternates, 2)
	require.NotNil(t, resolution.Matches[ariadne.ServiceDeezer].Best)
}

func TestScoreSummary(t *testing.T) {
	tests := []struct {
		name  string
		score int
		want  string
	}{
		{name: "strong", score: 100, want: "strong"},
		{name: "probable", score: 70, want: "probable"},
		{name: "weak", score: 50, want: "weak"},
		{name: "very weak", score: 49, want: "very_weak"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, scoreSummary(tt.score))
		})
	}
}
