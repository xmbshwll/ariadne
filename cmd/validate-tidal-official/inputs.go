package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/xmbshwll/ariadne/cmd/internal/validation"
	"github.com/xmbshwll/ariadne/internal/config"
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/parse"
)

var (
	errTIDALCredentialsRequired = errors.New("TIDAL_CLIENT_ID and TIDAL_CLIENT_SECRET must be set")
	errTIDALValidateUsage       = errors.New("usage: go run ./cmd/validate-tidal-official [-url <tidal-album-url>] [-sample-url-file <path>] [-out-dir <dir>] [-country-code <cc>]")
	errTIDALSampleURLEmpty      = errors.New("tidal sample url file is empty")
	errTIDALSampleURLRequired   = errors.New("provide either -url or -sample-url-file")
)

type options struct {
	sampleURL     string
	sampleURLPath string
	outputDir     string
	apiBaseURL    string
	authBaseURL   string
	countryCode   string
}

type validationInputs struct {
	opts        options
	appConfig   config.Config
	rawURL      string
	outputDir   string
	parsed      *model.ParsedAlbumURL
	countryCode string
}

func (i validationInputs) OutputDir() string {
	return i.outputDir
}

func (i validationInputs) SuccessMessage() string {
	return "wrote TIDAL official artifacts to " + i.outputDir
}

type validationArtifacts struct {
	targets map[string][]byte
	summary map[string]any
}

func loadValidationInputs(args []string) (validationInputs, error) {
	opts, err := parseFlags(args)
	if err != nil {
		return validationInputs{}, err
	}

	appConfig := config.Load()
	if !appConfig.TIDAL.Enabled() {
		return validationInputs{}, errTIDALCredentialsRequired
	}

	rawURL, err := validation.LoadSampleURL(opts.sampleURL, opts.sampleURLPath, "tidal", errTIDALSampleURLRequired, errTIDALSampleURLEmpty)
	if err != nil {
		return validationInputs{}, fmt.Errorf("load tidal sample url: %w", err)
	}
	parsed, err := parse.TIDALAlbumURL(rawURL)
	if err != nil {
		return validationInputs{}, fmt.Errorf("parse sample tidal album url: %w", err)
	}
	outputDir, err := validation.ResolveOutputDir(opts.outputDir, "ariadne-tidal-validation-")
	if err != nil {
		return validationInputs{}, fmt.Errorf("resolve tidal output dir: %w", err)
	}

	return validationInputs{
		opts:        opts,
		appConfig:   appConfig,
		rawURL:      rawURL,
		outputDir:   outputDir,
		parsed:      parsed,
		countryCode: normalizeCountryCode(opts.countryCode),
	}, nil
}

func normalizeCountryCode(raw string) string {
	countryCode := strings.ToUpper(strings.TrimSpace(raw))
	if countryCode == "" {
		return defaultCountryCode
	}
	return countryCode
}

func parseFlags(args []string) (options, error) {
	opts := options{
		apiBaseURL:  defaultTIDALAPIBaseURL,
		authBaseURL: defaultTIDALAuthBaseURL,
		countryCode: defaultCountryCode,
	}

	fs := flag.NewFlagSet("validate-tidal-official", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&opts.sampleURL, "url", "", "tidal album url to validate")
	fs.StringVar(&opts.sampleURLPath, "sample-url-file", opts.sampleURLPath, "path to file containing tidal album url")
	fs.StringVar(&opts.outputDir, "out-dir", opts.outputDir, "directory for validation artifacts")
	fs.StringVar(&opts.apiBaseURL, "api-base-url", opts.apiBaseURL, "tidal api base url")
	fs.StringVar(&opts.authBaseURL, "auth-base-url", opts.authBaseURL, "tidal auth base url")
	fs.StringVar(&opts.countryCode, "country-code", opts.countryCode, "tidal country code")
	if err := fs.Parse(args); err != nil {
		return options{}, errTIDALValidateUsage
	}
	if len(fs.Args()) != 0 {
		return options{}, errTIDALValidateUsage
	}
	return opts, nil
}
