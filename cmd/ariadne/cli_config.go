package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/xmbshwll/ariadne"
)

var (
	errNonPositiveCLIHTTPTimeout = errors.New("ARIADNE_HTTP_TIMEOUT must be positive")

	matchStrengthByName = map[string]ariadne.MatchStrength{
		"veryweak":  ariadne.MatchStrengthVeryWeak,
		"very_weak": ariadne.MatchStrengthVeryWeak,
		"weak":      ariadne.MatchStrengthWeak,
		"probable":  ariadne.MatchStrengthProbable,
		"strong":    ariadne.MatchStrengthStrong,
	}
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

func defaultResolveConfig(baseConfig ariadne.Config) resolveConfig {
	return resolveConfig{
		format:            outputFormatJSON,
		minStrengthName:   string(ariadne.MatchStrengthVeryWeak),
		minStrength:       ariadne.MatchStrengthVeryWeak,
		resolutionTimeout: defaultResolveTimeout,
		resolverConfig:    baseConfig,
	}
}

func configPathFromArgs(args []string) string {
	for i, arg := range args {
		switch {
		case arg == "--config":
			if i+1 >= len(args) {
				return ""
			}
			value := args[i+1]
			if strings.TrimSpace(value) == "" || strings.HasPrefix(value, "-") {
				return ""
			}
			return value
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

	httpTimeout := trimmedValue("ARIADNE_HTTP_TIMEOUT")
	if httpTimeout != "" {
		parsedTimeout, err := time.ParseDuration(httpTimeout)
		if err != nil {
			return ariadne.Config{}, fmt.Errorf("parse ARIADNE_HTTP_TIMEOUT %q: %w", httpTimeout, err)
		}
		if parsedTimeout <= 0 {
			return ariadne.Config{}, fmt.Errorf("invalid ARIADNE_HTTP_TIMEOUT %q: %w", httpTimeout, errNonPositiveCLIHTTPTimeout)
		}
		cfg.HTTPTimeout = parsedTimeout
	}

	loaded := ariadne.LoadConfigFromEnv(trimmedValue)
	loaded.HTTPTimeout = cfg.HTTPTimeout
	return loaded, nil
}

func looksLikeEnvFile(path string) bool {
	base := filepath.Base(path)
	return strings.HasPrefix(base, ".env") || strings.EqualFold(filepath.Ext(base), ".env")
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

func parseRequestedServices(raw string, appConfig ariadne.Config) ([]ariadne.ServiceName, error) {
	if strings.TrimSpace(raw) == "" {
		services := append([]ariadne.ServiceName(nil), appConfig.TargetServices...)
		for _, service := range services {
			if err := validateRequestedService(service, appConfig); err != nil {
				return nil, err
			}
		}
		return services, nil
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
	service, ok := ariadne.LookupServiceName(normalized)
	if !ok || !ariadne.SupportsTarget(service) {
		return "", fmt.Errorf("%w %q (expected one of the supported target services: %s)", errUnsupportedTargetService, raw, strings.Join(serviceNames(ariadne.SupportedTargetServices()), ", "))
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

func serviceNames(services []ariadne.ServiceName) []string {
	names := make([]string, 0, len(services))
	for _, service := range services {
		names = append(names, string(service))
	}
	return names
}
