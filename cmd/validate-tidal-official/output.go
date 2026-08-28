package main

import (
	"sort"
	"time"

	"github.com/xmbshwll/ariadne/cmd/internal/validation"
)

// tidalOfficialSummaryFile is the summary artifact the TIDAL tool writes last.
const tidalOfficialSummaryFile = "official-summary.json"

func buildValidationSummary(inputs validationInputs, title string, artistNames []string, releaseDate string, upc string, trackTitles []string, trackISRCs []string) map[string]any {
	artifacts := map[string]string{
		"source_payload_official": validation.JoinArtifactPath(inputs.outputDir, "source-payload-official.json"),
		"search_albums_official":  validation.JoinArtifactPath(inputs.outputDir, "search-albums-official.json"),
		"official_summary":        validation.JoinArtifactPath(inputs.outputDir, "official-summary.json"),
	}
	if upc != "" {
		artifacts["search_upc_official"] = validation.JoinArtifactPath(inputs.outputDir, "search-upc-official.json")
	}
	if len(trackISRCs) > 0 {
		artifacts["search_isrc_official"] = validation.JoinArtifactPath(inputs.outputDir, "search-isrc-official.json")
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
	targets := make([]validation.Artifact, 0, len(artifacts.targets))
	for _, name := range sortedArtifactNames(artifacts.targets) {
		targets = append(targets, validation.Artifact{Name: name, Body: artifacts.targets[name]})
	}
	return validation.WriteArtifacts(outputDir, targets, tidalOfficialSummaryFile, artifacts.summary)
}

// sortedArtifactNames keeps artifact write order deterministic; the summary
// references the paths, so the same run must always write in the same order.
func sortedArtifactNames(targets map[string][]byte) []string {
	names := make([]string, 0, len(targets))
	for name := range targets {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
