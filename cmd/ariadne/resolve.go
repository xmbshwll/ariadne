package main

// The resolve command: its configuration, flag binding and validation, execution across auto/song/album modes, and per-target failure reporting.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/xmbshwll/ariadne"
	"github.com/xmbshwll/ariadne/internal/wiring"
)

const (
	defaultConfigPath = ".env"
	outputFormatJSON  = "json"
	outputFormatYAML  = "yaml"
	outputFormatCSV   = "csv"
)

type resolveMode string

const (
	resolveModeAuto       resolveMode = "auto"
	resolveModeSong       resolveMode = "song"
	resolveModeAlbum      resolveMode = "album"
	defaultResolveTimeout             = 20 * time.Second
)

type resolveConfig struct {
	inputURL          string
	forceSong         bool
	forceAlbum        bool
	verbose           bool
	format            string
	requestedServices string
	minStrengthName   string
	minStrength       ariadne.MatchStrength
	resolutionTimeout time.Duration
	resolverConfig    ariadne.Config
}

func resolveCommandUse(timeout time.Duration) string {
	return fmt.Sprintf("resolve [--log-level=debug] [--song|--album] [--verbose] [--format=json|yaml|csv] [--services=spotify,deezer] [--min-strength=probable] [--apple-music-storefront=us] [--resolution-timeout=%s] <url>", timeout)
}

func resolveCommandUsage(timeout time.Duration) string {
	return "usage: ariadne " + resolveCommandUse(timeout)
}

var (
	defaultResolveCommandUse = resolveCommandUse(defaultResolveTimeout)
	resolveUsage             = resolveCommandUsage(defaultResolveTimeout)
	// resolverFactory builds the resolver one CLI run uses. It takes the narrow
	// entityResolver the resolve command actually calls, so tests can supply any
	// resolver instead of needing the library to expose adapter authoring.
	resolverFactory func(ariadne.Config) entityResolver = func(config ariadne.Config) entityResolver {
		return ariadne.New(config)
	}
	valueNormalizer = strings.NewReplacer("-", "", "_", "")
)

var (
	errNonPositiveCLIHTTPTimeout = errors.New("ARIADNE_HTTP_TIMEOUT must be positive")
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
	errTargetServiceCredentials  = errors.New("target search needs credentials")
	errUnsupportedMinStrength    = errors.New("unsupported min-strength")
	errEmptyResolution           = errors.New("empty resolution")
	errAllTargetSearchesFailed   = errors.New("all target searches failed")
	errUnsupportedResolveMode    = errors.New("unsupported resolve mode")
)

var matchStrengthByName = map[string]ariadne.MatchStrength{
	"veryweak": ariadne.MatchStrengthVeryWeak,
	"weak":     ariadne.MatchStrengthWeak,
	"probable": ariadne.MatchStrengthProbable,
	"strong":   ariadne.MatchStrengthStrong,
}

// entityResolver is what the resolve command needs from the library: turn one
// input URL into a resolved Entity Shape. ariadne.Resolver satisfies it.
type entityResolver interface {
	ResolveAlbum(ctx context.Context, inputURL string) (*ariadne.Resolution, error)
	ResolveSong(ctx context.Context, inputURL string) (*ariadne.SongResolution, error)
	Resolve(ctx context.Context, inputURL string) (*ariadne.EntityResolution, error)
}

func defaultResolveConfig(baseConfig ariadne.Config) resolveConfig {
	return resolveConfig{
		format:            outputFormatJSON,
		minStrengthName:   string(ariadne.MatchStrengthVeryWeak),
		minStrength:       ariadne.MatchStrengthVeryWeak,
		resolutionTimeout: defaultResolveTimeout,
		resolverConfig:    baseConfig,
	}
}

