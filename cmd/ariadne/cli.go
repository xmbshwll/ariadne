package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/xmbshwll/ariadne"
)

const (
	defaultConfigPath = ".env"
	outputFormatJSON  = "json"
	outputFormatYAML  = "yaml"
	outputFormatCSV   = "csv"
	resolveUsage      = "usage: ariadne resolve [--verbose] [--format=json|yaml|csv] [--services=spotify,deezer] [--min-strength=probable] [--apple-music-storefront=us] <album-url>"
)

const resolveHelpText = `Resolve album URLs across music services.

Usage:
  ariadne resolve [--verbose] [--format=json|yaml|csv] [--services=spotify,deezer] [--min-strength=probable] [--apple-music-storefront=us] <album-url>

Positional parameter:
  <album-url>
    Required.
    Values: a supported album URL from Apple Music, Deezer, Spotify, TIDAL,
    SoundCloud, YouTube Music, Bandcamp, or Amazon Music.
    Amazon Music URLs are recognized for parsing, but runtime resolution remains deferred.

Flags:
  --config
    Values: empty string to disable file loading, or a path to a config file.
    Supported file styles: .env-style key=value files, plus Viper-supported structured files such as yaml, yml, json, or toml.
    Default: %s
    Behavior: config file values are loaded first, environment variables override them, and explicit CLI flags override both.

  --verbose, -v
    Values: true, false.
    Default: false.
    false prints compact service-link output only.
    true includes source metadata, per-service summaries, scores, reasons, and alternates.

  --format
    Values:
      json  - indented JSON; best default for scripts and APIs.
      yaml  - YAML rendering of the same payload.
      csv   - compact or verbose CSV depending on --verbose.
    Default: json.

  --services
    Values: comma-separated list drawn from appleMusic, bandcamp, deezer, soundcloud, spotify, tidal, youtubeMusic, ytmusic.
    ytmusic is an alias for youtubeMusic.
    Use this to limit which target services are searched.
    Caveats:
      spotify requires SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET.
      tidal requires TIDAL_CLIENT_ID and TIDAL_CLIENT_SECRET.
      amazonMusic is not a valid target service.

  --min-strength
    Values:
      very_weak - include every retained match.
      weak      - exclude very weak matches.
      probable  - show only stronger likely matches.
      strong    - show only highest-confidence matches.
    Default: very_weak.

  --apple-music-storefront
    Values: an Apple Music storefront country code in ISO 3166-1 alpha-2 form, for example us, gb, de, fr, jp, ca, or au.
    Default: %s.
    Used for Apple Music lookups and searches when the source URL does not already imply a storefront.

Notes:
  - Spotify target search is enabled only when SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET are set.
  - Apple Music UPC and ISRC target search are enabled when APPLE_MUSIC_KEY_ID, APPLE_MUSIC_TEAM_ID, and APPLE_MUSIC_PRIVATE_KEY_PATH are set.
  - TIDAL source fetch and target search require TIDAL_CLIENT_ID and TIDAL_CLIENT_SECRET.`

var (
	resolverFactory = ariadne.New
	valueNormalizer = strings.NewReplacer("-", "", "_", "")
)

var (
	errExecuteCLI               = errors.New("execute cli")
	errRenderResolveHelp        = errors.New("render resolve help")
	errMissingCommand           = errors.New("missing command")
	errUnknownCommand           = errors.New("unknown command")
	errResolveUsage             = errors.New(resolveUsage)
	errUnsupportedFormat        = errors.New("unsupported format")
	errNoTargetServicesSelected = errors.New("no target services selected")
	errAmazonMusicTargetService = errors.New("amazonMusic is not available as a target service")
	errUnsupportedTargetService = errors.New("unsupported target service")
	errSpotifyTargetCredentials = errors.New("spotify target search requires SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET")
	errTIDALTargetCredentials   = errors.New("tidal target search requires TIDAL_CLIENT_ID and TIDAL_CLIENT_SECRET")
	errUnsupportedMinStrength   = errors.New("unsupported min-strength")
)

var (
	requestedServiceByName = map[string]ariadne.ServiceName{
		"applemusic":   ariadne.ServiceAppleMusic,
		"bandcamp":     ariadne.ServiceBandcamp,
		"deezer":       ariadne.ServiceDeezer,
		"soundcloud":   ariadne.ServiceSoundCloud,
		"spotify":      ariadne.ServiceSpotify,
		"tidal":        ariadne.ServiceTIDAL,
		"youtubemusic": ariadne.ServiceYouTubeMusic,
		"ytmusic":      ariadne.ServiceYouTubeMusic,
	}
	matchStrengthByName = map[string]ariadne.MatchStrength{
		"veryweak":  ariadne.MatchStrengthVeryWeak,
		"very_weak": ariadne.MatchStrengthVeryWeak,
		"weak":      ariadne.MatchStrengthWeak,
		"probable":  ariadne.MatchStrengthProbable,
		"strong":    ariadne.MatchStrengthStrong,
	}
)

