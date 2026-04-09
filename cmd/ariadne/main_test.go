package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne"
)

var (
	errCLIResolveBoom        = errors.New("boom")
	errUnsupportedCLIFixture = errors.New("unsupported")
	errCLIFixtureNotFound    = errors.New("not found")
)

var errRootBoom = errors.New("boom")

func TestRootError(t *testing.T) {
	err := fmt.Errorf("outer: %w", fmt.Errorf("middle: %w", errRootBoom))
	assert.ErrorIs(t, rootError(err), errRootBoom)
}

func TestRun(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantErr     string
		wantStdout  []string
		wantStderr  []string
		avoidStdout []string
	}{
		{
			name: "help",
			args: []string{"help"},
			wantStdout: []string{
				"Usage:",
				"ariadne resolve [--song|--album] [--verbose] [--format=json|yaml|csv] [--services=spotify,deezer] [--min-strength=probable] [--apple-music-storefront=us] [--resolution-timeout=20s] <url>",
				"<url>",
				"Values: a supported album URL from Apple Music, Deezer, Spotify, TIDAL",
				"URL from Apple Music, Bandcamp, Deezer, SoundCloud, Spotify, or TIDAL.",
				"Behavior: when neither --song nor --album is set, Ariadne asks the library",
				"--song",
				"--album",
				"Commands:",
				"resolve  Resolve a supported album or song URL across services.",
				"--config",
				"Behavior: config file values are loaded first, environment variables override them, and explicit CLI flags override both.",
				"--verbose, -v",
				"--format",
				"--services",
				"--min-strength",
				"--apple-music-storefront",
				"--http-timeout",
				"--resolution-timeout",
				"Spotify target search is enabled only when SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET are set",
				"TIDAL source fetch and target search require TIDAL_CLIENT_ID and TIDAL_CLIENT_SECRET",
				"Amazon Music URLs are recognized for parsing, but runtime resolution remains deferred.",
			},
			avoidStdout: []string{"Global Flags:", "help for resolve", "configuration source (values:"},
		},
		{
			name:       "missing command",
			args:       nil,
			wantErr:    "missing command",
			wantStderr: []string{"Usage:"},
		},
		{
			name:       "unknown command",
			args:       []string{"unknown"},
			wantErr:    "unknown command: unknown",
			wantStderr: []string{"Usage:"},
		},
		{
			name:        "resolve usage",
			args:        []string{"resolve"},
			wantErr:     "usage: ariadne resolve [--song|--album] [--verbose] [--format=json|yaml|csv] [--services=spotify,deezer] [--min-strength=probable] [--apple-music-storefront=us] [--resolution-timeout=20s] <url>",
			avoidStdout: []string{"{"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			err := run(tt.args, &stdout, &stderr)
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}

			for _, want := range tt.wantStdout {
				assert.Contains(t, stdout.String(), want)
			}
			for _, want := range tt.wantStderr {
				assert.Contains(t, stderr.String(), want)
			}
			for _, avoid := range tt.avoidStdout {
				assert.NotContains(t, stdout.String(), avoid)
			}
		})
	}
}

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

func TestResolverRequiresCredentialsForTIDALSourceFetch(t *testing.T) {
	resolver := ariadne.New(ariadne.DefaultConfig())

	_, err := resolver.ResolveAlbum(context.Background(), "https://tidal.com/album/156205493")
	require.Error(t, err)
	assert.ErrorIs(t, err, ariadne.ErrTIDALCredentialsNotConfigured)
}

func TestResolverReportsAmazonMusicAsDeferred(t *testing.T) {
	resolver := ariadne.New(ariadne.DefaultConfig())

	_, err := resolver.ResolveAlbum(context.Background(), "https://music.amazon.com/albums/B0064UPU4G")
	require.Error(t, err)
	assert.ErrorIs(t, err, ariadne.ErrAmazonMusicDeferred)
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
	assert.Nil(t, deezer.Best)
	assert.Len(t, deezer.Alternates, 1)

	require.NotNil(t, resolution.Matches[ariadne.ServiceAppleMusic].Best)
	assert.Len(t, resolution.Matches[ariadne.ServiceAppleMusic].Alternates, 2)
	require.NotNil(t, resolution.Matches[ariadne.ServiceDeezer].Best)
}

