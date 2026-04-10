package main

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/xmbshwll/ariadne"
)

const (
	defaultConfigPath = ".env"
	outputFormatJSON  = "json"
	outputFormatYAML  = "yaml"
	outputFormatCSV   = "csv"
	resolveUsage      = "usage: ariadne resolve [--song|--album] [--verbose] [--format=json|yaml|csv] [--services=spotify,deezer] [--min-strength=probable] [--apple-music-storefront=us] [--resolution-timeout=20s] <url>"
)

const resolveHelpText = `Resolve a supported music URL across music services.

Usage:
  ariadne resolve [--song|--album] [--verbose] [--format=json|yaml|csv] [--services=spotify,deezer] [--min-strength=probable] [--apple-music-storefront=us] [--resolution-timeout=20s] <url>

Positional parameter:
  <url>
    Required.
    Values: a supported album URL from Apple Music, Deezer, Spotify, TIDAL,
    SoundCloud, YouTube Music, Bandcamp, or Amazon Music, or a supported song
    URL from Apple Music, Bandcamp, Deezer, SoundCloud, Spotify, or TIDAL.
    Behavior: when neither --song nor --album is set, Ariadne asks the library
    to auto-detect the resource type from the URL.
    Amazon Music URLs are recognized for parsing, but runtime resolution remains deferred.

Flags:
  --config
    Values: empty string to disable file loading, or a path to a config file.
    Supported file styles: .env-style key=value files, plus Viper-supported structured files such as yaml, yml, json, or toml.
    Default: %s
    Behavior: config file values are loaded first, environment variables override them, and explicit CLI flags override both.

  --song
    Forces song resolution for the provided URL.
    Mutually exclusive with --album.

  --album
    Forces album resolution for the provided URL.
    Mutually exclusive with --song.

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

  --http-timeout
    Values: a Go duration such as 5s, 15s, 30s, or 1m.
    Default: %s.
    Sets the per-request timeout on Ariadne's default HTTP client for service API and page requests.

  --resolution-timeout
    Values: a Go duration such as 20s, 30s, 1m, or 2m.
    Default: 20s.
    Sets the overall timeout for one resolve command across parsing, source fetch, and target searches.

Notes:
  - Spotify target search is enabled only when SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET are set.
  - Apple Music UPC and ISRC target search are enabled when APPLE_MUSIC_KEY_ID, APPLE_MUSIC_TEAM_ID, and APPLE_MUSIC_PRIVATE_KEY_PATH are set.
  - TIDAL source fetch and target search require TIDAL_CLIENT_ID and TIDAL_CLIENT_SECRET.
  - Song resolution currently supports Apple Music, Bandcamp, Deezer, SoundCloud, Spotify, and TIDAL.`

var (
	resolverFactory = ariadne.New
	valueNormalizer = strings.NewReplacer("-", "", "_", "")
)

var (
	errRenderResolveHelp         = errors.New("render resolve help")
	errMissingCommand            = errors.New("missing command")
	errUnknownCommand            = errors.New("unknown command")
	errResolveUsage              = errors.New(resolveUsage)
	errConflictingEntityModeFlag = errors.New("--song and --album are mutually exclusive")
	errUnsupportedFormat         = errors.New("unsupported format")
	errNoTargetServicesSelected  = errors.New("no target services selected")
	errAmazonMusicTargetService  = errors.New("amazonMusic is not available as a target service")
	errUnsupportedTargetService  = errors.New("unsupported target service")
	errUnsupportedSongService    = errors.New("target service is not available for song resolution")
	errSpotifyTargetCredentials  = errors.New("spotify target search requires SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET")
	errTIDALTargetCredentials    = errors.New("tidal target search requires TIDAL_CLIENT_ID and TIDAL_CLIENT_SECRET")
	errUnsupportedMinStrength    = errors.New("unsupported min-strength")
	errEmptyResolution           = errors.New("empty resolution")
	errUnsupportedResolveMode    = errors.New("unsupported resolve mode")
)

