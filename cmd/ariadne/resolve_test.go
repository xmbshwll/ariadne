package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne"
	"github.com/xmbshwll/ariadne/internal/model"
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

var errUnsupportedCLIFixture = errors.New("unsupported")

// fixtureResolverForCLI stands in for ariadne.Resolver in CLI tests. The CLI's
// job is flags, config, output shapes, and error mapping, so a test describes
// the resolution it wants rendered instead of assembling a whole Entity
// Resolution pipeline; ariadne and internal/resolve tests cover ranking itself.
// A nil album or song means "this input is not that Entity Shape", which is how
// the auto-dispatch path decides.
type fixtureResolverForCLI struct {
	album *ariadne.Resolution
	song  *ariadne.SongResolution
	// albumErr and songErr stand in for a resolver that rejects the URL.
	albumErr error
	songErr  error
}

func (r fixtureResolverForCLI) ResolveAlbum(_ context.Context, _ string) (*ariadne.Resolution, error) {
	if r.albumErr != nil {
		return nil, r.albumErr
	}
	if r.album == nil {
		return nil, errUnsupportedCLIFixture
	}
	return r.album, nil
}

func (r fixtureResolverForCLI) ResolveSong(_ context.Context, _ string) (*ariadne.SongResolution, error) {
	if r.songErr != nil {
		return nil, r.songErr
	}
	if r.song == nil {
		return nil, errUnsupportedCLIFixture
	}
	return r.song, nil
}

// Resolve mirrors the dispatch order the real Resolver uses: song first, album
// when the input is not a song URL.
func (r fixtureResolverForCLI) Resolve(_ context.Context, _ string) (*ariadne.EntityResolution, error) {
	if r.song != nil {
		return &ariadne.EntityResolution{Parsed: r.song.Parsed, Song: r.song}, nil
	}
	if r.album != nil {
		return &ariadne.EntityResolution{Parsed: r.album.Parsed, Album: r.album}, nil
	}
	return nil, errUnsupportedCLIFixture
}

// fixtureAlbumResolution describes one album resolution for a source the CLI is
// asked to resolve: the source's own URL is the input URL.
func fixtureAlbumResolution(source ariadne.CanonicalAlbum, matches map[ariadne.ServiceName]ariadne.MatchResult) *ariadne.Resolution {
	return &ariadne.Resolution{
		InputURL: source.SourceURL,
		Parsed: ariadne.ParsedAlbumURL{
			Service: source.Service, EntityType: model.EntityTypeAlbum,
			ID: source.SourceID, CanonicalURL: source.SourceURL, RawURL: source.SourceURL,
		},
		Source:  source,
		Matches: matches,
	}
}

// fixtureSongResolution describes one song resolution for a source the CLI is
// asked to resolve.
func fixtureSongResolution(source ariadne.CanonicalSong, matches map[ariadne.ServiceName]ariadne.SongMatchResult) *ariadne.SongResolution {
	return &ariadne.SongResolution{
		InputURL: source.SourceURL,
		Parsed: ariadne.ParsedURL{
			Service: source.Service, EntityType: model.EntityTypeSong,
			ID: source.SourceID, CanonicalURL: source.SourceURL, RawURL: source.SourceURL,
		},
		Source:  source,
		Matches: matches,
	}
}

// withResolverFactory installs a resolver factory for one test.
func withResolverFactory(t *testing.T, factory func(ariadne.Config) entityResolver) {
	t.Helper()
	originalFactory := resolverFactory
	resolverFactory = factory
	t.Cleanup(func() {
		resolverFactory = originalFactory
	})
}

// withFixtureResolver installs a fixed set of resolutions for one test.
func withFixtureResolver(t *testing.T, fixture fixtureResolverForCLI) {
	t.Helper()
	withResolverFactory(t, func(ariadne.Config) entityResolver { return fixture })
}

var (
	errUnknownOops     = errors.New("unknown command \"oops\" for \"ariadne\"")
	errDifferentCLIArg = errors.New("different error")
)

func TestArgsWithoutNamedFlagsConsumesExplicitEmptyValue(t *testing.T) {
	args := []string{"--config", "", "resolve", "https://fixture.test/source"}
	assert.Equal(t, []string{"resolve", "https://fixture.test/source"}, argsWithoutNamedFlags(args, "--config"))
}

func TestConfigPathFromArgsPreservesExplicitEmptyValue(t *testing.T) {
	assert.Equal(t, "", configPathFromArgs([]string{"--config", "", "resolve", "https://fixture.test/source"}))
	assert.Equal(t, "", configPathFromArgs([]string{"--config=", "resolve", "https://fixture.test/source"}))
}