func TestRunResolveFixtureOutput(t *testing.T) {
	originalFactory := resolverFactory
	resolverFactory = func(_ ariadne.Config) *ariadne.Resolver {
		return ariadne.NewWithAdapters(
			[]ariadne.SourceAdapter{fixtureSourceAdapterForCLI{albumByURL: map[string]ariadne.CanonicalAlbum{
				"https://fixture.test/source": {
					Service:           ariadne.ServiceDeezer,
					SourceID:          "src-1",
					SourceURL:         "https://fixture.test/source",
					Title:             "Fixture Album",
					NormalizedTitle:   "fixture album",
					Artists:           []string{"Fixture Artist"},
					NormalizedArtists: []string{"fixture artist"},
					ReleaseDate:       "2024-02-03",
					UPC:               "123456789012",
					TrackCount:        2,
					Tracks:            []ariadne.CanonicalTrack{{Title: "Alpha", NormalizedTitle: "alpha", ISRC: "ISRC001"}, {Title: "Beta", NormalizedTitle: "beta"}},
				},
			}}},
			[]ariadne.TargetAdapter{
				fixtureTargetAdapterForCLI{service: ariadne.ServiceSpotify, upcResults: []ariadne.CandidateAlbum{{
					CanonicalAlbum: ariadne.CanonicalAlbum{
						Service:           ariadne.ServiceSpotify,
						SourceID:          "spotify-1",
						SourceURL:         "https://open.spotify.com/album/spotify-1",
						Title:             "Fixture Album",
						NormalizedTitle:   "fixture album",
						Artists:           []string{"Fixture Artist"},
						NormalizedArtists: []string{"fixture artist"},
						ReleaseDate:       "2024-02-03",
						UPC:               "123456789012",
						TrackCount:        2,
						Tracks:            []ariadne.CanonicalTrack{{Title: "Alpha", NormalizedTitle: "alpha", ISRC: "ISRC001"}, {Title: "Beta", NormalizedTitle: "beta"}},
					},
					CandidateID: "spotify-1",
					MatchURL:    "https://open.spotify.com/album/spotify-1",
				}}},
				fixtureTargetAdapterForCLI{service: ariadne.ServiceYouTubeMusic},
			},
		)
	}
	defer func() { resolverFactory = originalFactory }()

	var stdout bytes.Buffer
	err := runResolve([]string{"https://fixture.test/source"}, &stdout)
	require.NoError(t, err)

	var output map[string]string
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &output))
	assert.Equal(t, "https://fixture.test/source", output["deezer"])
	assert.Equal(t, "https://open.spotify.com/album/spotify-1", output["spotify"])
	_, ok := output["youtubeMusic"]
	assert.False(t, ok)
}

func TestRunResolveAutoDispatchesSongFixtureOutput(t *testing.T) {
	originalFactory := resolverFactory
	resolverFactory = func(_ ariadne.Config) *ariadne.Resolver {
		return ariadne.NewWithEntityAdapters(
			[]ariadne.SourceAdapter{fixtureSourceAdapterForCLI{albumByURL: map[string]ariadne.CanonicalAlbum{
				"https://fixture.test/source": {
					Service:   ariadne.ServiceDeezer,
					SourceID:  "src-1",
					SourceURL: "https://fixture.test/source",
					Title:     "Fixture Album",
				},
			}}},
			[]ariadne.TargetAdapter{fixtureTargetAdapterForCLI{service: ariadne.ServiceSpotify}},
			[]ariadne.SongSourceAdapter{fixtureSongSourceAdapterForCLI{songByURL: map[string]ariadne.CanonicalSong{
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
			}}},
			[]ariadne.SongTargetAdapter{fixtureSongTargetAdapterForCLI{service: ariadne.ServiceAppleMusic, isrcResults: []ariadne.CandidateSong{{
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
			}}}},
		)
	}
	defer func() { resolverFactory = originalFactory }()

	var stdout bytes.Buffer
	err := runResolve([]string{"https://fixture.test/songs/1"}, &stdout)
	require.NoError(t, err)

	var output map[string]string
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &output))
	assert.Equal(t, "https://fixture.test/songs/1", output["spotify"])
	assert.Equal(t, "https://music.apple.com/us/album/fixture-album/2?i=3", output["appleMusic"])
}

