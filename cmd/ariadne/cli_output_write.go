package main

import (
	"encoding/json"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"

	"github.com/xmbshwll/ariadne"
)

func writeCLIOutput(w io.Writer, resolution ariadne.Resolution, cfg resolveConfig) error {
	resolution = filterResolutionByStrength(resolution, cfg.minStrength)
	output := any(newCLILinks(resolution))
	if cfg.verbose {
		output = newCLIResolution(resolution)
	}
	return writeFormattedOutput(
		w,
		output,
		cfg.format,
		func() error {
			if cfg.verbose {
				return writeVerboseCSV(w, resolution)
			}
			return writeCompactCSV(w, resolution)
		},
	)
}

func writeCLISongOutput(w io.Writer, resolution ariadne.SongResolution, cfg resolveConfig) error {
	resolution = filterSongResolutionByStrength(resolution, cfg.minStrength)
	output := any(newCLISongLinks(resolution))
	if cfg.verbose {
		output = newCLISongResolution(resolution)
	}
	return writeFormattedOutput(
		w,
		output,
		cfg.format,
		func() error {
			if cfg.verbose {
				return writeVerboseSongCSV(w, resolution)
			}
			return writeCompactSongCSV(w, resolution)
		},
	)
}

func writeFormattedOutput(w io.Writer, output any, format string, writeCSV func() error) error {
	switch format {
	case outputFormatJSON:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(output); err != nil {
			return fmt.Errorf("encode output json: %w", err)
		}
		return nil
	case outputFormatYAML:
		data, err := yaml.Marshal(output)
		if err != nil {
			return fmt.Errorf("encode output yaml: %w", err)
		}
		if _, err := w.Write(data); err != nil {
			return fmt.Errorf("write output yaml: %w", err)
		}
		return nil
	case outputFormatCSV:
		return writeCSV()
	default:
		return fmt.Errorf("%w: %q", errUnsupportedFormat, format)
	}
}
