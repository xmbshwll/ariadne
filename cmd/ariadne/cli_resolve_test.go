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
	originalFactory := resolverFactory
	resolverFactory = func(_ ariadne.Config) *ariadne.Resolver {
		return ariadne.NewWithAdapters(ariadne.AdapterSet{
			AlbumSources: []ariadne.SourceAdapter{newFixtureSourceAdapterForCLI(map[string]ariadne.CanonicalAlbum{
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
			})},
			AlbumTargets: []ariadne.TargetAdapter{
				newFixtureTargetAdapterForCLI(ariadne.ServiceSpotify, []ariadne.CandidateAlbum{{
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
				}}, nil),
				newFixtureTargetAdapterForCLI(ariadne.ServiceYouTubeMusic, nil, nil),
			},
		})
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
		return ariadne.NewWithAdapters(ariadne.AdapterSet{
			AlbumSources: []ariadne.SourceAdapter{newFixtureSourceAdapterForCLI(map[string]ariadne.CanonicalAlbum{
				"https://fixture.test/source": {
					Service:   ariadne.ServiceDeezer,
					SourceID:  "src-1",
					SourceURL: "https://fixture.test/source",
					Title:     "Fixture Album",
				},
			})},
			AlbumTargets: []ariadne.TargetAdapter{newFixtureTargetAdapterForCLI(ariadne.ServiceSpotify, nil, nil)},
			SongSources: []ariadne.SongSourceAdapter{newFixtureSongSourceAdapterForCLI(map[string]ariadne.CanonicalSong{
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
			SongTargets: []ariadne.SongTargetAdapter{newFixtureSongTargetAdapterForCLI(ariadne.ServiceAppleMusic, []ariadne.CandidateSong{{
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
		})
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
		return ariadne.NewWithAdapters(ariadne.AdapterSet{
			SongSources: []ariadne.SongSourceAdapter{newFixtureSongSourceAdapterForCLI(map[string]ariadne.CanonicalSong{
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
			})},
			SongTargets: []ariadne.SongTargetAdapter{newFixtureSongTargetAdapterForCLI(ariadne.ServiceAppleMusic, []ariadne.CandidateSong{{
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
			}}, nil)},
		})
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
			targets = append(targets, newFixtureTargetAdapterForCLI(ariadne.ServiceDeezer, []ariadne.CandidateAlbum{{
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
			}}, nil))
		}
		return ariadne.NewWithAdapters(ariadne.AdapterSet{
			AlbumSources: []ariadne.SourceAdapter{newFixtureSourceAdapterForCLI(map[string]ariadne.CanonicalAlbum{
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
			})},
			AlbumTargets: targets,
		})
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
			err := runResolve([]string{"--format=" + tt.format, "https://fixture.test/source"}, &stdout)
			require.NoError(t, err)
			for _, want := range tt.want {
				assert.Contains(t, stdout.String(), want)
			}
		})
	}
}

// installSimpleAlbumFixtureResolver stubs the resolver factory with a one-album
// source and one Spotify target candidate.
func installSimpleAlbumFixtureResolver(t *testing.T) {
	t.Helper()
	withResolverFactory(t, func(_ ariadne.Config) *ariadne.Resolver {
		return ariadne.NewWithAdapters(ariadne.AdapterSet{
			AlbumSources: []ariadne.SourceAdapter{newFixtureSourceAdapterForCLI(map[string]ariadne.CanonicalAlbum{
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
			})},
			AlbumTargets: []ariadne.TargetAdapter{newFixtureTargetAdapterForCLI(ariadne.ServiceSpotify, []ariadne.CandidateAlbum{{
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
			}}, nil)},
		})
	})
}

func TestRunResolveVerboseCSVFixtureOutput(t *testing.T) {
	originalFactory := resolverFactory
	resolverFactory = func(_ ariadne.Config) *ariadne.Resolver {
		return ariadne.NewWithAdapters(ariadne.AdapterSet{
			AlbumSources: []ariadne.SourceAdapter{newFixtureSourceAdapterForCLI(map[string]ariadne.CanonicalAlbum{
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
			})},
			AlbumTargets: []ariadne.TargetAdapter{newFixtureTargetAdapterForCLI(ariadne.ServiceSpotify, []ariadne.CandidateAlbum{{
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
			}}, nil)},
		})
	}
	defer func() { resolverFactory = originalFactory }()

	var stdout bytes.Buffer
	err := runResolve([]string{"--verbose", "--format=csv", "https://fixture.test/source"}, &stdout)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "input_url,service,kind,url,found,summary,score,album_id,region_hint,title,artists,release_date,upc,reasons")
	assert.Contains(t, stdout.String(), ",deezer,source,https://fixture.test/source,true,source,")
	assert.Contains(t, stdout.String(), ",spotify,best,https://open.spotify.com/album/spotify-1,true,strong,155,spotify-1,")
}

func TestRunResolvePropagatesTargetErrors(t *testing.T) {
	tests := []struct {
		name    string
		factory func(ariadne.Config) *ariadne.Resolver
		args    []string
	}{
		{
			name: "album target failure",
			args: []string{"https://fixture.test/source"},
			factory: func(_ ariadne.Config) *ariadne.Resolver {
				return ariadne.NewWithAdapters(ariadne.AdapterSet{
					AlbumSources: []ariadne.SourceAdapter{newFixtureSourceAdapterForCLI(map[string]ariadne.CanonicalAlbum{
						"https://fixture.test/source": {
							Service:   ariadne.ServiceDeezer,
							SourceID:  "src-1",
							SourceURL: "https://fixture.test/source",
							Title:     "Fixture Album",
						},
					})},
					AlbumTargets: []ariadne.TargetAdapter{newFixtureTargetAdapterForCLI(ariadne.ServiceSpotify, nil, errCLIResolveBoom)},
				})
			},
		},
		{
			name: "forced song target failure",
			args: []string{"--song", "https://fixture.test/songs/1"},
			factory: func(_ ariadne.Config) *ariadne.Resolver {
				return ariadne.NewWithAdapters(ariadne.AdapterSet{
					SongSources: []ariadne.SongSourceAdapter{newFixtureSongSourceAdapterForCLI(map[string]ariadne.CanonicalSong{
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
					SongTargets: []ariadne.SongTargetAdapter{newFixtureSongTargetAdapterForCLI(ariadne.ServiceTIDAL, nil, errCLIResolveBoom)},
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withResolverFactory(t, tt.factory)

			var stdout bytes.Buffer
			err := runResolve(tt.args, &stdout)
			require.Error(t, err)
			assert.ErrorIs(t, err, errAllTargetSearchesFailed)
			assert.ErrorIs(t, err, errCLIResolveBoom)
		})
	}
}