func TestRunResolveForcedSongFixtureOutput(t *testing.T) {
	originalFactory := resolverFactory
	resolverFactory = func(_ ariadne.Config) *ariadne.Resolver {
		return ariadne.NewWithEntityAdapters(
			nil,
			nil,
			[]ariadne.SongSourceAdapter{fixtureSongSourceAdapterForCLI{songByURL: map[string]ariadne.CanonicalSong{
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
				},
			}}},
			[]ariadne.SongTargetAdapter{fixtureSongTargetAdapterForCLI{service: ariadne.ServiceAppleMusic, isrcResults: []ariadne.CandidateSong{{
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
				},
				CandidateID: "apple-song-1",
				MatchURL:    "https://music.apple.com/us/album/fixture-album/2?i=3",
			}}}},
		)
	}
	defer func() { resolverFactory = originalFactory }()

	var stdout bytes.Buffer
	err := run([]string{"resolve", "--song", "--verbose", "https://fixture.test/songs/1"}, &stdout, io.Discard)
	require.NoError(t, err)

	var output cliSongResolution
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &output))
	assert.Equal(t, "Fixture Song", output.Source.Title)
	assert.Equal(t, "ISRCSONG001", output.Source.ISRC)
	require.NotNil(t, output.Links["appleMusic"].Best)
	assert.Equal(t, "apple-song-1", output.Links["appleMusic"].Best.SongID)
}

func TestRunResolveServiceFilter(t *testing.T) {
	originalFactory := resolverFactory
	resolverFactory = func(cfg ariadne.Config) *ariadne.Resolver {
		targets := []ariadne.TargetAdapter{}
		for _, service := range cfg.TargetServices {
			if service != ariadne.ServiceDeezer {
				continue
			}
			targets = append(targets, fixtureTargetAdapterForCLI{service: ariadne.ServiceDeezer, upcResults: []ariadne.CandidateAlbum{{
				CanonicalAlbum: ariadne.CanonicalAlbum{
					Service:           ariadne.ServiceDeezer,
					SourceID:          "deezer-1",
					SourceURL:         "https://www.deezer.com/album/deezer-1",
					Title:             "Fixture Album",
					NormalizedTitle:   "fixture album",
					Artists:           []string{"Fixture Artist"},
					NormalizedArtists: []string{"fixture artist"},
					ReleaseDate:       "2024-02-03",
					UPC:               "123456789012",
				},
				CandidateID: "deezer-1",
				MatchURL:    "https://www.deezer.com/album/deezer-1",
			}}})
		}
		return ariadne.NewWithAdapters(
			[]ariadne.SourceAdapter{fixtureSourceAdapterForCLI{albumByURL: map[string]ariadne.CanonicalAlbum{
				"https://fixture.test/source": {
					Service:           ariadne.ServiceAppleMusic,
					SourceID:          "src-1",
					SourceURL:         "https://fixture.test/source",
					Title:             "Fixture Album",
					NormalizedTitle:   "fixture album",
					Artists:           []string{"Fixture Artist"},
					NormalizedArtists: []string{"fixture artist"},
					ReleaseDate:       "2024-02-03",
					UPC:               "123456789012",
				},
			}}},
			targets,
		)
	}
	defer func() { resolverFactory = originalFactory }()

	var stdout bytes.Buffer
	err := runResolve([]string{"--services=deezer", "https://fixture.test/source"}, &stdout)
	require.NoError(t, err)

	var output map[string]string
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &output))
	assert.Equal(t, "https://fixture.test/source", output["appleMusic"])
	assert.Equal(t, "https://www.deezer.com/album/deezer-1", output["deezer"])
	_, ok := output["spotify"]
	assert.False(t, ok)
}

