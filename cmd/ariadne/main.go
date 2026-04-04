package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	amazonmusicadapter "github.com/xmbshwll/ariadne/internal/adapters/amazonmusic"
	applemusicadapter "github.com/xmbshwll/ariadne/internal/adapters/applemusic"
	bandcampadapter "github.com/xmbshwll/ariadne/internal/adapters/bandcamp"
	deezeradapter "github.com/xmbshwll/ariadne/internal/adapters/deezer"
	soundcloudadapter "github.com/xmbshwll/ariadne/internal/adapters/soundcloud"
	spotifyadapter "github.com/xmbshwll/ariadne/internal/adapters/spotify"
	tidaladapter "github.com/xmbshwll/ariadne/internal/adapters/tidal"
	youtubemusicadapter "github.com/xmbshwll/ariadne/internal/adapters/youtubemusic"
	"github.com/xmbshwll/ariadne/internal/config"
	"github.com/xmbshwll/ariadne/internal/httpx"
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/resolve"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 {
		if err := printUsage(stderr); err != nil {
			return fmt.Errorf("print usage: %w", err)
		}
		return errors.New("missing command")
	}

	switch args[0] {
	case "resolve":
		return runResolve(args[1:], stdout)
	case "help", "--help", "-h":
		return printUsage(stdout)
	default:
		if err := printUsage(stderr); err != nil {
			return fmt.Errorf("print usage: %w", err)
		}
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

var resolverFactory = newResolver

func runResolve(args []string, stdout io.Writer) error {
	appConfig := config.Load()
	resolveConfig, err := parseResolveArgs(args, appConfig)
	if err != nil {
		return err
	}

	resolver := resolverFactory(appConfig, resolveConfig.appleMusicStorefront)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resolution, err := resolver.ResolveAlbum(ctx, resolveConfig.inputURL)
	if err != nil {
		return fmt.Errorf("resolve album: %w", err)
	}

	output := newCLIResolution(*resolution)
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		return fmt.Errorf("encode resolution json: %w", err)
	}
	return nil
}

func newResolver(appConfig config.Config, appleMusicStorefront string) *resolve.Resolver {
	client := httpx.NewClient()
	amazonMusic := amazonmusicadapter.New(client)
	appleMusic := applemusicadapter.New(
		client,
		applemusicadapter.WithDefaultStorefront(appleMusicStorefront),
		applemusicadapter.WithDeveloperTokenAuth(
			appConfig.AppleMusic.KeyID,
			appConfig.AppleMusic.TeamID,
			appConfig.AppleMusic.PrivateKeyPath,
		),
	)
	bandcamp := bandcampadapter.New(client)
	deezer := deezeradapter.New(client)
	soundCloud := soundcloudadapter.New(client)
	youTubeMusic := youtubemusicadapter.New(client)
	spotify := spotifyadapter.New(client, spotifyadapter.WithCredentials(appConfig.Spotify.ClientID, appConfig.Spotify.ClientSecret))
	tidal := tidaladapter.New(client, tidaladapter.WithCredentials(appConfig.TIDAL.ClientID, appConfig.TIDAL.ClientSecret))

	sources := []resolve.SourceAdapter{appleMusic, deezer, spotify, tidal, soundCloud, youTubeMusic, amazonMusic, bandcamp}
	targets := []resolve.TargetAdapter{appleMusic, bandcamp, deezer, soundCloud, youTubeMusic}
	if appConfig.Spotify.Enabled() {
		targets = append(targets, spotify)
	}
	if appConfig.TIDAL.Enabled() {
		targets = append(targets, tidal)
	}

	return resolve.New(sources, targets)
}

func printUsage(w io.Writer) error {
	_, err := io.WriteString(w, "ariadne\n\nUsage:\n  ariadne resolve [--apple-music-storefront=us] <album-url>\n\nNotes:\n  - current CLI wiring includes Deezer, Bandcamp, SoundCloud, YouTube Music, and Apple Music as source/target adapters\n  - Spotify source fetch works without credentials, but Spotify target search is enabled only when SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET are set\n  - Apple Music source fetch and metadata search work without extra credentials; UPC and ISRC target search are enabled when APPLE_MUSIC_KEY_ID, APPLE_MUSIC_TEAM_ID, and APPLE_MUSIC_PRIVATE_KEY_PATH are set\n  - SoundCloud source fetch and metadata search use public page hydration plus public-facing api-v2 search and remain experimental\n  - YouTube Music source fetch and metadata search use public HTML extraction with a browser-like user-agent and remain experimental\n  - TIDAL source fetch and target search both require TIDAL_CLIENT_ID and TIDAL_CLIENT_SECRET; there is no public runtime fallback\n  - Amazon Music album URLs are recognized, but runtime resolving remains deferred because no viable public metadata fetch path exists yet\n  - Deezer, Spotify, Apple Music, Bandcamp, SoundCloud, and YouTube Music album-like URLs work without extra credentials; TIDAL album URLs require TIDAL credentials\n  - Apple Music target search defaults to APPLE_MUSIC_STOREFRONT or us\n")
	if err != nil {
		return fmt.Errorf("write usage: %w", err)
	}
	return nil
}

