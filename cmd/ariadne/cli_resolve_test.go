package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xmbshwll/ariadne"
)

var errCLIResolveBoom = errors.New("boom")

const cliFixtureAlbumURL = "https://fixture.test/source"

// fixtureSourceAlbum is the album Source Input most CLI tests render.
var fixtureSourceAlbum = ariadne.CanonicalAlbum{
	Service:           ariadne.ServiceDeezer,
	SourceID:          "src-1",
	SourceURL:         cliFixtureAlbumURL,
	Title:             "Fixture Album",
	NormalizedTitle:   "fixture album",
	Artists:           []string{"Fixture Artist"},
	NormalizedArtists: []string{"fixture artist"},
	ReleaseDate:       "2024-02-03",
	UPC:               "123456789012",
	TrackCount:        2,
	Tracks: []ariadne.CanonicalTrack{
		{Title: "Alpha", NormalizedTitle: "alpha", ISRC: "ISRC001"},
		{Title: "Beta", NormalizedTitle: "beta"},
	},
}

// fixtureSpotifyMatch is the one strong Spotify album match most CLI tests
// expect to see rendered: score 155 ranks as "strong".
var fixtureSpotifyMatch = ariadne.MatchResult{
	Service: ariadne.ServiceSpotify,
	Best: &ariadne.ScoredMatch{
		URL:     "https://open.spotify.com/album/spotify-1",
		Score:   155,
		Reasons: []string{"upc exact match", "title exact match"},
		Candidate: ariadne.CandidateAlbum{
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
				Tracks: []ariadne.CanonicalTrack{
					{Title: "Alpha", NormalizedTitle: "alpha", ISRC: "ISRC001"},
					{Title: "Beta", NormalizedTitle: "beta"},
				},
			},
			CandidateID: "spotify-1",
			MatchURL:    "https://open.spotify.com/album/spotify-1",
		},
	},
}

// fixtureSourceSong is the song Source Input most song-mode CLI tests render.
var fixtureSourceSong = ariadne.CanonicalSong{
	Service:              ariadne.ServiceSpotify,
	SourceID:             "song-1",
	SourceURL:            "https://fixture.test/songs/1",
	RegionHint:           "us",
	Title:                "Fixture Song",
	NormalizedTitle:      "fixture song",
	Artists:              []string{"Fixture Artist"},
	NormalizedArtists:    []string{"fixture artist"},
	DurationMS:           180000,
	ISRC:                 "ISRCSONG001",
	TrackNumber:          1,
	AlbumTitle:           "Fixture Album",
	AlbumNormalizedTitle: "fixture album",
	ReleaseDate:          "2024-02-03",
}

// fixtureAppleSongMatch is the one Apple Music song match most song-mode CLI
// tests expect to see rendered.
var fixtureAppleSongMatch = ariadne.SongMatchResult{
	Service: ariadne.ServiceAppleMusic,
	Best: &ariadne.SongScoredMatch{
		URL:     "https://music.apple.com/us/album/fixture-album/2?i=3",
		Score:   160,
		Reasons: []string{"isrc exact match", "title exact match"},
		Candidate: ariadne.CandidateSong{
			CanonicalSong: ariadne.CanonicalSong{
				Service:              ariadne.ServiceAppleMusic,
				SourceID:             "apple-song-1",
				SourceURL:            "https://music.apple.com/us/album/fixture-album/2?i=3",
				RegionHint:           "us",
				Title:                "Fixture Song",
				NormalizedTitle:      "fixture song",
				Artists:              []string{"Fixture Artist"},
				NormalizedArtists:    []string{"fixture artist"},
				DurationMS:           180050,
				ISRC:                 "ISRCSONG001",
				TrackNumber:          1,
				AlbumTitle:           "Fixture Album",
				AlbumNormalizedTitle: "fixture album",
				ReleaseDate:          "2024-02-03",
			},
			CandidateID: "apple-song-1",
			MatchURL:    "https://music.apple.com/us/album/fixture-album/2?i=3",
		},
	},
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
	assert.ErrorIs(t, err, ariadne.ErrRuntimeDeferred)
	assert.ErrorIs(t, err, ariadne.ErrAmazonMusicDeferred)
}