func TestRunResolveYAMLFixtureOutput(t *testing.T) {
	originalFactory := resolverFactory
	resolverFactory = func(_ ariadne.Config) *ariadne.Resolver {
		return ariadne.NewWithAdapters(
			[]ariadne.SourceAdapter{fixtureSourceAdapterForCLI{albumByURL: map[string]ariadne.CanonicalAlbum{
				"https://fixture.test/source": {
					Service:           ariadne.ServiceDeezer,
					SourceID:          "src-1",
					SourceURL:         "https://fixture.test/source",
					Title:             "Fixture Album",
					NormalizedTitle:   "fixture album",
					Artists:           []string{"Fixture Artist"},
					NormalizedArtists: []string{"fixture artist"},
					ReleaseDate:       "2024-02-03",
					UPC:               "123456789012",
				},
			}}},
			[]ariadne.TargetAdapter{fixtureTargetAdapterForCLI{service: ariadne.ServiceSpotify, upcResults: []ariadne.CandidateAlbum{{
				CanonicalAlbum: ariadne.CanonicalAlbum{
					Service:           ariadne.ServiceSpotify,
					SourceID:          "spotify-1",
					SourceURL:         "https://open.spotify.com/album/spotify-1",
					Title:             "Fixture Album",
					NormalizedTitle:   "fixture album",
					Artists:           []string{"Fixture Artist"},
					NormalizedArtists: []string{"fixture artist"},
					ReleaseDate:       "2024-02-03",
					UPC:               "123456789012",
				},
				CandidateID: "spotify-1",
				MatchURL:    "https://open.spotify.com/album/spotify-1",
			}}}},
		)
	}
	defer func() { resolverFactory = originalFactory }()

	var stdout bytes.Buffer
	err := runResolve([]string{"--format=yaml", "https://fixture.test/source"}, &stdout)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "deezer: https://fixture.test/source")
	assert.Contains(t, stdout.String(), "spotify: https://open.spotify.com/album/spotify-1")
}

func TestRunResolveCSVFixtureOutput(t *testing.T) {
	originalFactory := resolverFactory
	resolverFactory = func(_ ariadne.Config) *ariadne.Resolver {
		return ariadne.NewWithAdapters(
			[]ariadne.SourceAdapter{fixtureSourceAdapterForCLI{albumByURL: map[string]ariadne.CanonicalAlbum{
				"https://fixture.test/source": {
					Service:           ariadne.ServiceDeezer,
					SourceID:          "src-1",
					SourceURL:         "https://fixture.test/source",
					Title:             "Fixture Album",
					NormalizedTitle:   "fixture album",
					Artists:           []string{"Fixture Artist"},
					NormalizedArtists: []string{"fixture artist"},
					ReleaseDate:       "2024-02-03",
					UPC:               "123456789012",
				},
			}}},
			[]ariadne.TargetAdapter{fixtureTargetAdapterForCLI{service: ariadne.ServiceSpotify, upcResults: []ariadne.CandidateAlbum{{
				CanonicalAlbum: ariadne.CanonicalAlbum{
					Service:           ariadne.ServiceSpotify,
					SourceID:          "spotify-1",
					SourceURL:         "https://open.spotify.com/album/spotify-1",
					Title:             "Fixture Album",
					NormalizedTitle:   "fixture album",
					Artists:           []string{"Fixture Artist"},
					NormalizedArtists: []string{"fixture artist"},
					ReleaseDate:       "2024-02-03",
					UPC:               "123456789012",
				},
				CandidateID: "spotify-1",
				MatchURL:    "https://open.spotify.com/album/spotify-1",
			}}}},
		)
	}
	defer func() { resolverFactory = originalFactory }()

	var stdout bytes.Buffer
	err := runResolve([]string{"--format=csv", "https://fixture.test/source"}, &stdout)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "service,url")
	assert.Contains(t, stdout.String(), "deezer,https://fixture.test/source")
	assert.Contains(t, stdout.String(), "spotify,https://open.spotify.com/album/spotify-1")
}