type resolveConfig struct {
	inputURL             string
	appleMusicStorefront string
}

func parseResolveArgs(args []string, appConfig config.Config) (resolveConfig, error) {
	config := resolveConfig{
		appleMusicStorefront: appConfig.AppleMusic.Storefront,
	}

	fs := flag.NewFlagSet("resolve", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&config.appleMusicStorefront, "apple-music-storefront", config.appleMusicStorefront, "preferred Apple Music storefront")
	if err := fs.Parse(args); err != nil {
		return resolveConfig{}, errors.New("usage: ariadne resolve [--apple-music-storefront=us] <album-url>")
	}
	remaining := fs.Args()
	if len(remaining) != 1 {
		return resolveConfig{}, errors.New("usage: ariadne resolve [--apple-music-storefront=us] <album-url>")
	}

	config.inputURL = remaining[0]
	config.appleMusicStorefront = strings.ToLower(strings.TrimSpace(config.appleMusicStorefront))
	if config.appleMusicStorefront == "" {
		config.appleMusicStorefront = "us"
	}
	return config, nil
}

type cliResolution struct {
	InputURL string                    `json:"input_url"`
	Source   cliAlbum                  `json:"source"`
	Links    map[string]cliMatchResult `json:"links,omitempty"`
}

type cliAlbum struct {
	Service      string   `json:"service"`
	ID           string   `json:"id"`
	URL          string   `json:"url"`
	RegionHint   string   `json:"region_hint,omitempty"`
	Title        string   `json:"title"`
	Artists      []string `json:"artists"`
	ReleaseDate  string   `json:"release_date,omitempty"`
	Label        string   `json:"label,omitempty"`
	UPC          string   `json:"upc,omitempty"`
	TrackCount   int      `json:"track_count,omitempty"`
	ArtworkURL   string   `json:"artwork_url,omitempty"`
	EditionHints []string `json:"edition_hints,omitempty"`
}

type cliMatchResult struct {
	Found      bool       `json:"found"`
	Summary    string     `json:"summary"`
	Best       *cliMatch  `json:"best,omitempty"`
	Alternates []cliMatch `json:"alternates,omitempty"`
}

type cliMatch struct {
	URL         string   `json:"url"`
	Score       int      `json:"score"`
	Reasons     []string `json:"reasons,omitempty"`
	AlbumID     string   `json:"album_id,omitempty"`
	RegionHint  string   `json:"region_hint,omitempty"`
	Title       string   `json:"title,omitempty"`
	Artists     []string `json:"artists,omitempty"`
	ReleaseDate string   `json:"release_date,omitempty"`
	UPC         string   `json:"upc,omitempty"`
}

func newCLIResolution(resolution resolve.Resolution) cliResolution {
	links := make(map[string]cliMatchResult, len(resolution.Matches))
	for service, match := range resolution.Matches {
		links[string(service)] = newCLIMatchResult(match)
	}

	return cliResolution{
		InputURL: resolution.InputURL,
		Source:   newCLIAlbum(resolution.Source),
		Links:    links,
	}
}

func newCLIAlbum(album model.CanonicalAlbum) cliAlbum {
	return cliAlbum{
		Service:      string(album.Service),
		ID:           album.SourceID,
		URL:          album.SourceURL,
		RegionHint:   album.RegionHint,
		Title:        album.Title,
		Artists:      append([]string(nil), album.Artists...),
		ReleaseDate:  album.ReleaseDate,
		Label:        album.Label,
		UPC:          album.UPC,
		TrackCount:   album.TrackCount,
		ArtworkURL:   album.ArtworkURL,
		EditionHints: append([]string(nil), album.EditionHints...),
	}
}

func newCLIMatchResult(result resolve.MatchResult) cliMatchResult {
	output := cliMatchResult{
		Found:      result.Best != nil,
		Summary:    "not_found",
		Alternates: make([]cliMatch, 0, len(result.Alternates)),
	}
	if result.Best != nil {
		best := newCLIMatch(*result.Best)
		output.Best = &best
		output.Summary = scoreSummary(result.Best.Score)
	}
	for _, alternate := range result.Alternates {
		output.Alternates = append(output.Alternates, newCLIMatch(alternate))
	}
	return output
}

func scoreSummary(score int) string {
	switch {
	case score >= 100:
		return "strong"
	case score >= 70:
		return "probable"
	case score >= 50:
		return "weak"
	default:
		return "very_weak"
	}
}

func newCLIMatch(match resolve.ScoredMatch) cliMatch {
	return cliMatch{
		URL:         match.URL,
		Score:       match.Score,
		Reasons:     append([]string(nil), match.Reasons...),
		AlbumID:     match.Candidate.CandidateID,
		RegionHint:  match.Candidate.RegionHint,
		Title:       match.Candidate.Title,
		Artists:     append([]string(nil), match.Candidate.Artists...),
		ReleaseDate: match.Candidate.ReleaseDate,
		UPC:         match.Candidate.UPC,
	}
}
