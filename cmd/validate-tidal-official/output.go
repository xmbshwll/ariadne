package main

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/xmbshwll/ariadne/cmd/internal/validation"
)

func buildValidationSummary(inputs validationInputs, title string, artistNames []string, releaseDate string, upc string, trackTitles []string, trackISRCs []string) map[string]any {
	artifacts := map[string]string{
		"source_payload_official": filepath.ToSlash(filepath.Join(inputs.outputDir, "source-payload-official.json")),
		"search_albums_official":  filepath.ToSlash(filepath.Join(inputs.outputDir, "search-albums-official.json")),
		"official_summary":        filepath.ToSlash(filepath.Join(inputs.outputDir, "official-summary.json")),
	}
	if upc != "" {
		artifacts["search_upc_official"] = filepath.ToSlash(filepath.Join(inputs.outputDir, "search-upc-official.json"))
	}
	if len(trackISRCs) > 0 {
		artifacts["search_isrc_official"] = filepath.ToSlash(filepath.Join(inputs.outputDir, "search-isrc-official.json"))
	}

	return map[string]any{
		"sample_url":          inputs.rawURL,
		"album_id":            inputs.parsed.ID,
		"canonical_url":       inputs.parsed.CanonicalURL,
		"country_code":        inputs.countryCode,
		"title":               title,
		"artists":             artistNames,
		"release_date":        releaseDate,
		"upc":                 upc,
		"track_title_samples": trackTitles,
		"track_isrc_samples":  trackISRCs,
		"generated_at":        time.Now().UTC().Format(time.RFC3339),
		"token_acquired":      true,
		"artifacts":           artifacts,
	}
}

func writeValidationArtifacts(outputDir string, artifacts validationArtifacts) error {
	for name, raw := range artifacts.targets {
		path := filepath.Join(outputDir, name)
		if err := validation.WritePrettyJSON(path, raw); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
	}
	summaryPath := filepath.Join(outputDir, "official-summary.json")
	if err := validation.WriteJSON(summaryPath, artifacts.summary); err != nil {
		return fmt.Errorf("write %s: %w", summaryPath, err)
	}
	return nil
}