func TestRunResolveVerboseCSVFixtureOutput(t *testing.T) {
	originalFactory := resolverFactory
	resolverFactory = func(_ ariadne.Config) *ariadne.Resolver {
		return ariadne.NewWithAdapters(
			[]ariadne.SourceAdapter{fixtureSourceAdapterForCLI{albumByURL: map[string]ariadne.CanonicalAlbum{
				"https://fixture.test/source": {
					Service:           ariadne.ServiceDeezer,
					SourceID:          "src-1",
					SourceURL:         "https://fixture.test/source",
					Title:             "Fixture Album",
					NormalizedTitle:   "fixture album",
					Artists:           []string{"Fixture Artist"},
					NormalizedArtists: []string{"fixture artist"},
					ReleaseDate:       "2024-02-03",
					UPC:               "123456789012",
				},
			}}},
			[]ariadne.TargetAdapter{fixtureTargetAdapterForCLI{service: ariadne.ServiceSpotify, upcResults: []ariadne.CandidateAlbum{{
				CanonicalAlbum: ariadne.CanonicalAlbum{
					Service:           ariadne.ServiceSpotify,
					SourceID:          "spotify-1",
					SourceURL:         "https://open.spotify.com/album/spotify-1",
					Title:             "Fixture Album",
					NormalizedTitle:   "fixture album",
					Artists:           []string{"Fixture Artist"},
					NormalizedArtists: []string{"fixture artist"},
					ReleaseDate:       "2024-02-03",
					UPC:               "123456789012",
				},
				CandidateID: "spotify-1",
				MatchURL:    "https://open.spotify.com/album/spotify-1",
			}}}},
		)
	}
	defer func() { resolverFactory = originalFactory }()

	var stdout bytes.Buffer
	err := runResolve([]string{"--verbose", "--format=csv", "https://fixture.test/source"}, &stdout)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "input_url,service,kind,url,found,summary,score,album_id,region_hint,title,artists,release_date,upc,reasons")
	assert.Contains(t, stdout.String(), ",deezer,source,https://fixture.test/source,true,source,")
	assert.Contains(t, stdout.String(), ",spotify,best,https://open.spotify.com/album/spotify-1,true,strong,155,spotify-1,")
}

func TestRunResolvePropagatesResolverErrors(t *testing.T) {
	originalFactory := resolverFactory
	resolverFactory = func(_ ariadne.Config) *ariadne.Resolver {
		return ariadne.NewWithAdapters(
			[]ariadne.SourceAdapter{fixtureSourceAdapterForCLI{albumByURL: map[string]ariadne.CanonicalAlbum{
				"https://fixture.test/source": {
					Service:   ariadne.ServiceDeezer,
					SourceID:  "src-1",
					SourceURL: "https://fixture.test/source",
					Title:     "Fixture Album",
				},
			}}},
			[]ariadne.TargetAdapter{fixtureTargetAdapterForCLI{service: ariadne.ServiceSpotify, metadataErr: errCLIResolveBoom}},
		)
	}
	defer func() { resolverFactory = originalFactory }()

	var stdout bytes.Buffer
	err := runResolve([]string{"https://fixture.test/source"}, &stdout)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boom")
}

func TestLoadCLIConfigFromDotEnv(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".env")
	content := strings.Join([]string{
		"SPOTIFY_CLIENT_ID=spotify-client",
		"SPOTIFY_CLIENT_SECRET=spotify-secret",
		"APPLE_MUSIC_STOREFRONT=gb",
		"APPLE_MUSIC_KEY_ID=apple-key",
		"APPLE_MUSIC_TEAM_ID=apple-team",
		"APPLE_MUSIC_PRIVATE_KEY_PATH=/tmp/AuthKey_TEST.p8",
		"TIDAL_CLIENT_ID=tidal-client",
		"TIDAL_CLIENT_SECRET=tidal-secret",
		"ARIADNE_HTTP_TIMEOUT=45s",
	}, "\n")
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o644))

	cfg, err := loadCLIConfig(configPath)
	require.NoError(t, err)
	assert.Equal(t, "spotify-client", cfg.Spotify.ClientID)
	assert.Equal(t, "spotify-secret", cfg.Spotify.ClientSecret)
	assert.Equal(t, "gb", cfg.AppleMusicStorefront)
	assert.Equal(t, "apple-key", cfg.AppleMusic.KeyID)
	assert.Equal(t, "apple-team", cfg.AppleMusic.TeamID)
	assert.Equal(t, "/tmp/AuthKey_TEST.p8", cfg.AppleMusic.PrivateKeyPath)
	assert.Equal(t, "tidal-client", cfg.TIDAL.ClientID)
	assert.Equal(t, "tidal-secret", cfg.TIDAL.ClientSecret)
	assert.Equal(t, 45*time.Second, cfg.HTTPTimeout)
}

