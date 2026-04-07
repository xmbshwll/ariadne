package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xmbshwll/ariadne"
)

var (
	errCLIResolveBoom        = errors.New("boom")
	errUnsupportedCLIFixture = errors.New("unsupported")
	errCLIFixtureNotFound    = errors.New("not found")
)

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
				"ariadne resolve [--verbose] [--format=json|yaml|csv] [--services=spotify,deezer] [--min-strength=probable] [--apple-music-storefront=us] <album-url>",
				"<album-url>",
				"Values: a supported album URL from Apple Music, Deezer, Spotify, TIDAL",
				"--config",
				"Behavior: config file values are loaded first, environment variables override them, and explicit CLI flags override both.",
				"--verbose, -v",
				"--format",
				"--services",
				"--min-strength",
				"--apple-music-storefront",
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
			wantErr:     "usage: ariadne resolve [--verbose] [--format=json|yaml|csv] [--services=spotify,deezer] [--min-strength=probable] [--apple-music-storefront=us] <album-url>",
			avoidStdout: []string{"{"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			err := run(tt.args, &stdout, &stderr)
			if tt.wantErr == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %q, want substring %q", err.Error(), tt.wantErr)
				}
			}

			for _, want := range tt.wantStdout {
				if !strings.Contains(stdout.String(), want) {
					t.Fatalf("stdout = %q, want substring %q", stdout.String(), want)
				}
			}
			for _, want := range tt.wantStderr {
				if !strings.Contains(stderr.String(), want) {
					t.Fatalf("stderr = %q, want substring %q", stderr.String(), want)
				}
			}
			for _, avoid := range tt.avoidStdout {
				if strings.Contains(stdout.String(), avoid) {
					t.Fatalf("stdout = %q, should not contain %q", stdout.String(), avoid)
				}
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
	if output.InputURL != resolution.InputURL {
		t.Fatalf("input url = %q, want %q", output.InputURL, resolution.InputURL)
	}
	if output.Source.Service != "spotify" {
		t.Fatalf("source service = %q, want spotify", output.Source.Service)
	}
	if output.Source.ID != "abc" {
		t.Fatalf("source id = %q, want abc", output.Source.ID)
	}
	if output.Source.RegionHint != "us" {
		t.Fatalf("source region hint = %q, want us", output.Source.RegionHint)
	}
	deezer, ok := output.Links["deezer"]
	if !ok {
		t.Fatalf("expected deezer link entry")
	}
	if !deezer.Found {
		t.Fatalf("expected deezer match to be found")
	}
	if deezer.Summary != "strong" {
		t.Fatalf("summary = %q, want strong", deezer.Summary)
	}
	if deezer.Best == nil {
		t.Fatalf("expected deezer best match")
	}
	if deezer.Best.URL != "https://www.deezer.com/album/1" {
		t.Fatalf("best url = %q", deezer.Best.URL)
	}
	if deezer.Best.AlbumID != "1" {
		t.Fatalf("best album id = %q", deezer.Best.AlbumID)
	}
	if deezer.Best.RegionHint != "" {
		t.Fatalf("best region hint = %q, want empty", deezer.Best.RegionHint)
	}
	if len(deezer.Best.Reasons) != 2 {
		t.Fatalf("best reasons len = %d, want 2", len(deezer.Best.Reasons))
	}

	appleMusic, ok := output.Links["appleMusic"]
	if !ok {
		t.Fatalf("expected appleMusic link entry")
	}
	if appleMusic.Best == nil {
		t.Fatalf("expected appleMusic best match")
	}
	if appleMusic.Best.RegionHint != "us" {
		t.Fatalf("appleMusic region hint = %q, want us", appleMusic.Best.RegionHint)
	}

	tidal, ok := output.Links["tidal"]
	if !ok {
		t.Fatalf("expected tidal link entry")
	}
	if !tidal.Found {
		t.Fatalf("expected tidal match to be found")
	}
	if tidal.Summary != "probable" {
		t.Fatalf("tidal summary = %q, want probable", tidal.Summary)
	}
	if tidal.Best == nil {
		t.Fatalf("expected tidal best match")
	}
	if tidal.Best.URL != "https://tidal.com/album/3" {
		t.Fatalf("tidal best url = %q", tidal.Best.URL)
	}
	if tidal.Best.AlbumID != "3" {
		t.Fatalf("tidal best album id = %q", tidal.Best.AlbumID)
	}
	if len(tidal.Alternates) != 1 {
		t.Fatalf("tidal alternates len = %d, want 1", len(tidal.Alternates))
	}
	if tidal.Alternates[0].AlbumID != "4" {
		t.Fatalf("tidal alternate album id = %q, want 4", tidal.Alternates[0].AlbumID)
	}
}