func TestRunResolveFixtureOutput(t *testing.T) {
	withFixtureResolver(t, fixtureResolverForCLI{
		album: fixtureAlbumResolution(fixtureSourceAlbum, map[ariadne.ServiceName]ariadne.MatchResult{
			ariadne.ServiceSpotify:      fixtureSpotifyMatch,
			ariadne.ServiceYouTubeMusic: {Service: ariadne.ServiceYouTubeMusic},
		}),
	})

	var stdout bytes.Buffer
	err := runResolve([]string{cliFixtureAlbumURL}, &stdout)
	require.NoError(t, err)

	var output map[string]string
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &output))
	assert.Equal(t, cliFixtureAlbumURL, output["deezer"])
	assert.Equal(t, "https://open.spotify.com/album/spotify-1", output["spotify"])
	_, ok := output["youtubeMusic"]
	assert.False(t, ok)
}

func TestRunResolveAutoDispatchesSongFixtureOutput(t *testing.T) {
	withFixtureResolver(t, fixtureResolverForCLI{
		album: fixtureAlbumResolution(fixtureSourceAlbum, map[ariadne.ServiceName]ariadne.MatchResult{
			ariadne.ServiceSpotify: {Service: ariadne.ServiceSpotify},
		}),
		song: fixtureSongResolution(fixtureSourceSong, map[ariadne.ServiceName]ariadne.SongMatchResult{
			ariadne.ServiceAppleMusic: fixtureAppleSongMatch,
		}),
	})

	var stdout bytes.Buffer
	err := runResolve([]string{"https://fixture.test/songs/1"}, &stdout)
	require.NoError(t, err)

	var output map[string]string
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &output))
	assert.Equal(t, "https://fixture.test/songs/1", output["spotify"])
	assert.Equal(t, "https://music.apple.com/us/album/fixture-album/2?i=3", output["appleMusic"])
}

func TestRunResolveForcedSongFixtureOutput(t *testing.T) {
	withFixtureResolver(t, fixtureResolverForCLI{
		song: fixtureSongResolution(fixtureSourceSong, map[ariadne.ServiceName]ariadne.SongMatchResult{
			ariadne.ServiceAppleMusic: fixtureAppleSongMatch,
		}),
	})

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
	var selected []ariadne.ServiceName
	withResolverFactory(t, func(cfg ariadne.Config) entityResolver {
		selected = cfg.TargetServices
		matches := map[ariadne.ServiceName]ariadne.MatchResult{}
		for _, service := range cfg.TargetServices {
			if service != ariadne.ServiceDeezer {
				continue
			}
			matches[service] = ariadne.MatchResult{
				Service: service,
				Best: &ariadne.ScoredMatch{
					URL:   "https://www.deezer.com/album/deezer-1",
					Score: 155,
					Candidate: ariadne.CandidateAlbum{
						CanonicalAlbum: ariadne.CanonicalAlbum{
							Service:     service,
							SourceID:    "deezer-1",
							SourceURL:   "https://www.deezer.com/album/deezer-1",
							Title:       "Fixture Album",
							Artists:     []string{"Fixture Artist"},
							ReleaseDate: "2024-02-03",
							UPC:         "123456789012",
						},
						CandidateID: "deezer-1",
						MatchURL:    "https://www.deezer.com/album/deezer-1",
					},
				},
			}
		}
		source := fixtureSourceAlbum
		source.Service = ariadne.ServiceAppleMusic
		source.SourceURL = cliFixtureAlbumURL
		return fixtureResolverForCLI{album: fixtureAlbumResolution(source, matches)}
	})

	var stdout bytes.Buffer
	err := runResolve([]string{"--services=deezer", cliFixtureAlbumURL}, &stdout)
	require.NoError(t, err)

	// The CLI's job is turning --services into the library's Target Services
	// selection; the Provider Catalog then limits Target Search to it.
	assert.Equal(t, []ariadne.ServiceName{ariadne.ServiceDeezer}, selected)

	var output map[string]string
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &output))
	assert.Equal(t, cliFixtureAlbumURL, output["appleMusic"])
	assert.Equal(t, "https://www.deezer.com/album/deezer-1", output["deezer"])
	_, ok := output["spotify"]
	assert.False(t, ok)
}

