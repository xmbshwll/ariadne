package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/pflag"

	"github.com/xmbshwll/ariadne"
)

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
	fs.StringVar(&config.requestedServices, "services", "", "comma-separated target services (values: appleMusic, bandcamp, deezer, soundcloud, spotify, tidal, youtubeMusic, ytmusic; ytmusic aliases youtubeMusic)")
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
		if ariadne.SupportsEnabledSongTarget(config.resolverConfig, service) {
			continue
		}
		return fmt.Errorf("%w %q (%s)", errUnsupportedSongService, service, enabledSongTargetServicesUsage(config.resolverConfig))
	}
	return nil
}

func requiresSongTargetValidation(config resolveConfig) bool {
	switch resolveModeFromConfig(config) {
	case resolveModeSong:
		return true
	case resolveModeAuto:
		return ariadne.SupportsRuntimeSongInputURL(config.inputURL)
	default:
		return false
	}
}

func enabledSongTargetServicesUsage(config ariadne.Config) string {
	names := serviceNames(ariadne.EnabledSongTargetServices(config))
	if len(names) == 0 {
		return "enabled for songs: none"
	}
	return "enabled for songs: " + strings.Join(names, ", ")
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
