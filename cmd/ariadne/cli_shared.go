package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/xmbshwll/ariadne"
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
	resolverFactory          = ariadne.New
	valueNormalizer          = strings.NewReplacer("-", "", "_", "")
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
	errSpotifyTargetCredentials  = errors.New("spotify target search requires SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET")
	errTIDALTargetCredentials    = errors.New("tidal target search requires TIDAL_CLIENT_ID and TIDAL_CLIENT_SECRET")
	errUnsupportedMinStrength    = errors.New("unsupported min-strength")
	errEmptyResolution           = errors.New("empty resolution")
	errUnsupportedResolveMode    = errors.New("unsupported resolve mode")
)

var matchStrengthByName = map[string]ariadne.MatchStrength{
	"veryweak": ariadne.MatchStrengthVeryWeak,
	"weak":     ariadne.MatchStrengthWeak,
	"probable": ariadne.MatchStrengthProbable,
	"strong":   ariadne.MatchStrengthStrong,
}
