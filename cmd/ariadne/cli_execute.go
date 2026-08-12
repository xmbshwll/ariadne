package main

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/xmbshwll/ariadne"
)

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
	return fmt.Errorf("%w: %s: %w", errAllTargetSearchesFailed, strings.Join(services, ", "), failures[services[0]])
}