func TestLoadCLIConfigEnvironmentOverridesFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(configPath, []byte("APPLE_MUSIC_STOREFRONT=gb\nSPOTIFY_CLIENT_ID=file-client\n"), 0o644))
	t.Setenv("APPLE_MUSIC_STOREFRONT", "de")
	t.Setenv("SPOTIFY_CLIENT_ID", "env-client")
	t.Setenv("ARIADNE_HTTP_TIMEOUT", "30s")

	cfg, err := loadCLIConfig(configPath)
	require.NoError(t, err)
	assert.Equal(t, "de", cfg.AppleMusicStorefront)
	assert.Equal(t, "env-client", cfg.Spotify.ClientID)
	assert.Equal(t, 30*time.Second, cfg.HTTPTimeout)
}

func TestParseResolveArgs(t *testing.T) {
	t.Setenv("APPLE_MUSIC_STOREFRONT", "de")

	tests := []struct {
		name            string
		args            []string
		wantURL         string
		wantStorefront  string
		wantFormat      string
		wantMinStrength ariadne.MatchStrength
		wantServices    []ariadne.ServiceName
		wantHTTPTimeout time.Duration
		wantErrContains string
	}{
		{
			name:            "uses env default storefront",
			args:            []string{"https://www.deezer.com/album/12047952"},
			wantURL:         "https://www.deezer.com/album/12047952",
			wantStorefront:  "de",
			wantFormat:      "json",
			wantMinStrength: ariadne.MatchStrengthVeryWeak,
		},
		{
			name:            "verbose flag",
			args:            []string{"--verbose", "https://www.deezer.com/album/12047952"},
			wantURL:         "https://www.deezer.com/album/12047952",
			wantStorefront:  "de",
			wantFormat:      "json",
			wantMinStrength: ariadne.MatchStrengthVeryWeak,
		},
		{
			name:            "yaml format",
			args:            []string{"--format=yaml", "https://www.deezer.com/album/12047952"},
			wantURL:         "https://www.deezer.com/album/12047952",
			wantStorefront:  "de",
			wantFormat:      "yaml",
			wantMinStrength: ariadne.MatchStrengthVeryWeak,
		},
		{
			name:            "service filter",
			args:            []string{"--services=deezer,bandcamp", "https://www.deezer.com/album/12047952"},
			wantURL:         "https://www.deezer.com/album/12047952",
			wantStorefront:  "de",
			wantFormat:      "json",
			wantMinStrength: ariadne.MatchStrengthVeryWeak,
			wantServices:    []ariadne.ServiceName{ariadne.ServiceDeezer, ariadne.ServiceBandcamp},
		},
		{
			name:            "flag overrides env storefront",
			args:            []string{"--apple-music-storefront=gb", "https://www.deezer.com/album/12047952"},
			wantURL:         "https://www.deezer.com/album/12047952",
			wantStorefront:  "gb",
			wantFormat:      "json",
			wantMinStrength: ariadne.MatchStrengthVeryWeak,
		},
		{
			name:            "missing url",
			args:            []string{"--apple-music-storefront=gb"},
			wantErrContains: "usage: ariadne resolve [--song|--album] [--verbose] [--format=json|yaml|csv] [--services=spotify,deezer] [--min-strength=probable] [--apple-music-storefront=us] [--resolution-timeout=20s] <url>",
		},
		{
			name:            "force song",
			args:            []string{"--song", "https://open.spotify.com/track/123"},
			wantURL:         "https://open.spotify.com/track/123",
			wantStorefront:  "de",
			wantFormat:      "json",
			wantMinStrength: ariadne.MatchStrengthVeryWeak,
		},
		{
			name:            "force album",
			args:            []string{"--album", "https://www.deezer.com/album/12047952"},
			wantURL:         "https://www.deezer.com/album/12047952",
			wantStorefront:  "de",
			wantFormat:      "json",
			wantMinStrength: ariadne.MatchStrengthVeryWeak,
		},
		{
			name:            "conflicting entity flags",
			args:            []string{"--song", "--album", "https://open.spotify.com/track/123"},
			wantErrContains: "--song and --album are mutually exclusive",
		},
		{
			name:            "unsupported service",
			args:            []string{"--services=amazonMusic", "https://www.deezer.com/album/12047952"},
			wantErrContains: "amazonMusic is not available as a target service",
		},
		{
			name:            "unsupported song target service",
			args:            []string{"--song", "--services=youtubeMusic", "https://open.spotify.com/track/123"},
			wantErrContains: "target service is not available for song resolution \"youtubeMusic\"",
		},
		{
			name:            "unsupported auto song target service",
			args:            []string{"--services=youtubeMusic", "https://open.spotify.com/track/123"},
			wantErrContains: "target service is not available for song resolution \"youtubeMusic\"",
		},
		{
			name:            "min strength",
			args:            []string{"--min-strength=probable", "https://www.deezer.com/album/12047952"},
			wantURL:         "https://www.deezer.com/album/12047952",
			wantStorefront:  "de",
			wantFormat:      "json",
			wantMinStrength: ariadne.MatchStrengthProbable,
		},
		{
			name:            "http timeout flag",
			args:            []string{"--http-timeout=45s", "https://www.deezer.com/album/12047952"},
			wantURL:         "https://www.deezer.com/album/12047952",
			wantStorefront:  "de",
			wantFormat:      "json",
			wantMinStrength: ariadne.MatchStrengthVeryWeak,
			wantHTTPTimeout: 45 * time.Second,
		},
		{
			name:            "resolution timeout flag",
			args:            []string{"--resolution-timeout=45s", "https://www.deezer.com/album/12047952"},
			wantURL:         "https://www.deezer.com/album/12047952",
			wantStorefront:  "de",
			wantFormat:      "json",
			wantMinStrength: ariadne.MatchStrengthVeryWeak,
		},
		{
			name:            "invalid format",
			args:            []string{"--format=xml", "https://www.deezer.com/album/12047952"},
			wantErrContains: "unsupported format \"xml\"",
		},
		{
			name:            "invalid min strength",
			args:            []string{"--min-strength=excellent", "https://www.deezer.com/album/12047952"},
			wantErrContains: "unsupported min-strength \"excellent\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolveConfig, err := parseResolveArgs(tt.args, ariadne.LoadConfig())
			if tt.wantErrContains != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrContains)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantURL, resolveConfig.inputURL)
			assert.Equal(t, tt.wantStorefront, resolveConfig.resolverConfig.AppleMusicStorefront)
			assert.Equal(t, tt.wantFormat, resolveConfig.format)
			assert.Equal(t, tt.wantMinStrength, resolveConfig.minStrength)
			if tt.wantMinStrength == "" {
				assert.Equal(t, ariadne.MatchStrengthVeryWeak, resolveConfig.minStrength)
			}
			wantHTTPTimeout := tt.wantHTTPTimeout
			if wantHTTPTimeout == 0 {
				wantHTTPTimeout = 15 * time.Second
			}
			assert.Equal(t, wantHTTPTimeout, resolveConfig.resolverConfig.HTTPTimeout)
			wantResolutionTimeout := 20 * time.Second
			if tt.name == "resolution timeout flag" {
				wantResolutionTimeout = 45 * time.Second
			}
			assert.Equal(t, wantResolutionTimeout, resolveConfig.resolutionTimeout)
			assert.Len(t, resolveConfig.resolverConfig.TargetServices, len(tt.wantServices))
			for i, service := range tt.wantServices {
				assert.Equal(t, service, resolveConfig.resolverConfig.TargetServices[i])
			}
			if tt.name == "verbose flag" {
				assert.True(t, resolveConfig.verbose)
			}
			if tt.name == "force song" {
				assert.True(t, resolveConfig.forceSong)
			}
			if tt.name == "force album" {
				assert.True(t, resolveConfig.forceAlbum)
			}
		})
	}
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