func run(args []string, stdout io.Writer, stderr io.Writer) error {
	configPath := configPathFromArgs(args)
	helpConfig := ariadne.DefaultConfig()
	commandArgs := argsWithoutConfigFlag(args)

	if len(commandArgs) == 0 {
		if err := renderRootHelp(stderr, helpConfig, configPath); err != nil {
			return fmt.Errorf("print usage: %w", err)
		}
		return errMissingCommand
	}
	if isHelpArg(commandArgs[0]) {
		if len(commandArgs) == 1 {
			return renderRootHelp(stdout, helpConfig, configPath)
		}
		return executeRootCommand(stdout, stderr, helpConfig, configPath, args)
	}
	if containsHelpArg(commandArgs[1:]) {
		return executeRootCommand(stdout, stderr, helpConfig, configPath, args)
	}

	baseConfig, err := loadCLIConfig(configPath)
	if err != nil {
		return err
	}

	unknownCommand := firstCommandArg(commandArgs, args)

	if err := executeRootCommand(stdout, stderr, baseConfig, configPath, args); err != nil {
		if isUnknownCommandError(err) {
			if helpErr := renderRootHelp(stderr, baseConfig, configPath); helpErr != nil {
				return fmt.Errorf("print usage: %w", helpErr)
			}
			return fmt.Errorf("%w: %s", errUnknownCommand, unknownCommand)
		}
		return err
	}
	return nil
}

func argsWithoutConfigFlag(args []string) []string {
	filtered := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--config":
			if i+1 < len(args) {
				value := args[i+1]
				if value == "" || !strings.HasPrefix(value, "-") {
					i++
				}
			}
			continue
		case strings.HasPrefix(arg, "--config="):
			continue
		default:
			filtered = append(filtered, arg)
		}
	}
	return filtered
}

func firstCommandArg(commandArgs []string, args []string) string {
	if len(commandArgs) > 0 {
		return commandArgs[0]
	}
	if len(args) > 0 {
		return args[0]
	}
	return "<unknown>"
}

func containsHelpArg(args []string) bool {
	return slices.ContainsFunc(args, isHelpArg)
}

func isHelpArg(arg string) bool {
	return arg == "help" || arg == "--help" || arg == "-h"
}

func isUnknownCommandError(err error) bool {
	return strings.Contains(err.Error(), "unknown command ")
}

func executeRootCommand(stdout io.Writer, stderr io.Writer, baseConfig ariadne.Config, configPath string, args []string) error {
	root := newRootCmd(stdout, stderr, baseConfig, configPath)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		return fmt.Errorf("execute root command: %w", err)
	}
	return nil
}

func newRootCmd(stdout io.Writer, stderr io.Writer, baseConfig ariadne.Config, configPath string) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "ariadne",
		Short:         "Resolve music URLs across services.",
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
		Use:   "resolve [--song|--album] [--verbose] [--format=json|yaml|csv] [--services=spotify,deezer] [--min-strength=probable] [--apple-music-storefront=us] [--resolution-timeout=20s] <url>",
		Short: "Resolve one music URL into likely equivalents on other services.",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errResolveUsage
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			config.inputURL = args[0]
			normalized, err := normalizeAndValidateResolveConfig(config)
			if err != nil {
				return err
			}
			return executeResolve(normalized, cmd.OutOrStdout(), resolveModeFromConfig(normalized))
		},
	}

	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		_, _ = io.WriteString(cmd.OutOrStdout(), resolveHelpTextFor(baseConfig, configPath))
	})

	bindResolveFlags(cmd.Flags(), &config)
	return cmd
}

func renderRootHelp(w io.Writer, baseConfig ariadne.Config, configPath string) error {
	if _, err := io.WriteString(w, rootHelpTextFor(baseConfig, configPath)); err != nil {
		return fmt.Errorf("%w: %w", errRenderResolveHelp, err)
	}
	return nil
}

func resolveHelpTextFor(baseConfig ariadne.Config, configPath string) string {
	if configPath == "" {
		configPath = `"" (disable file loading)`
	}

	storefrontDefault := "APPLE_MUSIC_STOREFRONT or us"
	if baseConfig.AppleMusicStorefront != "" {
		storefrontDefault = baseConfig.AppleMusicStorefront
	}

	return fmt.Sprintf(resolveHelpText, configPath, storefrontDefault, baseConfig.HTTPTimeout)
}

func rootHelpTextFor(baseConfig ariadne.Config, configPath string) string {
	return strings.Join([]string{
		"Usage:",
		"  ariadne <command> [flags]",
		"",
		"Commands:",
		"  resolve  Resolve a supported album or song URL across services.",
		"",
		resolveHelpTextFor(baseConfig, configPath),
	}, "\n")
}
