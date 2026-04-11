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
	output := newCLIOutput(resolution, cfg)

	switch cfg.format {
	case outputFormatJSON:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(output); err != nil {
			return fmt.Errorf("encode resolution json: %w", err)
		}
		return nil
	case outputFormatYAML:
		data, err := yaml.Marshal(output)
		if err != nil {
			return fmt.Errorf("encode resolution yaml: %w", err)
		}
		if _, err := w.Write(data); err != nil {
			return fmt.Errorf("write resolution yaml: %w", err)
		}
		return nil
	case outputFormatCSV:
		if cfg.verbose {
			return writeVerboseCSV(w, resolution)
		}
		return writeCompactCSV(w, resolution)
	default:
		return fmt.Errorf("%w: %q", errUnsupportedFormat, cfg.format)
	}
}

func newCLISongOutput(resolution ariadne.SongResolution, cfg resolveConfig) any {
	if cfg.verbose {
		return newCLISongResolution(resolution)
	}
	return newCLISongLinks(resolution)
}

func writeCLISongOutput(w io.Writer, resolution ariadne.SongResolution, cfg resolveConfig) error {
	resolution = filterSongResolutionByStrength(resolution, cfg.minStrength)
	output := newCLISongOutput(resolution, cfg)

	switch cfg.format {
	case outputFormatJSON:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(output); err != nil {
			return fmt.Errorf("encode song resolution json: %w", err)
		}
		return nil
	case outputFormatYAML:
		data, err := yaml.Marshal(output)
		if err != nil {
			return fmt.Errorf("encode song resolution yaml: %w", err)
		}
		if _, err := w.Write(data); err != nil {
			return fmt.Errorf("write song resolution yaml: %w", err)
		}
		return nil
	case outputFormatCSV:
		if cfg.verbose {
			return writeVerboseSongCSV(w, resolution)
		}
		return writeCompactSongCSV(w, resolution)
	default:
		return fmt.Errorf("%w: %q", errUnsupportedFormat, cfg.format)
	}
}