type fixtureSourceAdapterForCLI struct {
	albumByURL map[string]ariadne.CanonicalAlbum
}

func (a fixtureSourceAdapterForCLI) Service() ariadne.ServiceName {
	return "fixture"
}

func (a fixtureSourceAdapterForCLI) ParseAlbumURL(raw string) (*ariadne.ParsedAlbumURL, error) {
	album, ok := a.albumByURL[raw]
	if !ok {
		return nil, errUnsupportedCLIFixture
	}
	return &ariadne.ParsedAlbumURL{Service: album.Service, EntityType: "album", ID: album.SourceID, CanonicalURL: raw, RawURL: raw}, nil
}

func (a fixtureSourceAdapterForCLI) FetchAlbum(_ context.Context, parsed ariadne.ParsedAlbumURL) (*ariadne.CanonicalAlbum, error) {
	album, ok := a.albumByURL[parsed.RawURL]
	if !ok {
		return nil, errCLIFixtureNotFound
	}
	albumCopy := album
	return &albumCopy, nil
}

type fixtureTargetAdapterForCLI struct {
	service     ariadne.ServiceName
	upcResults  []ariadne.CandidateAlbum
	isrcResults []ariadne.CandidateAlbum
	metaResults []ariadne.CandidateAlbum
	metadataErr error
}

