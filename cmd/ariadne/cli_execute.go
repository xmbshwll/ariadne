package main

import (
	"context"
	"fmt"
	"io"
	"strings"
)

func runResolve(args []string, stdout io.Writer) error {
	baseConfig, err := loadCLIConfigWithLogger(configPathFromArgs(args), nil)
	if err != nil {
		return err
	}
	config, err := parseResolveArgs(args, baseConfig)
	if err != nil {
		return err
	}
	return executeResolve(config, stdout, nil, resolveModeFromConfig(config))
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
			//nolint:wrapcheck // main prints the root cause without extra CLI wrappers.
			return fail(err)
		}
		if resolution == nil {
			return fail(emptyResolutionError)
		}
		return succeed(writeCLISongOutput(stdout, *resolution, config))
	case resolveModeAlbum:
		resolution, err := resolver.ResolveAlbum(ctx, config.inputURL)
		if err != nil {
			//nolint:wrapcheck // main prints the root cause without extra CLI wrappers.
			return fail(err)
		}
		if resolution == nil {
			return fail(emptyResolutionError)
		}
		return succeed(writeCLIOutput(stdout, *resolution, config))
	case resolveModeAuto:
		resolution, err := resolver.Resolve(ctx, config.inputURL)
		if err != nil {
			//nolint:wrapcheck // main prints the root cause without extra CLI wrappers.
			return fail(err)
		}
		if resolution == nil {
			return fail(emptyResolutionError)
		}
		if resolution.Song != nil {
			return succeed(writeCLISongOutput(stdout, *resolution.Song, config))
		}
		if resolution.Album != nil {
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