type resolveConfig struct {
	inputURL          string
	verbose           bool
	format            string
	requestedServices string
	minStrengthName   string
	minStrength       ariadne.MatchStrength
	resolverConfig    ariadne.Config
}

func run(args []string, stdout io.Writer, stderr io.Writer) error {
	configPath := configPathFromArgs(args)
	baseConfig, err := loadCLIConfig(configPath)
	if err != nil {
		return err
	}

	if len(args) == 0 {
		if err := renderResolveHelp(stderr, baseConfig, configPath); err != nil {
			return fmt.Errorf("print usage: %w", err)
		}
		return errMissingCommand
	}
	if isHelpArg(args[0]) {
		return renderResolveHelp(stdout, baseConfig, configPath)
	}

	root := newRootCmd(stdout, stderr, baseConfig, configPath)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		if isUnknownCommandError(err) {
			if helpErr := renderResolveHelp(stderr, baseConfig, configPath); helpErr != nil {
				return fmt.Errorf("print usage: %w", helpErr)
			}
			return fmt.Errorf("%w: %s", errUnknownCommand, args[0])
		}
		return fmt.Errorf("%w: %w", errExecuteCLI, err)
	}
	return nil
}

func isHelpArg(arg string) bool {
	return arg == "help" || arg == "--help" || arg == "-h"
}

func isUnknownCommandError(err error) bool {
	return strings.HasPrefix(err.Error(), "unknown command ")
}

func newRootCmd(stdout io.Writer, stderr io.Writer, baseConfig ariadne.Config, configPath string) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "ariadne",
		Short:         "Resolve album URLs across music services.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.PersistentFlags().String("config", configPath, "configuration source (values: empty string to disable file loading, or a path to an .env, yaml, yml, json, or toml file)")
	cmd.AddCommand(newResolveCmd(baseConfig, configPath))
	return cmd
}

func newResolveCmd(baseConfig ariadne.Config, configPath string) *cobra.Command {
	config := defaultResolveConfig(baseConfig)

	cmd := &cobra.Command{
		Use:   "resolve [--verbose] [--format=json|yaml|csv] [--services=spotify,deezer] [--min-strength=probable] [--apple-music-storefront=us] <album-url>",
		Short: "Resolve one album URL into likely equivalents on other services.",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errResolveUsage
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			config.inputURL = args[0]
			normalized, err := normalizeResolveConfig(config)
			if err != nil {
				return err
			}
			return executeResolve(normalized, cmd.OutOrStdout())
		},
	}

	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		_, _ = io.WriteString(cmd.OutOrStdout(), resolveHelpTextFor(baseConfig, configPath))
	})

	bindResolveFlags(cmd.Flags(), &config)
	return cmd
}

func renderResolveHelp(w io.Writer, baseConfig ariadne.Config, configPath string) error {
	if _, err := io.WriteString(w, resolveHelpTextFor(baseConfig, configPath)); err != nil {
		return fmt.Errorf("%w: %w", errRenderResolveHelp, err)
	}
	return nil
}

func defaultResolveConfig(baseConfig ariadne.Config) resolveConfig {
	return resolveConfig{
		format:          outputFormatJSON,
		minStrengthName: string(ariadne.MatchStrengthVeryWeak),
		minStrength:     ariadne.MatchStrengthVeryWeak,
		resolverConfig:  baseConfig,
	}
}

func resolveHelpTextFor(baseConfig ariadne.Config, configPath string) string {
	if configPath == "" {
		configPath = `"" (disable file loading)`
	}

	storefrontDefault := "APPLE_MUSIC_STOREFRONT or us"
	if baseConfig.AppleMusicStorefront != "" {
		storefrontDefault = baseConfig.AppleMusicStorefront
	}

	return fmt.Sprintf(resolveHelpText, configPath, storefrontDefault)
}

func configPathFromArgs(args []string) string {
	for i, arg := range args {
		switch {
		case arg == "--config":
			if i+1 >= len(args) {
				return defaultConfigPath
			}
			return args[i+1]
		case strings.HasPrefix(arg, "--config="):
			value, _ := strings.CutPrefix(arg, "--config=")
			return value
		}
	}
	return defaultConfigPath
}