func TestIsUnknownCommandError(t *testing.T) {
	assert.False(t, isUnknownCommandError(nil))
	assert.True(t, isUnknownCommandError(errUnknownOops))
	assert.False(t, isUnknownCommandError(errDifferentCLIArg))
}

func TestParseRequestedServicesSkipsEmptySegments(t *testing.T) {
	services, err := parseRequestedServices(" deezer, ,bandcamp,, ", ariadne.Config{})
	require.NoError(t, err)
	assert.Equal(t, []ariadne.ServiceName{ariadne.ServiceDeezer, ariadne.ServiceBandcamp}, services)
}

func TestParseMatchStrengthNormalizesAliases(t *testing.T) {
	for _, raw := range []string{"very_weak", "very-weak", "veryweak", " VERY_WEAK "} {
		strength, err := parseMatchStrength(raw)
		require.NoError(t, err)
		assert.Equal(t, ariadne.MatchStrengthVeryWeak, strength)
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
		"ARIADNE_HTTP_TIMEOUT=45s",
		"ARIADNE_TARGET_SERVICES=spotify,appleMusic,spotify",
	}, "\n")
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o644))

	cfg, err := loadCLIConfigWithLogger(configPath, nil)
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
	assert.Equal(t, []ariadne.ServiceName{ariadne.ServiceSpotify, ariadne.ServiceAppleMusic}, cfg.TargetServices)
}

func TestLoadCLIConfigEnvironmentOverridesFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(configPath, []byte("APPLE_MUSIC_STOREFRONT=gb\nSPOTIFY_CLIENT_ID=file-client\n"), 0o644))
	t.Setenv("APPLE_MUSIC_STOREFRONT", "de")
	t.Setenv("SPOTIFY_CLIENT_ID", "env-client")
	t.Setenv("ARIADNE_HTTP_TIMEOUT", "30s")

	cfg, err := loadCLIConfigWithLogger(configPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "de", cfg.AppleMusicStorefront)
	assert.Equal(t, "env-client", cfg.Spotify.ClientID)
	assert.Equal(t, 30*time.Second, cfg.HTTPTimeout)
}

func TestParseResolveArgsPreservesConfiguredTargetServices(t *testing.T) {
	resolveConfig, err := parseResolveArgs(
		[]string{"https://www.deezer.com/album/12047952"},
		ariadne.Config{
			Spotify:        ariadne.SpotifyConfig{ClientID: "client-id", ClientSecret: "client-secret"},
			TargetServices: []ariadne.ServiceName{ariadne.ServiceSpotify, ariadne.ServiceAppleMusic},
		},
	)
	require.NoError(t, err)
	assert.Equal(t, []ariadne.ServiceName{ariadne.ServiceSpotify, ariadne.ServiceAppleMusic}, resolveConfig.resolverConfig.TargetServices)
}

func TestParseResolveArgsValidatesConfiguredTargetServices(t *testing.T) {
	_, err := parseResolveArgs(
		[]string{"https://www.deezer.com/album/12047952"},
		ariadne.Config{TargetServices: []ariadne.ServiceName{ariadne.ServiceSpotify}},
	)
	require.ErrorIs(t, err, errTargetServiceCredentials)
	require.Contains(t, rootError(err).Error(), "SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET",
		"main prints the unwrapped error, so the Credential Token names must survive unwrapping")
}

func TestLoadCLIConfigRejectsNonPositiveHTTPTimeout(t *testing.T) {
	tests := []struct {
		name        string
		timeout     string
		wantMessage string
	}{
		{name: "zero", timeout: "0s", wantMessage: "invalid ARIADNE_HTTP_TIMEOUT \"0s\": ARIADNE_HTTP_TIMEOUT must be positive"},
		{name: "negative", timeout: "-5s", wantMessage: "invalid ARIADNE_HTTP_TIMEOUT \"-5s\": ARIADNE_HTTP_TIMEOUT must be positive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			configPath := filepath.Join(dir, ".env")
			require.NoError(t, os.WriteFile(configPath, []byte("ARIADNE_HTTP_TIMEOUT="+tt.timeout+"\n"), 0o644))

			_, err := loadCLIConfigWithLogger(configPath, nil)
			require.Error(t, err)
			assert.EqualError(t, err, tt.wantMessage)
		})
	}
}

