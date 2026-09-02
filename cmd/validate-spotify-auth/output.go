package main

import (
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
		"artifacts":          buildValidationArtifactPaths(inputs.outputDir),
	}
}

func buildValidationArtifactPaths(outputDir string) map[string]string {
	return map[string]string{
		"source_payload_api":    validationArtifactPath(outputDir, spotifySourcePayloadFile),
		"search_upc_results":    validationArtifactPath(outputDir, spotifySearchUPCFile),
		"search_isrc_results":   validationArtifactPath(outputDir, spotifySearchISRCFile),
		"search_metadata":       validationArtifactPath(outputDir, spotifySearchMetadataFile),
		"authenticated_summary": validationArtifactPath(outputDir, spotifyAuthenticatedReport),
	}
}

func writeValidationArtifacts(outputDir string, artifacts validationArtifacts) error {
	return validation.WriteArtifacts(outputDir, []validation.Artifact{
		{Name: spotifySourcePayloadFile, Body: artifacts.albumBody},
		{Name: spotifySearchUPCFile, Body: artifacts.upcBody},
		{Name: spotifySearchISRCFile, Body: artifacts.isrcBody},
		{Name: spotifySearchMetadataFile, Body: artifacts.metadataBody},
	}, spotifyAuthenticatedReport, artifacts.summary)
}

func validationArtifactPath(outputDir, name string) string {
	return filepath.ToSlash(filepath.Join(outputDir, name))
}
