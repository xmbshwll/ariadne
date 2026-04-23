package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/xmbshwll/ariadne/cmd/internal/validation"
)

func buildValidationSummary(inputs validationInputs, album spotifyAlbumPayload, upc string, isrcs []string) map[string]any {
	return map[string]any{
		"sample_url":         inputs.rawURL,
		"album_id":           inputs.parsed.ID,
		"canonical_url":      inputs.parsed.CanonicalURL,
		"title":              strings.TrimSpace(album.Name),
		"artists":            albumArtists(album),
		"release_date":       strings.TrimSpace(album.ReleaseDate),
		"label":              strings.TrimSpace(album.Label),
		"upc":                upc,
		"track_isrc_samples": isrcs,
		"generated_at":       time.Now().UTC().Format(time.RFC3339),
		"artifacts": map[string]string{
			"source_payload_api":    filepath.ToSlash(filepath.Join(inputs.outputDir, "source-payload-api.json")),
			"search_upc_results":    filepath.ToSlash(filepath.Join(inputs.outputDir, "search-upc-results.json")),
			"search_isrc_results":   filepath.ToSlash(filepath.Join(inputs.outputDir, "search-isrc-results.json")),
			"search_metadata":       filepath.ToSlash(filepath.Join(inputs.outputDir, "search-metadata-results.json")),
			"authenticated_summary": filepath.ToSlash(filepath.Join(inputs.outputDir, "authenticated-summary.json")),
		},
	}
}

func writeValidationArtifacts(outputDir string, artifacts validationArtifacts) error {
	sourcePayloadPath := filepath.Join(outputDir, "source-payload-api.json")
	if err := validation.WritePrettyJSON(sourcePayloadPath, artifacts.albumBody); err != nil {
		return fmt.Errorf("write %s: %w", sourcePayloadPath, err)
	}
	searchUPCPath := filepath.Join(outputDir, "search-upc-results.json")
	if err := validation.WritePrettyJSON(searchUPCPath, artifacts.upcBody); err != nil {
		return fmt.Errorf("write %s: %w", searchUPCPath, err)
	}
	searchISRCPath := filepath.Join(outputDir, "search-isrc-results.json")
	if err := validation.WritePrettyJSON(searchISRCPath, artifacts.isrcBody); err != nil {
		return fmt.Errorf("write %s: %w", searchISRCPath, err)
	}
	searchMetadataPath := filepath.Join(outputDir, "search-metadata-results.json")
	if err := validation.WritePrettyJSON(searchMetadataPath, artifacts.metadataBody); err != nil {
		return fmt.Errorf("write %s: %w", searchMetadataPath, err)
	}
	summaryPath := filepath.Join(outputDir, "authenticated-summary.json")
	if err := validation.WriteJSON(summaryPath, artifacts.summary); err != nil {
		return fmt.Errorf("write %s: %w", summaryPath, err)
	}
	return nil
}
