package main

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/xmbshwll/ariadne/cmd/internal/validation"
	"github.com/xmbshwll/ariadne/internal/config"
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/parse"
)

var (
	errSpotifyCredentialsRequired = errors.New("SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET must be set")
	errSpotifyValidateUsage       = errors.New("usage: go run ./cmd/validate-spotify-auth [-url <spotify-album-url>] [-sample-url-file <path>] [-out-dir <dir>]")
	errSpotifySampleURLEmpty      = errors.New("spotify sample url file is empty")
	errSpotifySampleURLRequired   = errors.New("provide either -url or -sample-url-file")
)

type options struct {
	sampleURL     string
	sampleURLPath string
	outputDir     string
	apiBaseURL    string
	authBaseURL   string
}

type validationInputs struct {
	opts      options
	appConfig config.Config
	rawURL    string
	outputDir string
	parsed    *model.ParsedAlbumURL
}

func (i validationInputs) OutputDir() string {
	return i.outputDir
}

func (i validationInputs) SuccessMessage() string {
	return "wrote Spotify authenticated artifacts to " + i.outputDir
}

type validationArtifacts struct {
	albumBody    []byte
	upcBody      []byte
	isrcBody     []byte
	metadataBody []byte
	summary      map[string]any
}

func loadValidationInputs(args []string) (validationInputs, error) {
	opts, err := parseFlags(args)
	if err != nil {
		return validationInputs{}, err
	}

	appConfig := config.Load()
	if !appConfig.Spotify.Enabled() {
		return validationInputs{}, errSpotifyCredentialsRequired
	}

	rawURL, err := validation.LoadSampleURL(opts.sampleURL, opts.sampleURLPath, "spotify", errSpotifySampleURLRequired, errSpotifySampleURLEmpty)
	if err != nil {
		return validationInputs{}, fmt.Errorf("load spotify sample url: %w", err)
	}
	parsed, err := parse.SpotifyAlbumURL(rawURL)
	if err != nil {
		return validationInputs{}, fmt.Errorf("parse sample spotify album url: %w", err)
	}
	outputDir, err := validation.ResolveOutputDir(opts.outputDir, "ariadne-spotify-validation-")
	if err != nil {
		return validationInputs{}, fmt.Errorf("resolve spotify output dir: %w", err)
	}

	return validationInputs{
		opts:      opts,
		appConfig: appConfig,
		rawURL:    rawURL,
		outputDir: outputDir,
		parsed:    parsed,
	}, nil
}

func parseFlags(args []string) (options, error) {
	opts := options{
		apiBaseURL:  defaultSpotifyAPIBaseURL,
		authBaseURL: defaultSpotifyAuthBaseURL,
	}

	fs := flag.NewFlagSet("validate-spotify-auth", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&opts.sampleURL, "url", "", "spotify album url to validate")
	fs.StringVar(&opts.sampleURLPath, "sample-url-file", opts.sampleURLPath, "path to file containing spotify album url")
	fs.StringVar(&opts.outputDir, "out-dir", opts.outputDir, "directory for validation artifacts")
	fs.StringVar(&opts.apiBaseURL, "api-base-url", opts.apiBaseURL, "spotify api base url")
	fs.StringVar(&opts.authBaseURL, "auth-base-url", opts.authBaseURL, "spotify auth base url")
	if err := fs.Parse(args); err != nil {
		return options{}, errSpotifyValidateUsage
	}
	if len(fs.Args()) != 0 {
		return options{}, errSpotifyValidateUsage
	}
	return opts, nil
}
