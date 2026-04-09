package main

import (
	"context"
	"fmt"
	"io"
)

func runResolve(args []string, stdout io.Writer) error {
	baseConfig, err := loadCLIConfig(configPathFromArgs(args))
	if err != nil {
		return err
	}
	config, err := parseResolveArgs(args, baseConfig)
	if err != nil {
		return err
	}
	return executeResolve(config, stdout, resolveModeFromConfig(config))
}

func executeResolve(config resolveConfig, stdout io.Writer, mode resolveMode) error {
	resolver := resolverFactory(config.resolverConfig)
	ctx, cancel := context.WithTimeout(context.Background(), config.resolutionTimeout)
	defer cancel()

	switch mode {
	case resolveModeSong:
		resolution, err := resolver.ResolveSong(ctx, config.inputURL)
		if err != nil {
			//nolint:wrapcheck // main prints the root cause without extra CLI wrappers.
			return err
		}
		return writeCLISongOutput(stdout, *resolution, config)
	case resolveModeAlbum:
		resolution, err := resolver.ResolveAlbum(ctx, config.inputURL)
		if err != nil {
			//nolint:wrapcheck // main prints the root cause without extra CLI wrappers.
			return err
		}
		return writeCLIOutput(stdout, *resolution, config)
	case resolveModeAuto:
		resolution, err := resolver.Resolve(ctx, config.inputURL)
		if err != nil {
			//nolint:wrapcheck // main prints the root cause without extra CLI wrappers.
			return err
		}
		if resolution.Song != nil {
			return writeCLISongOutput(stdout, *resolution.Song, config)
		}
		if resolution.Album != nil {
			return writeCLIOutput(stdout, *resolution.Album, config)
		}
		return fmt.Errorf("resolve %q: %w", config.inputURL, errEmptyResolution)
	default:
		return fmt.Errorf("%w %q", errUnsupportedResolveMode, mode)
	}
}
