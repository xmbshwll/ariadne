package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/xmbshwll/ariadne/cmd/internal/validation"
)

const (
	spotifySourcePayloadFile   = "source-payload-api.json"
	spotifySearchUPCFile       = "search-upc-results.json"
	spotifySearchISRCFile      = "search-isrc-results.json"
	spotifySearchMetadataFile  = "search-metadata-results.json"
	spotifyAuthenticatedReport = "authenticated-summary.json"
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
			"source_payload_api":    validationArtifactPath(inputs.outputDir, spotifySourcePayloadFile),
			"search_upc_results":    validationArtifactPath(inputs.outputDir, spotifySearchUPCFile),
			"search_isrc_results":   validationArtifactPath(inputs.outputDir, spotifySearchISRCFile),
			"search_metadata":       validationArtifactPath(inputs.outputDir, spotifySearchMetadataFile),
			"authenticated_summary": validationArtifactPath(inputs.outputDir, spotifyAuthenticatedReport),
		},
	}
}

func writeValidationArtifacts(outputDir string, artifacts validationArtifacts) error {
	sourcePayloadPath := filepath.Join(outputDir, spotifySourcePayloadFile)
	if err := validation.WritePrettyJSON(sourcePayloadPath, artifacts.albumBody); err != nil {
		return fmt.Errorf("write %s: %w", sourcePayloadPath, err)
	}
	searchUPCPath := filepath.Join(outputDir, spotifySearchUPCFile)
	if err := validation.WritePrettyJSON(searchUPCPath, artifacts.upcBody); err != nil {
		return fmt.Errorf("write %s: %w", searchUPCPath, err)
	}
	searchISRCPath := filepath.Join(outputDir, spotifySearchISRCFile)
	if err := validation.WritePrettyJSON(searchISRCPath, artifacts.isrcBody); err != nil {
		return fmt.Errorf("write %s: %w", searchISRCPath, err)
	}
	searchMetadataPath := filepath.Join(outputDir, spotifySearchMetadataFile)
	if err := validation.WritePrettyJSON(searchMetadataPath, artifacts.metadataBody); err != nil {
		return fmt.Errorf("write %s: %w", searchMetadataPath, err)
	}
	summaryPath := filepath.Join(outputDir, spotifyAuthenticatedReport)
	if err := validation.WriteJSON(summaryPath, artifacts.summary); err != nil {
		return fmt.Errorf("write %s: %w", summaryPath, err)
	}
	return nil
}

func validationArtifactPath(outputDir, name string) string {
	return filepath.ToSlash(filepath.Join(outputDir, name))
}