func (a fixtureTargetAdapterForCLI) Service() ariadne.ServiceName {
	return a.service
}

func (a fixtureTargetAdapterForCLI) SearchByUPC(_ context.Context, _ string) ([]ariadne.CandidateAlbum, error) {
	return append([]ariadne.CandidateAlbum(nil), a.upcResults...), nil
}

func (a fixtureTargetAdapterForCLI) SearchByISRC(_ context.Context, _ []string) ([]ariadne.CandidateAlbum, error) {
	return append([]ariadne.CandidateAlbum(nil), a.isrcResults...), nil
}

func (a fixtureTargetAdapterForCLI) SearchByMetadata(_ context.Context, _ ariadne.CanonicalAlbum) ([]ariadne.CandidateAlbum, error) {
	if a.metadataErr != nil {
		return nil, a.metadataErr
	}
	return append([]ariadne.CandidateAlbum(nil), a.metaResults...), nil
}

type fixtureSongSourceAdapterForCLI struct {
	songByURL map[string]ariadne.CanonicalSong
}

func (a fixtureSongSourceAdapterForCLI) Service() ariadne.ServiceName {
	return "fixture-song"
}

func (a fixtureSongSourceAdapterForCLI) ParseSongURL(raw string) (*ariadne.ParsedURL, error) {
	song, ok := a.songByURL[raw]
	if !ok {
		return nil, errUnsupportedCLIFixture
	}
	return &ariadne.ParsedURL{Service: song.Service, EntityType: "song", ID: song.SourceID, CanonicalURL: raw, RawURL: raw}, nil
}

func (a fixtureSongSourceAdapterForCLI) FetchSong(_ context.Context, parsed ariadne.ParsedURL) (*ariadne.CanonicalSong, error) {
	song, ok := a.songByURL[parsed.RawURL]
	if !ok {
		return nil, errCLIFixtureNotFound
	}
	songCopy := song
	return &songCopy, nil
}

type fixtureSongTargetAdapterForCLI struct {
	service     ariadne.ServiceName
	isrcResults []ariadne.CandidateSong
	metaResults []ariadne.CandidateSong
	metadataErr error
}

func (a fixtureSongTargetAdapterForCLI) Service() ariadne.ServiceName {
	return a.service
}

func (a fixtureSongTargetAdapterForCLI) SearchSongByISRC(_ context.Context, _ string) ([]ariadne.CandidateSong, error) {
	return append([]ariadne.CandidateSong(nil), a.isrcResults...), nil
}

func (a fixtureSongTargetAdapterForCLI) SearchSongByMetadata(_ context.Context, _ ariadne.CanonicalSong) ([]ariadne.CandidateSong, error) {
	if a.metadataErr != nil {
		return nil, a.metadataErr
	}
	return append([]ariadne.CandidateSong(nil), a.metaResults...), nil
}