func bindResolveFlags(fs *pflag.FlagSet, config *resolveConfig) {
	fs.StringVar(&config.resolverConfig.AppleMusicStorefront, "apple-music-storefront", config.resolverConfig.AppleMusicStorefront, "preferred Apple Music storefront (values: ISO 3166-1 alpha-2 code such as us, gb, de, fr, jp, ca, au; used when the source URL has no storefront)")
	fs.BoolVar(&config.forceSong, "song", false, "force song resolution for the input URL")
	fs.BoolVar(&config.forceAlbum, "album", false, "force album resolution for the input URL")
	fs.BoolVarP(&config.verbose, "verbose", "v", false, "print full resolution details (values: true or false; false emits compact links, true emits metadata, scores, reasons, and alternates)")
	fs.StringVar(&config.format, "format", config.format, "output format (values: json for structured output, yaml for YAML, csv for spreadsheet-friendly export)")
	fs.StringVar(&config.requestedServices, "services", "", "comma-separated target services (values: "+targetServiceNamesUsage()+")")
	fs.StringVar(&config.minStrengthName, "min-strength", config.minStrengthName, "minimum match strength (values: very_weak, weak, probable, strong; filters weaker results out of the final output)")
	fs.DurationVar(&config.resolverConfig.HTTPTimeout, "http-timeout", config.resolverConfig.HTTPTimeout, "per-request HTTP timeout (values: Go durations such as 5s, 15s, 30s, 1m; applies to Ariadne's default client)")
	fs.DurationVar(&config.resolutionTimeout, "resolution-timeout", config.resolutionTimeout, "overall resolution timeout (values: Go durations such as 20s, 30s, 1m, 2m; bounds the full resolve operation across all services)")
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

	return normalizeAndValidateResolveConfig(config)
}

func normalizeAndValidateResolveConfig(config resolveConfig) (resolveConfig, error) {
	normalized, err := normalizeResolveConfig(config)
	if err != nil {
		return resolveConfig{}, err
	}
	if err := validateResolveConfig(normalized); err != nil {
		return resolveConfig{}, err
	}
	return normalized, nil
}

func normalizeResolveConfig(config resolveConfig) (resolveConfig, error) {
	if config.forceSong && config.forceAlbum {
		return resolveConfig{}, errConflictingEntityModeFlag
	}

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
	if config.resolutionTimeout <= 0 {
		config.resolutionTimeout = defaultResolveTimeout
	}
	return config, nil
}

func validateResolveConfig(config resolveConfig) error {
	if !requiresSongTargetValidation(config) {
		return nil
	}

	for _, service := range config.resolverConfig.TargetServices {
		decision := evaluateTarget(config.resolverConfig, string(service), wiring.EntityShapeSong)
		if decision.Status == wiring.TargetServiceRequestAvailable {
			continue
		}
		return targetServiceDecisionError(errUnsupportedSongService, decision)
	}
	return nil
}

func requiresSongTargetValidation(config resolveConfig) bool {
	switch resolveModeFromConfig(config) {
	case resolveModeSong:
		return true
	case resolveModeAuto:
		return wiring.Default.SupportsRuntimeSongInputURL(config.inputURL)
	default:
		return false
	}
}

func resolveModeFromConfig(config resolveConfig) resolveMode {
	if config.forceSong {
		return resolveModeSong
	}
	if config.forceAlbum {
		return resolveModeAlbum
	}
	return resolveModeAuto
}

func runResolveConfig(config resolveConfig, stdout io.Writer, logger *cliLogger) error {
	return executeResolve(config, stdout, logger, resolveModeFromConfig(config))
}

func runResolve(args []string, stdout io.Writer) error {
	baseConfig, err := loadCLIConfigWithLogger(configPathFromArgs(args), nil)
	if err != nil {
		return err
	}
	config, err := parseResolveArgs(args, baseConfig)
	if err != nil {
		return err
	}
	return runResolveConfig(config, stdout, nil)
}