func TestRunResolveFormatFixtureOutput(t *testing.T) {
	tests := []struct {
		name   string
		format string
		want   []string
	}{
		{name: "yaml", format: "yaml", want: []string{
			"deezer: https://fixture.test/source",
			"spotify: https://open.spotify.com/album/spotify-1",
		}},
		{name: "csv", format: "csv", want: []string{
			"service,url",
			"deezer,https://fixture.test/source",
			"spotify,https://open.spotify.com/album/spotify-1",
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			installSimpleAlbumFixtureResolver(t)

			var stdout bytes.Buffer
			err := runResolve([]string{"--format=" + tt.format, cliFixtureAlbumURL}, &stdout)
			require.NoError(t, err)
			for _, want := range tt.want {
				assert.Contains(t, stdout.String(), want)
			}
		})
	}
}

// installSimpleAlbumFixtureResolver installs a one-album source with one strong
// Spotify match.
func installSimpleAlbumFixtureResolver(t *testing.T) {
	t.Helper()
	withFixtureResolver(t, fixtureResolverForCLI{
		album: fixtureAlbumResolution(fixtureSourceAlbum, map[ariadne.ServiceName]ariadne.MatchResult{
			ariadne.ServiceSpotify: fixtureSpotifyMatch,
		}),
	})
}

func TestRunResolveVerboseCSVFixtureOutput(t *testing.T) {
	installSimpleAlbumFixtureResolver(t)

	var stdout bytes.Buffer
	err := runResolve([]string{"--verbose", "--format=csv", cliFixtureAlbumURL}, &stdout)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "input_url,service,kind,url,found,summary,score,album_id,region_hint,title,artists,release_date,upc,reasons")
	assert.Contains(t, stdout.String(), ",deezer,source,https://fixture.test/source,true,source,")
	assert.Contains(t, stdout.String(), ",spotify,best,https://open.spotify.com/album/spotify-1,true,strong,155,spotify-1,")
}

func TestRunResolvePropagatesTargetErrors(t *testing.T) {
	tests := []struct {
		name    string
		fixture fixtureResolverForCLI
		args    []string
	}{
		{
			name: "album target failure",
			args: []string{cliFixtureAlbumURL},
			fixture: fixtureResolverForCLI{
				album: fixtureAlbumResolution(fixtureSourceAlbum, map[ariadne.ServiceName]ariadne.MatchResult{
					ariadne.ServiceSpotify: {Service: ariadne.ServiceSpotify, Err: errCLIResolveBoom},
				}),
			},
		},
		{
			name: "forced song target failure",
			args: []string{"--song", "https://fixture.test/songs/1"},
			fixture: fixtureResolverForCLI{
				song: fixtureSongResolution(fixtureSourceSong, map[ariadne.ServiceName]ariadne.SongMatchResult{
					ariadne.ServiceTIDAL: {Service: ariadne.ServiceTIDAL, Err: errCLIResolveBoom},
				}),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withFixtureResolver(t, tt.fixture)

			var stdout bytes.Buffer
			err := runResolve(tt.args, &stdout)
			require.Error(t, err)
			assert.ErrorIs(t, err, errAllTargetSearchesFailed)
			assert.ErrorIs(t, err, errCLIResolveBoom)
		})
	}
}
