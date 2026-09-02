package main

import (
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
		"artifacts":          buildValidationArtifactPaths(inputs.outputDir, artifacts),
	}
}

func buildValidationArtifactPaths(outputDir string, artifacts validationArtifacts) map[string]string {
	artifactPaths := map[string]string{
		"source_payload_official":  validationArtifactPath(outputDir, appleMusicSourcePayloadFile),
		"search_metadata_official": validationArtifactPath(outputDir, appleMusicSearchMetadataFile),
		"official_summary":         validationArtifactPath(outputDir, appleMusicOfficialSummaryFile),
	}
	addValidationArtifactPath(artifactPaths, "search_upc_official", outputDir, appleMusicSearchUPCFile, artifacts.upcBody)
	addValidationArtifactPath(artifactPaths, "search_isrc_official", outputDir, appleMusicSearchISRCFile, artifacts.isrcBody)
	return artifactPaths
}

func addValidationArtifactPath(paths map[string]string, key, outputDir, name string, body []byte) {
	if len(body) == 0 {
		return
	}
	paths[key] = validationArtifactPath(outputDir, name)
}

func writeValidationArtifacts(outputDir string, artifacts validationArtifacts) error {
	return validation.WriteArtifacts(outputDir, []validation.Artifact{
		{Name: appleMusicSourcePayloadFile, Body: artifacts.albumBody},
		{Name: appleMusicSearchMetadataFile, Body: artifacts.metadataBody},
		{Name: appleMusicSearchUPCFile, Body: artifacts.upcBody},
		{Name: appleMusicSearchISRCFile, Body: artifacts.isrcBody},
	}, appleMusicOfficialSummaryFile, artifacts.summary)
}

func validationArtifactPath(outputDir, name string) string {
	return validation.JoinArtifactPath(outputDir, name)
}
