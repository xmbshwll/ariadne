package main

import (
	"encoding/json"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"

	"github.com/xmbshwll/ariadne"
)

// resolutionOutput[DTO] is the per-Entity-Shape output adapter: one shape for
// the renderers, one links map for compact mode, and the two CSV writers. The
// dispatch (filter by strength, pick compact or verbose, format) is written
// once in writeResolutionOutput.
type resolutionOutput[DTO any] struct {
	compact    func() any // the links map for JSON/YAML compact mode
	verbose    func() any // the full DTO for JSON/YAML verbose mode
	compactCSV func(io.Writer) error
	verboseCSV func(io.Writer) error
}

func writeCLIOutput(w io.Writer, resolution ariadne.Resolution, cfg resolveConfig) error {
	filtered := filterResolutionByStrength(resolution, cfg.minStrength)
	return writeResolutionOutput(w, cfg, resolutionOutput[cliResolution]{
		compact:    func() any { return newCLILinks(filtered) },
		verbose:    func() any { return newCLIResolution(filtered) },
		compactCSV: func(w io.Writer) error { return writeCompactCSV(w, filtered) },
		verboseCSV: func(w io.Writer) error { return writeVerboseCSV(w, filtered) },
	})
}

func writeCLISongOutput(w io.Writer, resolution ariadne.SongResolution, cfg resolveConfig) error {
	filtered := filterSongResolutionByStrength(resolution, cfg.minStrength)
	return writeResolutionOutput(w, cfg, resolutionOutput[cliSongResolution]{
		compact:    func() any { return newCLISongLinks(filtered) },
		verbose:    func() any { return newCLISongResolution(filtered) },
		compactCSV: func(w io.Writer) error { return writeCompactSongCSV(w, filtered) },
		verboseCSV: func(w io.Writer) error { return writeVerboseSongCSV(w, filtered) },
	})
}

// writeResolutionOutput applies the one dispatch every entity shape shares:
// compact renders links only, verbose renders the full DTO, and the format
// decides between JSON, YAML, and the matching CSV writer.
func writeResolutionOutput[DTO any](w io.Writer, cfg resolveConfig, output resolutionOutput[DTO]) error {
	if cfg.verbose {
		return writeFormattedOutput(w, output.verbose(), cfg.format, func() error { return output.verboseCSV(w) })
	}
	return writeFormattedOutput(w, output.compact(), cfg.format, func() error { return output.compactCSV(w) })
}

func writeFormattedOutput(w io.Writer, output any, format string, writeCSV func() error) error {
	switch format {
	case outputFormatJSON:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		// Titles and URLs carry raw text, not HTML; escaping would turn < and &
		// into \u003c and \u0026 for every script reading the output.
		encoder.SetEscapeHTML(false)
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
