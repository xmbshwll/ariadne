package main

// Program entry: main, the root cobra command, help rendering, and raw-argv helpers.

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"os"

	"errors"

	"github.com/spf13/cobra"
	"github.com/xmbshwll/ariadne"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, rootError(err))
		os.Exit(1)
	}
}

func rootError(err error) error {
	for {
		unwrapped := errors.Unwrap(err)
		if unwrapped == nil {
			return err
		}
		err = unwrapped
	}
}

func run(args []string, stdout io.Writer, stderr io.Writer) error {
	configPath := configPathFromArgs(args)
	helpConfig := ariadne.DefaultConfig()
	commandArgs := argsWithoutPersistentFlags(args)

	switch {
	case len(commandArgs) == 0:
		if err := renderRootHelp(stderr, helpConfig, configPath); err != nil {
			return fmt.Errorf("print usage: %w", err)
		}
		return errMissingCommand
	case isHelpArg(commandArgs[0]):
		if len(commandArgs) == 1 {
			return renderRootHelp(stdout, helpConfig, configPath)
		}
		return executeRootCommand(stdout, stderr, helpConfig, configPath, nil, args)
	case containsHelpArg(commandArgs[1:]):
		return executeRootCommand(stdout, stderr, helpConfig, configPath, nil, args)
	}

	logger, err := newCLILoggerFromArgs(args, stderr)
	if err != nil {
		return err
	}

	baseConfig, err := loadCLIConfigWithLogger(configPath, logger)
	if err != nil {
		return err
	}

	unknownCommand := firstCommandArg(commandArgs, args)
	if err := executeRootCommand(stdout, stderr, baseConfig, configPath, logger, args); err != nil {
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

func executeRootCommand(stdout io.Writer, stderr io.Writer, baseConfig ariadne.Config, configPath string, logger *cliLogger, args []string) error {
	root := newRootCmd(stdout, stderr, baseConfig, configPath, logger)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		return fmt.Errorf("execute root command: %w", err)
	}
	return nil
}

func newRootCmd(stdout io.Writer, stderr io.Writer, baseConfig ariadne.Config, configPath string, logger *cliLogger) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "ariadne",
		Short:         "Resolve music URLs across services.",
		Version:       currentVersion().String(),
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.PersistentFlags().String("config", configPath, "configuration source (values: empty string to disable file loading, or a path to an .env, yaml, yml, json, or toml file)")
	cmd.PersistentFlags().String("log-level", defaultCLILogLevel.String(), "CLI log level (values: error, warn, info, debug; debug prints effective config including secrets)")
	cmd.AddCommand(newResolveCmd(baseConfig, configPath, logger))
	return cmd
}

func newResolveCmd(baseConfig ariadne.Config, configPath string, logger *cliLogger) *cobra.Command {
	config := defaultResolveConfig(baseConfig)

	cmd := &cobra.Command{
		Use:   defaultResolveCommandUse,
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
			return runResolveConfig(normalized, cmd.OutOrStdout(), logger)
		},
	}

	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		_, _ = io.WriteString(cmd.OutOrStdout(), resolveHelpTextFor(baseConfig, configPath))
	})

	bindResolveFlags(cmd.Flags(), &config)
	return cmd
}

const resolveHelpText = `Resolve a supported music URL across music services.

Usage:
  ariadne %s

Positional parameter:
  <url>
    Required.
    Values: a supported album URL from Apple Music, Deezer, Spotify, TIDAL,
    SoundCloud, YouTube Music, Bandcamp, or Amazon Music, or a supported song
    URL from Apple Music, Bandcamp, Deezer, SoundCloud, Spotify, or TIDAL.
    Amazon Music URLs and YouTube Music song URLs are recognized for parsing,
    but runtime hydration remains deferred.
    Behavior: when neither --song nor --album is set, Ariadne asks the library
    to auto-detect the resource type from the URL.

Flags:
  --config
    Values: empty string to disable file loading, or a path to a config file.
    Supported file styles: .env-style key=value files, plus Viper-supported structured files such as yaml, yml, json, or toml.
    Default: %s
    Behavior: config file values are loaded first, environment variables override them, and explicit CLI flags override both.

  --log-level
    Values: error, warn, info, debug.
    Default: error.
    Environment override: ARIADNE_LOG_LEVEL.
    Behavior: writes CLI diagnostics to stderr. debug prints effective configuration, including secrets loaded from env or config files.

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
    Values: comma-separated list drawn from %s.
    Use this to limit which target services are searched.
    Caveats:
%s

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
    Default: %s.
    Sets the overall timeout for one resolve command across parsing, source fetch, and target searches.

Notes:
  - Apple Music UPC and ISRC target search are enabled when APPLE_MUSIC_KEY_ID, APPLE_MUSIC_TEAM_ID, and APPLE_MUSIC_PRIVATE_KEY_PATH are set.
%s
%s`

func renderRootHelp(w io.Writer, baseConfig ariadne.Config, configPath string) error {
	if _, err := io.WriteString(w, rootHelpTextFor(baseConfig, configPath)); err != nil {
		return fmt.Errorf("%w: %w", errRenderResolveHelp, err)
	}
	return nil
}

func resolveHelpTextFor(baseConfig ariadne.Config, configPath string) string {
	return fmt.Sprintf(
		resolveHelpText,
		defaultResolveCommandUse,
		helpConfigPath(configPath),
		targetServiceNamesUsage(),
		helpServiceCaveats(),
		helpStorefrontDefault(baseConfig),
		baseConfig.HTTPTimeout,
		defaultResolveTimeout,
		helpSongHydrationNote(),
		strings.Join(helpServiceNotes(), "\n"),
	)
}

func helpConfigPath(configPath string) string {
	if configPath == "" {
		return `"" (disable file loading)`
	}
	return configPath
}

func helpStorefrontDefault(baseConfig ariadne.Config) string {
	if baseConfig.AppleMusicStorefront == "" {
		return "APPLE_MUSIC_STOREFRONT or us"
	}
	return baseConfig.AppleMusicStorefront
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

func argsWithoutPersistentFlags(args []string) []string {
	return argsWithoutNamedFlags(args, "--config", "--log-level")
}

func argsWithoutNamedFlags(args []string, flags ...string) []string {
	filtered := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if matchesNamedFlag(arg, flags...) {
			if i+1 < len(args) {
				value := args[i+1]
				if value == "" || !strings.HasPrefix(value, "-") {
					i++
				}
			}
			continue
		}
		if matchesNamedFlagAssignment(arg, flags...) {
			continue
		}
		filtered = append(filtered, arg)
	}
	return filtered
}

func matchesNamedFlag(arg string, flags ...string) bool {
	return slices.Contains(flags, arg)
}

func matchesNamedFlagAssignment(arg string, flags ...string) bool {
	return slices.ContainsFunc(flags, func(flag string) bool {
		return strings.HasPrefix(arg, flag+"=")
	})
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
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "unknown command ")
}