func TestResolverRequiresCredentialsForTIDALSourceFetch(t *testing.T) {
	resolver := ariadne.New(ariadne.DefaultConfig())

	_, err := resolver.ResolveAlbum(context.Background(), "https://tidal.com/album/156205493")
	if err == nil {
		t.Fatalf("expected TIDAL credential error")
	}
	if !errors.Is(err, ariadne.ErrTIDALCredentialsNotConfigured) {
		t.Fatalf("error = %v, want tidal credential error", err)
	}
}

func TestResolverReportsAmazonMusicAsDeferred(t *testing.T) {
	resolver := ariadne.New(ariadne.DefaultConfig())

	_, err := resolver.ResolveAlbum(context.Background(), "https://music.amazon.com/albums/B0064UPU4G")
	if err == nil {
		t.Fatalf("expected Amazon Music deferred error")
	}
	if !errors.Is(err, ariadne.ErrAmazonMusicDeferred) {
		t.Fatalf("error = %v, want amazon music deferred error", err)
	}
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
	if len(output) != 3 {
		t.Fatalf("link count = %d, want 3", len(output))
	}
	if output["deezer"] != "https://www.deezer.com/album/source" {
		t.Fatalf("deezer link = %q", output["deezer"])
	}
	if output["spotify"] != "https://open.spotify.com/album/spotify-1" {
		t.Fatalf("spotify link = %q", output["spotify"])
	}
	if output["appleMusic"] != "https://music.apple.com/us/album/album/2" {
		t.Fatalf("appleMusic link = %q", output["appleMusic"])
	}
	if _, ok := output["youtubeMusic"]; ok {
		t.Fatalf("did not expect youtubeMusic link")
	}
}

func TestFilterResolutionByStrength(t *testing.T) {
	resolution := ariadne.Resolution{
		Source: ariadne.CanonicalAlbum{Service: ariadne.ServiceDeezer, SourceURL: "https://www.deezer.com/album/source"},
		Matches: map[ariadne.ServiceName]ariadne.MatchResult{
			ariadne.ServiceSpotify:    {Best: &ariadne.ScoredMatch{URL: "https://open.spotify.com/album/strong", Score: 120}},
			ariadne.ServiceAppleMusic: {Best: &ariadne.ScoredMatch{URL: "https://music.apple.com/us/album/weak", Score: 55}},
		},
	}

	filtered := filterResolutionByStrength(resolution, ariadne.MatchStrengthProbable)
	if len(filtered.Matches) != 1 {
		t.Fatalf("match count = %d, want 1", len(filtered.Matches))
	}
	if _, ok := filtered.Matches[ariadne.ServiceSpotify]; !ok {
		t.Fatalf("expected spotify to remain")
	}
	if _, ok := filtered.Matches[ariadne.ServiceAppleMusic]; ok {
		t.Fatalf("expected appleMusic to be filtered out")
	}
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
	if err != nil {
		t.Fatalf("runResolve error: %v", err)
	}

	var output map[string]string
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("unmarshal cli output: %v", err)
	}
	if output["deezer"] != "https://fixture.test/source" {
		t.Fatalf("source link = %q", output["deezer"])
	}
	if output["spotify"] != "https://open.spotify.com/album/spotify-1" {
		t.Fatalf("spotify link = %q", output["spotify"])
	}
	if _, ok := output["youtubeMusic"]; ok {
		t.Fatalf("expected youtubeMusic to be omitted")
	}
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
	if err != nil {
		t.Fatalf("runResolve error: %v", err)
	}

	var output map[string]string
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("unmarshal cli output: %v", err)
	}
	if output["appleMusic"] != "https://fixture.test/source" {
		t.Fatalf("source link = %q", output["appleMusic"])
	}
	if output["deezer"] != "https://www.deezer.com/album/deezer-1" {
		t.Fatalf("deezer link = %q", output["deezer"])
	}
	if _, ok := output["spotify"]; ok {
		t.Fatalf("expected spotify to be filtered out")
	}
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
	if err != nil {
		t.Fatalf("runResolve error: %v", err)
	}
	if !strings.Contains(stdout.String(), "deezer: https://fixture.test/source") {
		t.Fatalf("yaml output = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "spotify: https://open.spotify.com/album/spotify-1") {
		t.Fatalf("yaml output = %q", stdout.String())
	}
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
	if err != nil {
		t.Fatalf("runResolve error: %v", err)
	}
	if !strings.Contains(stdout.String(), "service,url") {
		t.Fatalf("csv output = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "deezer,https://fixture.test/source") {
		t.Fatalf("csv output = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "spotify,https://open.spotify.com/album/spotify-1") {
		t.Fatalf("csv output = %q", stdout.String())
	}
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
	if err != nil {
		t.Fatalf("runResolve error: %v", err)
	}
	if !strings.Contains(stdout.String(), "input_url,service,kind,url,found,summary,score,album_id,region_hint,title,artists,release_date,upc,reasons") {
		t.Fatalf("csv output = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), ",deezer,source,https://fixture.test/source,true,source,") {
		t.Fatalf("csv output = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), ",spotify,best,https://open.spotify.com/album/spotify-1,true,strong,155,spotify-1,") {
		t.Fatalf("csv output = %q", stdout.String())
	}
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
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "resolve album") || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("error = %q", err.Error())
	}
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
	}, "\n")
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := loadCLIConfig(configPath)
	if err != nil {
		t.Fatalf("loadCLIConfig error: %v", err)
	}
	if cfg.Spotify.ClientID != "spotify-client" || cfg.Spotify.ClientSecret != "spotify-secret" {
		t.Fatalf("unexpected spotify config: %#v", cfg.Spotify)
	}
	if cfg.AppleMusicStorefront != "gb" {
		t.Fatalf("apple storefront = %q, want gb", cfg.AppleMusicStorefront)
	}
	if cfg.AppleMusic.KeyID != "apple-key" || cfg.AppleMusic.TeamID != "apple-team" || cfg.AppleMusic.PrivateKeyPath != "/tmp/AuthKey_TEST.p8" {
		t.Fatalf("unexpected apple music config: %#v", cfg.AppleMusic)
	}
	if cfg.TIDAL.ClientID != "tidal-client" || cfg.TIDAL.ClientSecret != "tidal-secret" {
		t.Fatalf("unexpected tidal config: %#v", cfg.TIDAL)
	}
}