func executeResolve(config resolveConfig, stdout io.Writer, logger *cliLogger, mode resolveMode) error {
	logResolveStart(logger, config, mode)

	resolver := resolverFactory(config.resolverConfig)
	ctx, cancel := context.WithTimeout(context.Background(), config.resolutionTimeout)
	defer cancel()

	emptyResolutionError := fmt.Errorf("resolve %q: %w", config.inputURL, errEmptyResolution)
	fail := func(err error) error {
		logResolveFailure(logger, config, mode, err)
		return err
	}
	succeed := func(err error) error {
		if err != nil {
			return fail(err)
		}
		logResolveSuccess(logger, config, mode)
		return nil
	}

	switch mode {
	case resolveModeSong:
		resolution, err := resolver.ResolveSong(ctx, config.inputURL)
		if err != nil {
			return fail(err)
		}
		if resolution == nil {
			return fail(emptyResolutionError)
		}
		if err := songTargetFailure(resolution.Matches, logger); err != nil {
			return fail(err)
		}
		return succeed(writeCLISongOutput(stdout, *resolution, config))
	case resolveModeAlbum:
		resolution, err := resolver.ResolveAlbum(ctx, config.inputURL)
		if err != nil {
			return fail(err)
		}
		if resolution == nil {
			return fail(emptyResolutionError)
		}
		if err := albumTargetFailure(resolution.Matches, logger); err != nil {
			return fail(err)
		}
		return succeed(writeCLIOutput(stdout, *resolution, config))
	case resolveModeAuto:
		resolution, err := resolver.Resolve(ctx, config.inputURL)
		if err != nil {
			return fail(err)
		}
		if resolution == nil {
			return fail(emptyResolutionError)
		}
		if resolution.Song != nil {
			if err := songTargetFailure(resolution.Song.Matches, logger); err != nil {
				return fail(err)
			}
			return succeed(writeCLISongOutput(stdout, *resolution.Song, config))
		}
		if resolution.Album != nil {
			if err := albumTargetFailure(resolution.Album.Matches, logger); err != nil {
				return fail(err)
			}
			return succeed(writeCLIOutput(stdout, *resolution.Album, config))
		}
		return fail(emptyResolutionError)
	default:
		return fail(fmt.Errorf("%w %q", errUnsupportedResolveMode, mode))
	}
}

func logResolveStart(logger *cliLogger, config resolveConfig, mode resolveMode) {
	services := strings.Join(serviceNames(config.resolverConfig.TargetServices), ",")
	if services == "" {
		services = "default"
	}

	logger.Debugf("resolve start mode=%s url=%q", mode, config.inputURL)
	logger.Debugf(
		"resolve settings format=%s verbose=%t min_strength=%s services=%q http_timeout=%s resolution_timeout=%s",
		config.format,
		config.verbose,
		config.minStrength,
		services,
		config.resolverConfig.HTTPTimeout,
		config.resolutionTimeout,
	)
}

func logResolveFailure(logger *cliLogger, config resolveConfig, mode resolveMode, err error) {
	logger.Debugf("resolve failed mode=%s url=%q error=%v", mode, config.inputURL, err)
}

func logResolveSuccess(logger *cliLogger, config resolveConfig, mode resolveMode) {
	logger.Debugf("resolve complete mode=%s url=%q", mode, config.inputURL)
}

// albumTargetFailure warns about per-service Target Search failures and
// reports a hard failure when every requested target failed.
func albumTargetFailure(matches map[ariadne.ServiceName]ariadne.MatchResult, logger *cliLogger) error {
	return targetFailure(matches, func(match ariadne.MatchResult) error { return match.Err }, logger)
}

func songTargetFailure(matches map[ariadne.ServiceName]ariadne.SongMatchResult, logger *cliLogger) error {
	return targetFailure(matches, func(match ariadne.SongMatchResult) error { return match.Err }, logger)
}

func targetFailure[M any](matches map[ariadne.ServiceName]M, matchErr func(M) error, logger *cliLogger) error {
	failures := make(map[string]error, len(matches))
	for service, match := range matches {
		err := matchErr(match)
		if err == nil {
			continue
		}
		logger.Warnf("target search failed service=%s error=%v", service, err)
		failures[string(service)] = err
	}
	return allTargetsFailedError(len(matches), failures)
}

func allTargetsFailedError(total int, failures map[string]error) error {
	if total == 0 || len(failures) != total {
		return nil
	}
	services := make([]string, 0, len(failures))
	for service := range failures {
		services = append(services, service)
	}
	sort.Strings(services)
	errs := make([]error, 0, len(services))
	for _, service := range services {
		errs = append(errs, failures[service])
	}
	return fmt.Errorf("%w: %s: %w", errAllTargetSearchesFailed, strings.Join(services, ", "), errors.Join(errs...))
}