func loadCLIConfig(configPath string) (ariadne.Config, error) {
	cfg := ariadne.DefaultConfig()
	v := viper.New()
	v.AutomaticEnv()

	if strings.TrimSpace(configPath) != "" {
		v.SetConfigFile(configPath)
		if looksLikeEnvFile(configPath) {
			v.SetConfigType("env")
		}
		if err := v.ReadInConfig(); err != nil {
			var notFound viper.ConfigFileNotFoundError
			if !errors.As(err, &notFound) && !errors.Is(err, os.ErrNotExist) {
				return ariadne.Config{}, fmt.Errorf("load config %q: %w", configPath, err)
			}
		}
	}

	trimmedValue := func(key string) string {
		return strings.TrimSpace(v.GetString(key))
	}

	cfg.Spotify.ClientID = trimmedValue("SPOTIFY_CLIENT_ID")
	cfg.Spotify.ClientSecret = trimmedValue("SPOTIFY_CLIENT_SECRET")
	cfg.AppleMusic.KeyID = trimmedValue("APPLE_MUSIC_KEY_ID")
	cfg.AppleMusic.TeamID = trimmedValue("APPLE_MUSIC_TEAM_ID")
	cfg.AppleMusic.PrivateKeyPath = trimmedValue("APPLE_MUSIC_PRIVATE_KEY_PATH")
	cfg.TIDAL.ClientID = trimmedValue("TIDAL_CLIENT_ID")
	cfg.TIDAL.ClientSecret = trimmedValue("TIDAL_CLIENT_SECRET")

	storefront := strings.ToLower(trimmedValue("APPLE_MUSIC_STOREFRONT"))
	if storefront != "" {
		cfg.AppleMusicStorefront = storefront
	}

	return cfg, nil
}

func looksLikeEnvFile(path string) bool {
	base := filepath.Base(path)
	return strings.HasPrefix(base, ".env") || strings.EqualFold(filepath.Ext(base), ".env")
}

func bindResolveFlags(fs *pflag.FlagSet, config *resolveConfig) {
	fs.StringVar(&config.resolverConfig.AppleMusicStorefront, "apple-music-storefront", config.resolverConfig.AppleMusicStorefront, "preferred Apple Music storefront (values: ISO 3166-1 alpha-2 code such as us, gb, de, fr, jp, ca, au; used when the source URL has no storefront)")
	fs.BoolVarP(&config.verbose, "verbose", "v", false, "print full resolution details (values: true or false; false emits compact links, true emits metadata, scores, reasons, and alternates)")
	fs.StringVar(&config.format, "format", config.format, "output format (values: json for structured output, yaml for YAML, csv for spreadsheet-friendly export)")
	fs.StringVar(&config.requestedServices, "services", "", "comma-separated target services (values: appleMusic, bandcamp, deezer, soundcloud, spotify, tidal, youtubeMusic, ytmusic; ytmusic aliases youtubeMusic)")
	fs.StringVar(&config.minStrengthName, "min-strength", config.minStrengthName, "minimum match strength (values: very_weak, weak, probable, strong; filters weaker results out of the final output)")
}

func parseResolveArgs(args []string, baseConfig ariadne.Config) (resolveConfig, error) {
	config := defaultResolveConfig(baseConfig)
	fs := pflag.NewFlagSet("resolve", pflag.ContinueOnError)
	fs.SetOutput(io.Discard)
	bindResolveFlags(fs, &config)
	if err := fs.Parse(args); err != nil {
		return resolveConfig{}, errResolveUsage
	}
	remaining := fs.Args()
	if len(remaining) != 1 {
		return resolveConfig{}, errResolveUsage
	}
	config.inputURL = remaining[0]
	return normalizeResolveConfig(config)
}

func normalizeResolveConfig(config resolveConfig) (resolveConfig, error) {
	format, err := normalizeOutputFormat(config.format)
	if err != nil {
		return resolveConfig{}, err
	}
	config.format = format

	services, err := parseRequestedServices(config.requestedServices, config.resolverConfig)
	if err != nil {
		return resolveConfig{}, err
	}
	config.resolverConfig.TargetServices = services

	strength, err := parseMatchStrength(config.minStrengthName)
	if err != nil {
		return resolveConfig{}, err
	}
	config.minStrength = strength
	return config, nil
}

func runResolve(args []string, stdout io.Writer) error {
	baseConfig, err := loadCLIConfig(configPathFromArgs(args))
	if err != nil {
		return err
	}
	config, err := parseResolveArgs(args, baseConfig)
	if err != nil {
		return err
	}
	return executeResolve(config, stdout)
}