func TestParseResolveArgs(t *testing.T) {
	baseConfig := ariadne.LoadConfigFromEnv(func(key string) string {
		switch key {
		case "APPLE_MUSIC_STOREFRONT":
			return "de"
		default:
			return ""
		}
	})

	tests := []struct {
		name                  string
		args                  []string
		wantURL               string
		wantStorefront        string
		wantFormat            string
		wantMinStrength       ariadne.MatchStrength
		wantServices          []ariadne.ServiceName
		wantHTTPTimeout       time.Duration
		wantResolutionTimeout time.Duration
		wantVerbose           bool
		wantForceSong         bool
		wantForceAlbum        bool
		wantErrContains       string
	}{
		{
			name:            "uses base config storefront",
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
			wantVerbose:     true,
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
			name:            "service filter aliases",
			args:            []string{"--services=apple-music,yt_music", "https://www.deezer.com/album/12047952"},
			wantURL:         "https://www.deezer.com/album/12047952",
			wantStorefront:  "de",
			wantFormat:      "json",
			wantMinStrength: ariadne.MatchStrengthVeryWeak,
			wantServices:    []ariadne.ServiceName{ariadne.ServiceAppleMusic, ariadne.ServiceYouTubeMusic},
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
			wantErrContains: "usage: ariadne resolve [--log-level=debug] [--song|--album] [--verbose] [--format=json|yaml|csv] [--services=spotify,deezer] [--min-strength=probable] [--apple-music-storefront=us] [--resolution-timeout=20s] <url>",
		},
		{
			name:            "force song",
			args:            []string{"--song", "https://open.spotify.com/track/123"},
			wantURL:         "https://open.spotify.com/track/123",
			wantStorefront:  "de",
			wantFormat:      "json",
			wantMinStrength: ariadne.MatchStrengthVeryWeak,
			wantForceSong:   true,
		},
		{
			name:            "force album",
			args:            []string{"--album", "https://www.deezer.com/album/12047952"},
			wantURL:         "https://www.deezer.com/album/12047952",
			wantStorefront:  "de",
			wantFormat:      "json",
			wantMinStrength: ariadne.MatchStrengthVeryWeak,
			wantForceAlbum:  true,
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
			name:                  "resolution timeout flag",
			args:                  []string{"--resolution-timeout=45s", "https://www.deezer.com/album/12047952"},
			wantURL:               "https://www.deezer.com/album/12047952",
			wantStorefront:        "de",
			wantFormat:            "json",
			wantMinStrength:       ariadne.MatchStrengthVeryWeak,
			wantResolutionTimeout: 45 * time.Second,
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
			resolveConfig, err := parseResolveArgs(tt.args, baseConfig)
			if tt.wantErrContains != "" {
				require.Error(t, err)
				assert.Contains(t, rootError(err).Error(), tt.wantErrContains)
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
			wantResolutionTimeout := tt.wantResolutionTimeout
			if wantResolutionTimeout == 0 {
				wantResolutionTimeout = defaultResolveTimeout
			}
			assert.Equal(t, wantResolutionTimeout, resolveConfig.resolutionTimeout)
			assert.Len(t, resolveConfig.resolverConfig.TargetServices, len(tt.wantServices))
			for i, service := range tt.wantServices {
				assert.Equal(t, service, resolveConfig.resolverConfig.TargetServices[i])
			}
			assert.Equal(t, tt.wantVerbose, resolveConfig.verbose)
			assert.Equal(t, tt.wantForceSong, resolveConfig.forceSong)
			assert.Equal(t, tt.wantForceAlbum, resolveConfig.forceAlbum)
		})
	}
}

var (
	errAllTargetsAlpha = errors.New("alpha boom")
	errAllTargetsBeta  = errors.New("beta boom")
)

func TestAllTargetsFailedErrorExposesEveryFailure(t *testing.T) {
	failures := map[string]error{
		"beta":  errAllTargetsBeta,
		"alpha": errAllTargetsAlpha,
	}

	err := allTargetsFailedError(2, failures)

	require.Error(t, err)
	assert.ErrorIs(t, err, errAllTargetSearchesFailed)
	assert.ErrorIs(t, err, errAllTargetsAlpha)
	assert.ErrorIs(t, err, errAllTargetsBeta)
	assert.Contains(t, err.Error(), "alpha, beta")
}

func TestAllTargetsFailedErrorReturnsNilWhenSomeTargetSucceeded(t *testing.T) {
	err := allTargetsFailedError(2, map[string]error{"alpha": errAllTargetsAlpha})

	assert.NoError(t, err)
}
