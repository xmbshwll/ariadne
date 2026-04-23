package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/xmbshwll/ariadne/cmd/internal/validation"
	"github.com/xmbshwll/ariadne/internal/applemusicauth"
	"github.com/xmbshwll/ariadne/internal/config"
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/parse"
)

var (
	errAppleMusicCredentialsRequired = errors.New("APPLE_MUSIC_KEY_ID, APPLE_MUSIC_TEAM_ID, and APPLE_MUSIC_PRIVATE_KEY_PATH must be set")
	errAppleMusicValidateUsage       = errors.New("usage: go run ./cmd/validate-apple-music-official [-url <apple-music-album-url>] [-sample-url-file <path>] [-out-dir <dir>] [-storefront <code>]")
	errAppleMusicSampleURLEmpty      = errors.New("apple music sample url file is empty")
	errAppleMusicSampleURLRequired   = errors.New("provide either -url or -sample-url-file")
)

type options struct {
	sampleURL     string
	sampleURLPath string
	outputDir     string
	apiBaseURL    string
	storefront    string
}

type validationInputs struct {
	opts           options
	appConfig      config.Config
	developerToken string
	rawURL         string
	outputDir      string
	parsed         *model.ParsedAlbumURL
	storefront     string
}

func (i validationInputs) OutputDir() string {
	return i.outputDir
}

func (i validationInputs) SuccessMessage() string {
	return "wrote Apple Music official artifacts to " + i.outputDir
}

type validationArtifacts struct {
	albumBody    []byte
	metadataBody []byte
	upcBody      []byte
	isrcBody     []byte
	summary      map[string]any
}

func loadValidationInputs(args []string) (validationInputs, error) {
	opts, err := parseFlags(args)
	if err != nil {
		return validationInputs{}, err
	}

	appConfig := config.Load()
	if !appConfig.AppleMusic.AuthEnabled() {
		return validationInputs{}, errAppleMusicCredentialsRequired
	}
	developerToken, err := applemusicauth.GenerateDeveloperToken(applemusicauth.Config{
		KeyID:          appConfig.AppleMusic.KeyID,
		TeamID:         appConfig.AppleMusic.TeamID,
		PrivateKeyPath: appConfig.AppleMusic.PrivateKeyPath,
	}, time.Now().UTC())
	if err != nil {
		return validationInputs{}, fmt.Errorf("generate apple music developer token: %w", err)
	}

	rawURL, err := validation.LoadSampleURL(opts.sampleURL, opts.sampleURLPath, "apple music", errAppleMusicSampleURLRequired, errAppleMusicSampleURLEmpty)
	if err != nil {
		return validationInputs{}, fmt.Errorf("load apple music sample url: %w", err)
	}
	outputDir, err := validation.ResolveOutputDir(opts.outputDir, "ariadne-apple-music-validation-")
	if err != nil {
		return validationInputs{}, fmt.Errorf("resolve apple music output dir: %w", err)
	}
	parsed, err := parse.AppleMusicAlbumURL(rawURL)
	if err != nil {
		return validationInputs{}, fmt.Errorf("parse sample apple music album url: %w", err)
	}

	return validationInputs{
		opts:           opts,
		appConfig:      appConfig,
		developerToken: developerToken,
		rawURL:         rawURL,
		outputDir:      outputDir,
		parsed:         parsed,
		storefront:     resolveStorefront(opts.storefront, parsed.RegionHint, appConfig.AppleMusic.Storefront),
	}, nil
}

func parseFlags(args []string) (options, error) {
	opts := options{
		apiBaseURL: defaultAPIBaseURL,
	}

	fs := flag.NewFlagSet("validate-apple-music-official", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&opts.sampleURL, "url", "", "apple music album url to validate")
	fs.StringVar(&opts.sampleURLPath, "sample-url-file", opts.sampleURLPath, "path to file containing apple music album url")
	fs.StringVar(&opts.outputDir, "out-dir", opts.outputDir, "directory for validation artifacts")
	fs.StringVar(&opts.apiBaseURL, "api-base-url", opts.apiBaseURL, "apple music api base url")
	fs.StringVar(&opts.storefront, "storefront", "", "apple music storefront override")
	if err := fs.Parse(args); err != nil {
		return options{}, errAppleMusicValidateUsage
	}
	if len(fs.Args()) != 0 {
		return options{}, errAppleMusicValidateUsage
	}
	return opts, nil
}
