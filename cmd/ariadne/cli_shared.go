package main

import (
	"errors"
	"strings"
	"time"

	"github.com/xmbshwll/ariadne"
)

const (
	defaultConfigPath = ".env"
	outputFormatJSON  = "json"
	outputFormatYAML  = "yaml"
	outputFormatCSV   = "csv"
	resolveUsage      = "usage: ariadne resolve [--log-level=debug] [--song|--album] [--verbose] [--format=json|yaml|csv] [--services=spotify,deezer] [--min-strength=probable] [--apple-music-storefront=us] [--resolution-timeout=20s] <url>"
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

var (
	resolverFactory = ariadne.New
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
	errSpotifyTargetCredentials  = errors.New("spotify target search requires SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET")
	errTIDALTargetCredentials    = errors.New("tidal target search requires TIDAL_CLIENT_ID and TIDAL_CLIENT_SECRET")
	errUnsupportedMinStrength    = errors.New("unsupported min-strength")
	errEmptyResolution           = errors.New("empty resolution")
	errUnsupportedResolveMode    = errors.New("unsupported resolve mode")
)

var matchStrengthByName = map[string]ariadne.MatchStrength{
	"veryweak":  ariadne.MatchStrengthVeryWeak,
	"very_weak": ariadne.MatchStrengthVeryWeak,
	"weak":      ariadne.MatchStrengthWeak,
	"probable":  ariadne.MatchStrengthProbable,
	"strong":    ariadne.MatchStrengthStrong,
}