func TestLoadCLIConfigEnvironmentOverridesFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(configPath, []byte("APPLE_MUSIC_STOREFRONT=gb\nSPOTIFY_CLIENT_ID=file-client\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv("APPLE_MUSIC_STOREFRONT", "de")
	t.Setenv("SPOTIFY_CLIENT_ID", "env-client")

	cfg, err := loadCLIConfig(configPath)
	if err != nil {
		t.Fatalf("loadCLIConfig error: %v", err)
	}
	if cfg.AppleMusicStorefront != "de" {
		t.Fatalf("apple storefront = %q, want de", cfg.AppleMusicStorefront)
	}
	if cfg.Spotify.ClientID != "env-client" {
		t.Fatalf("spotify client id = %q, want env-client", cfg.Spotify.ClientID)
	}
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
			wantErrContains: "usage: ariadne resolve [--verbose] [--format=json|yaml|csv] [--services=spotify,deezer] [--min-strength=probable] [--apple-music-storefront=us] <album-url>",
		},
		{
			name:            "unsupported service",
			args:            []string{"--services=amazonMusic", "https://www.deezer.com/album/12047952"},
			wantErrContains: "amazonMusic is not available as a target service",
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
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErrContains)
				}
				if !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Fatalf("error = %q, want substring %q", err.Error(), tt.wantErrContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resolveConfig.inputURL != tt.wantURL {
				t.Fatalf("inputURL = %q, want %q", resolveConfig.inputURL, tt.wantURL)
			}
			if resolveConfig.resolverConfig.AppleMusicStorefront != tt.wantStorefront {
				t.Fatalf("appleMusicStorefront = %q, want %q", resolveConfig.resolverConfig.AppleMusicStorefront, tt.wantStorefront)
			}
			if resolveConfig.format != tt.wantFormat {
				t.Fatalf("format = %q, want %q", resolveConfig.format, tt.wantFormat)
			}
			if resolveConfig.minStrength != tt.wantMinStrength {
				t.Fatalf("minStrength = %q, want %q", resolveConfig.minStrength, tt.wantMinStrength)
			}
			if tt.wantMinStrength == "" && resolveConfig.minStrength != ariadne.MatchStrengthVeryWeak {
				t.Fatalf("minStrength = %q, want default very_weak", resolveConfig.minStrength)
			}
			if len(resolveConfig.resolverConfig.TargetServices) != len(tt.wantServices) {
				t.Fatalf("services len = %d, want %d", len(resolveConfig.resolverConfig.TargetServices), len(tt.wantServices))
			}
			for i, service := range tt.wantServices {
				if resolveConfig.resolverConfig.TargetServices[i] != service {
					t.Fatalf("service[%d] = %q, want %q", i, resolveConfig.resolverConfig.TargetServices[i], service)
				}
			}
			if tt.name == "verbose flag" && !resolveConfig.verbose {
				t.Fatalf("expected verbose flag to be set")
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
			if got := scoreSummary(tt.score); got != tt.want {
				t.Fatalf("scoreSummary(%d) = %q, want %q", tt.score, got, tt.want)
			}
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
