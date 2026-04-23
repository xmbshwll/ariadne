package main

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/xmbshwll/ariadne/cmd/internal/validation"
)

const (
	appleMusicSourcePayloadFile   = "source-payload-official.json"
	appleMusicSearchMetadataFile  = "search-metadata-official.json"
	appleMusicSearchUPCFile       = "search-upc-official.json"
	appleMusicSearchISRCFile      = "search-isrc-official.json"
	appleMusicOfficialSummaryFile = "official-summary.json"
)

func buildValidationSummary(inputs validationInputs, artifacts validationArtifacts, title, artist, releaseDate, label, upc string, isrcs []string) map[string]any {
	artifactPaths := map[string]string{
		"source_payload_official":  validationArtifactPath(inputs.outputDir, appleMusicSourcePayloadFile),
		"search_metadata_official": validationArtifactPath(inputs.outputDir, appleMusicSearchMetadataFile),
		"official_summary":         validationArtifactPath(inputs.outputDir, appleMusicOfficialSummaryFile),
	}
	if len(artifacts.upcBody) > 0 {
		artifactPaths["search_upc_official"] = validationArtifactPath(inputs.outputDir, appleMusicSearchUPCFile)
	}
	if len(artifacts.isrcBody) > 0 {
		artifactPaths["search_isrc_official"] = validationArtifactPath(inputs.outputDir, appleMusicSearchISRCFile)
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
		"artifacts":          artifactPaths,
	}
}

func writeValidationArtifacts(outputDir string, artifacts validationArtifacts) error {
	sourcePayloadPath := filepath.Join(outputDir, appleMusicSourcePayloadFile)
	if err := validation.WritePrettyJSON(sourcePayloadPath, artifacts.albumBody); err != nil {
		return fmt.Errorf("write %s: %w", sourcePayloadPath, err)
	}
	metadataPath := filepath.Join(outputDir, appleMusicSearchMetadataFile)
	if err := validation.WritePrettyJSON(metadataPath, artifacts.metadataBody); err != nil {
		return fmt.Errorf("write %s: %w", metadataPath, err)
	}
	if len(artifacts.upcBody) > 0 {
		upcPath := filepath.Join(outputDir, appleMusicSearchUPCFile)
		if err := validation.WritePrettyJSON(upcPath, artifacts.upcBody); err != nil {
			return fmt.Errorf("write %s: %w", upcPath, err)
		}
	}
	if len(artifacts.isrcBody) > 0 {
		isrcPath := filepath.Join(outputDir, appleMusicSearchISRCFile)
		if err := validation.WritePrettyJSON(isrcPath, artifacts.isrcBody); err != nil {
			return fmt.Errorf("write %s: %w", isrcPath, err)
		}
	}
	summaryPath := filepath.Join(outputDir, appleMusicOfficialSummaryFile)
	if err := validation.WriteJSON(summaryPath, artifacts.summary); err != nil {
		return fmt.Errorf("write %s: %w", summaryPath, err)
	}
	return nil
}

func validationArtifactPath(outputDir, name string) string {
	return filepath.ToSlash(filepath.Join(outputDir, name))
}
