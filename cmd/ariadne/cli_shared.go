package main

import (
	"context"
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
