package main

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/xmbshwll/ariadne/cmd/internal/validation"
)

func buildValidationSummary(inputs validationInputs, title, artist, releaseDate, label, upc string, isrcs []string) map[string]any {
	artifacts := map[string]string{
		"source_payload_official":  filepath.ToSlash(filepath.Join(inputs.outputDir, "source-payload-official.json")),
		"search_metadata_official": filepath.ToSlash(filepath.Join(inputs.outputDir, "search-metadata-official.json")),
		"official_summary":         filepath.ToSlash(filepath.Join(inputs.outputDir, "official-summary.json")),
	}
	if upc != "" {
		artifacts["search_upc_official"] = filepath.ToSlash(filepath.Join(inputs.outputDir, "search-upc-official.json"))
	}
	if len(isrcs) > 0 {
		artifacts["search_isrc_official"] = filepath.ToSlash(filepath.Join(inputs.outputDir, "search-isrc-official.json"))
	}

	return map[string]any{
		"sample_url":         inputs.rawURL,
		"album_id":           inputs.parsed.ID,
		"canonical_url":      inputs.parsed.CanonicalURL,
		"storefront":         inputs.storefront,
		"auth_mode":          "generated_p8_token",
		"title":              title,
		"artists":            nonEmptyStrings(artist),
		"release_date":       releaseDate,
		"label":              label,
		"upc":                upc,
		"track_isrc_samples": isrcs,
		"generated_at":       time.Now().UTC().Format(time.RFC3339),
		"artifacts":          artifacts,
	}
}

func writeValidationArtifacts(outputDir string, artifacts validationArtifacts) error {
	sourcePayloadPath := filepath.Join(outputDir, "source-payload-official.json")
	if err := validation.WritePrettyJSON(sourcePayloadPath, artifacts.albumBody); err != nil {
		return fmt.Errorf("write %s: %w", sourcePayloadPath, err)
	}
	metadataPath := filepath.Join(outputDir, "search-metadata-official.json")
	if err := validation.WritePrettyJSON(metadataPath, artifacts.metadataBody); err != nil {
		return fmt.Errorf("write %s: %w", metadataPath, err)
	}
	if len(artifacts.upcBody) > 0 {
		upcPath := filepath.Join(outputDir, "search-upc-official.json")
		if err := validation.WritePrettyJSON(upcPath, artifacts.upcBody); err != nil {
			return fmt.Errorf("write %s: %w", upcPath, err)
		}
	}
	if len(artifacts.isrcBody) > 0 {
		isrcPath := filepath.Join(outputDir, "search-isrc-official.json")
		if err := validation.WritePrettyJSON(isrcPath, artifacts.isrcBody); err != nil {
			return fmt.Errorf("write %s: %w", isrcPath, err)
		}
	}
	summaryPath := filepath.Join(outputDir, "official-summary.json")
	if err := validation.WriteJSON(summaryPath, artifacts.summary); err != nil {
		return fmt.Errorf("write %s: %w", summaryPath, err)
	}
	return nil
}
