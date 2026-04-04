package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	amazonmusicadapter "github.com/xmbshwll/ariadne/internal/adapters/amazonmusic"
	tidaladapter "github.com/xmbshwll/ariadne/internal/adapters/tidal"
	"github.com/xmbshwll/ariadne/internal/config"
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/resolve"
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
			name:       "help",
			args:       []string{"help"},
			wantStdout: []string{"Usage:", "ariadne resolve [--apple-music-storefront=us] <album-url>", "Spotify source fetch works without credentials", "Apple Music source fetch and metadata search work without extra credentials; UPC and ISRC target search are enabled", "SoundCloud source fetch and metadata search use public page hydration plus public-facing api-v2 search and remain experimental", "YouTube Music source fetch and metadata search use public HTML extraction with a browser-like user-agent and remain experimental", "TIDAL source fetch and target search both require TIDAL_CLIENT_ID and TIDAL_CLIENT_SECRET", "Amazon Music album URLs are recognized, but runtime resolving remains deferred", "Apple Music target search defaults to APPLE_MUSIC_STOREFRONT or us"},
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
			wantErr:     "usage: ariadne resolve [--apple-music-storefront=us] <album-url>",
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
	resolution := resolve.Resolution{
		InputURL: "https://open.spotify.com/album/abc",
		Source: model.CanonicalAlbum{
			Service:      model.ServiceSpotify,
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
		Matches: map[model.ServiceName]resolve.MatchResult{
			model.ServiceDeezer: {
				Service: model.ServiceDeezer,
				Best: &resolve.ScoredMatch{
					URL:     "https://www.deezer.com/album/1",
					Score:   140,
					Reasons: []string{"upc exact match", "title exact match"},
					Candidate: model.CandidateAlbum{
						CandidateID: "1",
						CanonicalAlbum: model.CanonicalAlbum{
							Title:       "Album",
							Artists:     []string{"Artist"},
							ReleaseDate: "2024-01-01",
							UPC:         "123",
						},
					},
				},
			},
			model.ServiceAppleMusic: {
				Service: model.ServiceAppleMusic,
				Best: &resolve.ScoredMatch{
					URL:     "https://music.apple.com/us/album/album/2",
					Score:   95,
					Reasons: []string{"title exact match", "primary artist exact match"},
					Candidate: model.CandidateAlbum{
						CandidateID: "2",
						CanonicalAlbum: model.CanonicalAlbum{
							RegionHint:  "us",
							Title:       "Album",
							Artists:     []string{"Artist"},
							ReleaseDate: "2024-01-01",
						},
					},
				},
			},
			model.ServiceTIDAL: {
				Service: model.ServiceTIDAL,
				Best: &resolve.ScoredMatch{
					URL:     "https://tidal.com/album/3",
					Score:   88,
					Reasons: []string{"upc exact match", "track isrc overlap"},
					Candidate: model.CandidateAlbum{
						CandidateID: "3",
						CanonicalAlbum: model.CanonicalAlbum{
							Title:       "Album",
							Artists:     []string{"Artist"},
							ReleaseDate: "2024-01-01",
							UPC:         "123",
						},
					},
				},
				Alternates: []resolve.ScoredMatch{{
					URL:     "https://tidal.com/album/4",
					Score:   59,
					Reasons: []string{"title exact match"},
					Candidate: model.CandidateAlbum{
						CandidateID: "4",
						CanonicalAlbum: model.CanonicalAlbum{
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

func TestNewResolverRequiresCredentialsForTIDALSourceFetch(t *testing.T) {
	resolver := newResolver(config.Config{}, "us")

	_, err := resolver.ResolveAlbum(context.Background(), "https://tidal.com/album/156205493")
	if err == nil {
		t.Fatalf("expected TIDAL credential error")
	}
	if !errors.Is(err, tidaladapter.ErrCredentialsNotConfigured) {
		t.Fatalf("error = %v, want tidal credential error", err)
	}
}

func TestNewResolverReportsAmazonMusicAsDeferred(t *testing.T) {
	resolver := newResolver(config.Config{}, "us")

	_, err := resolver.ResolveAlbum(context.Background(), "https://music.amazon.com/albums/B0064UPU4G")
	if err == nil {
		t.Fatalf("expected Amazon Music deferred error")
	}
	if !errors.Is(err, amazonmusicadapter.ErrDeferredRuntimeAdapter) {
		t.Fatalf("error = %v, want amazon music deferred error", err)
	}
}

func TestRunResolveFixtureOutput(t *testing.T) {
	originalFactory := resolverFactory
	resolverFactory = func(_ config.Config, _ string) *resolve.Resolver {
		return resolve.New(
			[]resolve.SourceAdapter{fixtureSourceAdapterForCLI{albumByURL: map[string]model.CanonicalAlbum{
				"https://fixture.test/source": {
					Service:           model.ServiceDeezer,
					SourceID:          "src-1",
					SourceURL:         "https://fixture.test/source",
					Title:             "Fixture Album",
					NormalizedTitle:   "fixture album",
					Artists:           []string{"Fixture Artist"},
					NormalizedArtists: []string{"fixture artist"},
					ReleaseDate:       "2024-02-03",
					UPC:               "123456789012",
					TrackCount:        2,
					Tracks:            []model.CanonicalTrack{{Title: "Alpha", NormalizedTitle: "alpha", ISRC: "ISRC001"}, {Title: "Beta", NormalizedTitle: "beta"}},
				},
			}}},
			[]resolve.TargetAdapter{
				fixtureTargetAdapterForCLI{service: model.ServiceSpotify, upcResults: []model.CandidateAlbum{{
					CanonicalAlbum: model.CanonicalAlbum{
						Service:           model.ServiceSpotify,
						SourceID:          "spotify-1",
						SourceURL:         "https://open.spotify.com/album/spotify-1",
						Title:             "Fixture Album",
						NormalizedTitle:   "fixture album",
						Artists:           []string{"Fixture Artist"},
						NormalizedArtists: []string{"fixture artist"},
						ReleaseDate:       "2024-02-03",
						UPC:               "123456789012",
						TrackCount:        2,
						Tracks:            []model.CanonicalTrack{{Title: "Alpha", NormalizedTitle: "alpha", ISRC: "ISRC001"}, {Title: "Beta", NormalizedTitle: "beta"}},
					},
					CandidateID: "spotify-1",
					MatchURL:    "https://open.spotify.com/album/spotify-1",
				}}},
				fixtureTargetAdapterForCLI{service: model.ServiceYouTubeMusic},
			},
		)
	}
	defer func() { resolverFactory = originalFactory }()

	var stdout bytes.Buffer
	err := runResolve([]string{"https://fixture.test/source"}, &stdout)
	if err != nil {
		t.Fatalf("runResolve error: %v", err)
	}

	var output cliResolution
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("unmarshal cli output: %v", err)
	}
	if output.Source.Service != "deezer" {
		t.Fatalf("source service = %q, want deezer", output.Source.Service)
	}
	if output.Source.Title != "Fixture Album" {
		t.Fatalf("source title = %q", output.Source.Title)
	}
	spotify := output.Links["spotify"]
	if !spotify.Found {
		t.Fatalf("expected spotify match to be found")
	}
	if spotify.Summary != "strong" {
		t.Fatalf("spotify summary = %q, want strong", spotify.Summary)
	}
	if spotify.Best == nil || spotify.Best.AlbumID != "spotify-1" {
		t.Fatalf("unexpected spotify best match: %#v", spotify.Best)
	}
	youtubeMusic := output.Links["youtubeMusic"]
	if youtubeMusic.Found {
		t.Fatalf("expected youtubeMusic to be not found")
	}
	if youtubeMusic.Summary != "not_found" {
		t.Fatalf("youtubeMusic summary = %q, want not_found", youtubeMusic.Summary)
	}
}

func TestRunResolvePropagatesResolverErrors(t *testing.T) {
	originalFactory := resolverFactory
	resolverFactory = func(_ config.Config, _ string) *resolve.Resolver {
		return resolve.New(
			[]resolve.SourceAdapter{fixtureSourceAdapterForCLI{albumByURL: map[string]model.CanonicalAlbum{
				"https://fixture.test/source": {
					Service:   model.ServiceDeezer,
					SourceID:  "src-1",
					SourceURL: "https://fixture.test/source",
					Title:     "Fixture Album",
				},
			}}},
			[]resolve.TargetAdapter{fixtureTargetAdapterForCLI{service: model.ServiceSpotify, metadataErr: errors.New("boom")}},
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

func TestParseResolveArgs(t *testing.T) {
	t.Setenv("APPLE_MUSIC_STOREFRONT", "de")

	tests := []struct {
		name            string
		args            []string
		wantURL         string
		wantStorefront  string
		wantErrContains string
	}{
		{
			name:           "uses env default storefront",
			args:           []string{"https://www.deezer.com/album/12047952"},
			wantURL:        "https://www.deezer.com/album/12047952",
			wantStorefront: "de",
		},
		{
			name:           "flag overrides env storefront",
			args:           []string{"--apple-music-storefront=gb", "https://www.deezer.com/album/12047952"},
			wantURL:        "https://www.deezer.com/album/12047952",
			wantStorefront: "gb",
		},
		{
			name:            "missing url",
			args:            []string{"--apple-music-storefront=gb"},
			wantErrContains: "usage: ariadne resolve [--apple-music-storefront=us] <album-url>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolveConfig, err := parseResolveArgs(tt.args, config.Load())
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
			if resolveConfig.appleMusicStorefront != tt.wantStorefront {
				t.Fatalf("appleMusicStorefront = %q, want %q", resolveConfig.appleMusicStorefront, tt.wantStorefront)
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
	albumByURL map[string]model.CanonicalAlbum
}

func (a fixtureSourceAdapterForCLI) Service() model.ServiceName {
	return "fixture"
}

func (a fixtureSourceAdapterForCLI) ParseAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	album, ok := a.albumByURL[raw]
	if !ok {
		return nil, errors.New("unsupported")
	}
	return &model.ParsedAlbumURL{Service: album.Service, EntityType: "album", ID: album.SourceID, CanonicalURL: raw, RawURL: raw}, nil
}

func (a fixtureSourceAdapterForCLI) FetchAlbum(_ context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalAlbum, error) {
	album, ok := a.albumByURL[parsed.RawURL]
	if !ok {
		return nil, errors.New("not found")
	}
	albumCopy := album
	return &albumCopy, nil
}

type fixtureTargetAdapterForCLI struct {
	service     model.ServiceName
	upcResults  []model.CandidateAlbum
	isrcResults []model.CandidateAlbum
	metaResults []model.CandidateAlbum
	metadataErr error
}

func (a fixtureTargetAdapterForCLI) Service() model.ServiceName {
	return a.service
}

func (a fixtureTargetAdapterForCLI) SearchByUPC(_ context.Context, _ string) ([]model.CandidateAlbum, error) {
	return append([]model.CandidateAlbum(nil), a.upcResults...), nil
}

func (a fixtureTargetAdapterForCLI) SearchByISRC(_ context.Context, _ []string) ([]model.CandidateAlbum, error) {
	return append([]model.CandidateAlbum(nil), a.isrcResults...), nil
}

func (a fixtureTargetAdapterForCLI) SearchByMetadata(_ context.Context, _ model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
	if a.metadataErr != nil {
		return nil, a.metadataErr
	}
	return append([]model.CandidateAlbum(nil), a.metaResults...), nil
}