func executeResolve(config resolveConfig, stdout io.Writer) error {
	resolver := resolverFactory(config.resolverConfig)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resolution, err := resolver.ResolveAlbum(ctx, config.inputURL)
	if err != nil {
		return fmt.Errorf("resolve album: %w", err)
	}

	if err := writeCLIOutput(stdout, *resolution, config); err != nil {
		return err
	}
	return nil
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

func newCLIOutput(resolution ariadne.Resolution, cfg resolveConfig) any {
	if cfg.verbose {
		return newCLIResolution(resolution)
	}
	return newCLILinks(resolution)
}

func writeCLIOutput(w io.Writer, resolution ariadne.Resolution, cfg resolveConfig) error {
	resolution = filterResolutionByStrength(resolution, cfg.minStrength)
	output := newCLIOutput(resolution, cfg)

	switch cfg.format {
	case outputFormatJSON:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(output); err != nil {
			return fmt.Errorf("encode resolution json: %w", err)
		}
		return nil
	case outputFormatYAML:
		data, err := yaml.Marshal(output)
		if err != nil {
			return fmt.Errorf("encode resolution yaml: %w", err)
		}
		if _, err := w.Write(data); err != nil {
			return fmt.Errorf("write resolution yaml: %w", err)
		}
		return nil
	case outputFormatCSV:
		if cfg.verbose {
			return writeVerboseCSV(w, resolution)
		}
		return writeCompactCSV(w, resolution)
	default:
		return fmt.Errorf("%w: %q", errUnsupportedFormat, cfg.format)
	}
}

func newCLIResolution(resolution ariadne.Resolution) cliResolution {
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

func newCLILinks(resolution ariadne.Resolution) map[string]string {
	links := map[string]string{}
	if resolution.Source.Service != "" && resolution.Source.SourceURL != "" {
		links[string(resolution.Source.Service)] = resolution.Source.SourceURL
	}
	for service, match := range resolution.Matches {
		if match.Best == nil || match.Best.URL == "" {
			continue
		}
		if _, exists := links[string(service)]; exists {
			continue
		}
		links[string(service)] = match.Best.URL
	}
	return links
}

func writeCompactCSV(w io.Writer, resolution ariadne.Resolution) error {
	writer := csv.NewWriter(w)
	if err := writer.Write([]string{"service", "url"}); err != nil {
		return fmt.Errorf("write csv header: %w", err)
	}
	links := newCLILinks(resolution)
	services := sortedKeys(links)
	for _, service := range services {
		if err := writer.Write([]string{service, links[service]}); err != nil {
			return fmt.Errorf("write csv row: %w", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush csv: %w", err)
	}
	return nil
}

func writeVerboseCSV(w io.Writer, resolution ariadne.Resolution) error {
	writer := csv.NewWriter(w)
	headers := []string{"input_url", "service", "kind", "url", "found", "summary", "score", "album_id", "region_hint", "title", "artists", "release_date", "upc", "reasons"}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("write csv header: %w", err)
	}
	for _, row := range newVerboseCSVRows(resolution) {
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("write csv row: %w", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush csv: %w", err)
	}
	return nil
}

func newVerboseCSVRows(resolution ariadne.Resolution) [][]string {
	rows := [][]string{{
		resolution.InputURL,
		string(resolution.Source.Service),
		"source",
		resolution.Source.SourceURL,
		"true",
		"source",
		"",
		resolution.Source.SourceID,
		resolution.Source.RegionHint,
		resolution.Source.Title,
		strings.Join(resolution.Source.Artists, " | "),
		resolution.Source.ReleaseDate,
		resolution.Source.UPC,
		"",
	}}

	services := make([]string, 0, len(resolution.Matches))
	for service := range resolution.Matches {
		services = append(services, string(service))
	}
	sort.Strings(services)
	for _, service := range services {
		result := resolution.Matches[ariadne.ServiceName(service)]
		if result.Best == nil {
			rows = append(rows, []string{
				resolution.InputURL,
				service,
				"best",
				"",
				"false",
				"not_found",
				"",
				"",
				"",
				"",
				"",
				"",
				"",
				"",
			})
			continue
		}
		rows = append(rows, newCSVMatchRow(resolution.InputURL, service, "best", true, scoreSummary(result.Best.Score), *result.Best))
		for _, alternate := range result.Alternates {
			rows = append(rows, newCSVMatchRow(resolution.InputURL, service, "alternate", true, scoreSummary(alternate.Score), alternate))
		}
	}
	return rows
}

func newCSVMatchRow(inputURL, service, kind string, found bool, summary string, match ariadne.ScoredMatch) []string {
	return []string{
		inputURL,
		service,
		kind,
		match.URL,
		strconv.FormatBool(found),
		summary,
		strconv.Itoa(match.Score),
		match.Candidate.CandidateID,
		match.Candidate.RegionHint,
		match.Candidate.Title,
		strings.Join(match.Candidate.Artists, " | "),
		match.Candidate.ReleaseDate,
		match.Candidate.UPC,
		strings.Join(match.Reasons, " | "),
	}
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func parseRequestedServices(raw string, appConfig ariadne.Config) ([]ariadne.ServiceName, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	services := make([]ariadne.ServiceName, 0)
	seen := map[ariadne.ServiceName]struct{}{}
	for part := range strings.SplitSeq(raw, ",") {
		service, err := normalizeRequestedService(part)
		if err != nil {
			return nil, err
		}
		if err := validateRequestedService(service, appConfig); err != nil {
			return nil, err
		}
		if _, ok := seen[service]; ok {
			continue
		}
		seen[service] = struct{}{}
		services = append(services, service)
	}
	if len(services) == 0 {
		return nil, errNoTargetServicesSelected
	}
	return services, nil
}

func normalizeRequestedService(raw string) (ariadne.ServiceName, error) {
	normalized := normalizeLookupKey(raw)
	if normalized == "" {
		return "", errNoTargetServicesSelected
	}
	if normalized == "amazonmusic" || normalized == "amazon" {
		return "", errAmazonMusicTargetService
	}
	service, ok := requestedServiceByName[normalized]
	if !ok {
		return "", fmt.Errorf("%w %q (expected one of appleMusic, bandcamp, deezer, soundcloud, spotify, tidal, youtubeMusic)", errUnsupportedTargetService, raw)
	}
	return service, nil
}

func validateRequestedService(service ariadne.ServiceName, appConfig ariadne.Config) error {
	switch service {
	case ariadne.ServiceSpotify:
		if !appConfig.SpotifyEnabled() {
			return errSpotifyTargetCredentials
		}
	case ariadne.ServiceTIDAL:
		if !appConfig.TIDALEnabled() {
			return errTIDALTargetCredentials
		}
	}
	return nil
}

func normalizeOutputFormat(raw string) (string, error) {
	format := strings.ToLower(strings.TrimSpace(raw))
	if format == "" {
		return outputFormatJSON, nil
	}
	if format != outputFormatJSON && format != outputFormatYAML && format != outputFormatCSV {
		return "", fmt.Errorf("%w %q (expected json, yaml, or csv)", errUnsupportedFormat, format)
	}
	return format, nil
}

func parseMatchStrength(raw string) (ariadne.MatchStrength, error) {
	normalized := normalizeLookupKey(raw)
	if normalized == "" {
		return ariadne.MatchStrengthVeryWeak, nil
	}
	strength, ok := matchStrengthByName[normalized]
	if !ok {
		return "", fmt.Errorf("%w %q (expected very_weak, weak, probable, or strong)", errUnsupportedMinStrength, raw)
	}
	return strength, nil
}

func normalizeLookupKey(raw string) string {
	return valueNormalizer.Replace(strings.ToLower(strings.TrimSpace(raw)))
}

func filterResolutionByStrength(resolution ariadne.Resolution, minStrength ariadne.MatchStrength) ariadne.Resolution {
	if minStrength == ariadne.MatchStrengthVeryWeak {
		return resolution
	}
	filtered := resolution
	filtered.Matches = make(map[ariadne.ServiceName]ariadne.MatchResult, len(resolution.Matches))
	for service, match := range resolution.Matches {
		if match.Best == nil {
			continue
		}
		if !meetsMinimumStrength(match.Best.Score, minStrength) {
			continue
		}
		filtered.Matches[service] = match
	}
	return filtered
}

func meetsMinimumStrength(score int, minStrength ariadne.MatchStrength) bool {
	return matchStrengthRank(ariadne.MatchStrengthForScore(score)) >= matchStrengthRank(minStrength)
}

func matchStrengthRank(strength ariadne.MatchStrength) int {
	switch strength {
	case ariadne.MatchStrengthStrong:
		return 3
	case ariadne.MatchStrengthProbable:
		return 2
	case ariadne.MatchStrengthWeak:
		return 1
	default:
		return 0
	}
}

func newCLIAlbum(album ariadne.CanonicalAlbum) cliAlbum {
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

func newCLIMatchResult(result ariadne.MatchResult) cliMatchResult {
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
	return string(ariadne.MatchStrengthForScore(score))
}

func newCLIMatch(match ariadne.ScoredMatch) cliMatch {
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
