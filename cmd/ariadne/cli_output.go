package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/xmbshwll/ariadne"
)

type cliResolution struct {
	InputURL string                    `json:"input_url"`
	Source   cliAlbum                  `json:"source"`
	Links    map[string]cliMatchResult `json:"links,omitempty"`
}

type cliAlbum struct {
	Service      string   `json:"service"`
	ID           string   `json:"id"`
	URL          string   `json:"url"`
	RegionHint   string   `json:"region_hint,omitempty"`
	Title        string   `json:"title"`
	Artists      []string `json:"artists"`
	ReleaseDate  string   `json:"release_date,omitempty"`
	Label        string   `json:"label,omitempty"`
	UPC          string   `json:"upc,omitempty"`
	TrackCount   int      `json:"track_count,omitempty"`
	ArtworkURL   string   `json:"artwork_url,omitempty"`
	EditionHints []string `json:"edition_hints,omitempty"`
}

type cliMatchResult struct {
	Found      bool       `json:"found"`
	Summary    string     `json:"summary"`
	Best       *cliMatch  `json:"best,omitempty"`
	Alternates []cliMatch `json:"alternates,omitempty"`
}

type cliMatch struct {
	URL         string   `json:"url"`
	Score       int      `json:"score"`
	Reasons     []string `json:"reasons,omitempty"`
	AlbumID     string   `json:"album_id,omitempty"`
	RegionHint  string   `json:"region_hint,omitempty"`
	Title       string   `json:"title,omitempty"`
	Artists     []string `json:"artists,omitempty"`
	ReleaseDate string   `json:"release_date,omitempty"`
	UPC         string   `json:"upc,omitempty"`
}

type cliSongResolution struct {
	InputURL string                        `json:"input_url"`
	Source   cliSong                       `json:"source"`
	Links    map[string]cliSongMatchResult `json:"links,omitempty"`
}

type cliSong struct {
	Service      string   `json:"service"`
	ID           string   `json:"id"`
	URL          string   `json:"url"`
	RegionHint   string   `json:"region_hint,omitempty"`
	Title        string   `json:"title"`
	Artists      []string `json:"artists"`
	DurationMS   int      `json:"duration_ms,omitempty"`
	ISRC         string   `json:"isrc,omitempty"`
	Explicit     bool     `json:"explicit,omitempty"`
	DiscNumber   int      `json:"disc_number,omitempty"`
	TrackNumber  int      `json:"track_number,omitempty"`
	AlbumID      string   `json:"album_id,omitempty"`
	AlbumTitle   string   `json:"album_title,omitempty"`
	ReleaseDate  string   `json:"release_date,omitempty"`
	ArtworkURL   string   `json:"artwork_url,omitempty"`
	EditionHints []string `json:"edition_hints,omitempty"`
}

type cliSongMatchResult struct {
	Found      bool           `json:"found"`
	Summary    string         `json:"summary"`
	Best       *cliSongMatch  `json:"best,omitempty"`
	Alternates []cliSongMatch `json:"alternates,omitempty"`
}

type cliSongMatch struct {
	URL         string   `json:"url"`
	Score       int      `json:"score"`
	Reasons     []string `json:"reasons,omitempty"`
	SongID      string   `json:"song_id,omitempty"`
	RegionHint  string   `json:"region_hint,omitempty"`
	Title       string   `json:"title,omitempty"`
	Artists     []string `json:"artists,omitempty"`
	DurationMS  int      `json:"duration_ms,omitempty"`
	ISRC        string   `json:"isrc,omitempty"`
	AlbumTitle  string   `json:"album_title,omitempty"`
	TrackNumber int      `json:"track_number,omitempty"`
	ReleaseDate string   `json:"release_date,omitempty"`
}

func newCLIOutput(resolution ariadne.Resolution, cfg resolveConfig) any {
	if cfg.verbose {
		return newCLIResolution(resolution)
	}
	return newCLILinks(resolution)
}

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

func newCLIResolution(resolution ariadne.Resolution) cliResolution {
	links := make(map[string]cliMatchResult, len(resolution.Matches))
	for service, match := range resolution.Matches {
		links[string(service)] = newCLIMatchResult(match)
	}

	return cliResolution{
		InputURL: resolution.InputURL,
		Source:   newCLIAlbum(resolution.Source),
		Links:    links,
	}
}

func newCLILinks(resolution ariadne.Resolution) map[string]string {
	links := map[string]string{}
	if resolution.Source.Service != "" && resolution.Source.SourceURL != "" {
		links[string(resolution.Source.Service)] = resolution.Source.SourceURL
	}
	for service, match := range resolution.Matches {
		if match.Best == nil || match.Best.URL == "" {
			continue
		}
		if _, exists := links[string(service)]; exists {
			continue
		}
		links[string(service)] = match.Best.URL
	}
	return links
}

func newCLISongResolution(resolution ariadne.SongResolution) cliSongResolution {
	links := make(map[string]cliSongMatchResult, len(resolution.Matches))
	for service, match := range resolution.Matches {
		links[string(service)] = newCLISongMatchResult(match)
	}

	return cliSongResolution{
		InputURL: resolution.InputURL,
		Source:   newCLISong(resolution.Source),
		Links:    links,
	}
}

func newCLISongLinks(resolution ariadne.SongResolution) map[string]string {
	links := map[string]string{}
	if resolution.Source.Service != "" && resolution.Source.SourceURL != "" {
		links[string(resolution.Source.Service)] = resolution.Source.SourceURL
	}
	for service, match := range resolution.Matches {
		if match.Best == nil || match.Best.URL == "" {
			continue
		}
		if _, exists := links[string(service)]; exists {
			continue
		}
		links[string(service)] = match.Best.URL
	}
	return links
}

func writeCompactCSV(w io.Writer, resolution ariadne.Resolution) error {
	writer := csv.NewWriter(w)
	if err := writer.Write([]string{"service", "url"}); err != nil {
		return fmt.Errorf("write csv header: %w", err)
	}
	links := newCLILinks(resolution)
	services := sortedKeys(links)
	for _, service := range services {
		if err := writer.Write([]string{service, links[service]}); err != nil {
			return fmt.Errorf("write csv row: %w", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush csv: %w", err)
	}
	return nil
}

func writeVerboseCSV(w io.Writer, resolution ariadne.Resolution) error {
	writer := csv.NewWriter(w)
	headers := []string{"input_url", "service", "kind", "url", "found", "summary", "score", "album_id", "region_hint", "title", "artists", "release_date", "upc", "reasons"}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("write csv header: %w", err)
	}
	for _, row := range newVerboseCSVRows(resolution) {
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("write csv row: %w", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush csv: %w", err)
	}
	return nil
}

func writeCompactSongCSV(w io.Writer, resolution ariadne.SongResolution) error {
	writer := csv.NewWriter(w)
	if err := writer.Write([]string{"service", "url"}); err != nil {
		return fmt.Errorf("write csv header: %w", err)
	}
	links := newCLISongLinks(resolution)
	services := sortedKeys(links)
	for _, service := range services {
		if err := writer.Write([]string{service, links[service]}); err != nil {
			return fmt.Errorf("write csv row: %w", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush csv: %w", err)
	}
	return nil
}

func writeVerboseSongCSV(w io.Writer, resolution ariadne.SongResolution) error {
	writer := csv.NewWriter(w)
	headers := []string{"input_url", "service", "kind", "url", "found", "summary", "score", "song_id", "region_hint", "title", "artists", "duration_ms", "isrc", "album_title", "track_number", "release_date", "reasons"}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("write csv header: %w", err)
	}
	for _, row := range newVerboseSongCSVRows(resolution) {
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("write csv row: %w", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush csv: %w", err)
	}
	return nil
}

func newVerboseCSVRows(resolution ariadne.Resolution) [][]string {
	rows := [][]string{{
		resolution.InputURL,
		string(resolution.Source.Service),
		"source",
		resolution.Source.SourceURL,
		"true",
		"source",
		"",
		resolution.Source.SourceID,
		resolution.Source.RegionHint,
		resolution.Source.Title,
		strings.Join(resolution.Source.Artists, " | "),
		resolution.Source.ReleaseDate,
		resolution.Source.UPC,
		"",
	}}

	services := make([]string, 0, len(resolution.Matches))
	for service := range resolution.Matches {
		services = append(services, string(service))
	}
	sort.Strings(services)
	for _, service := range services {
		result := resolution.Matches[ariadne.ServiceName(service)]
		if result.Best == nil {
			rows = append(rows, []string{
				resolution.InputURL,
				service,
				"best",
				"",
				"false",
				"not_found",
				"",
				"",
				"",
				"",
				"",
				"",
				"",
				"",
			})
			continue
		}
		rows = append(rows, newCSVMatchRow(resolution.InputURL, service, "best", true, scoreSummary(result.Best.Score), *result.Best))
		for _, alternate := range result.Alternates {
			rows = append(rows, newCSVMatchRow(resolution.InputURL, service, "alternate", true, scoreSummary(alternate.Score), alternate))
		}
	}
	return rows
}

func newCSVMatchRow(inputURL, service, kind string, found bool, summary string, match ariadne.ScoredMatch) []string {
	return []string{
		inputURL,
		service,
		kind,
		match.URL,
		strconv.FormatBool(found),
		summary,
		strconv.Itoa(match.Score),
		match.Candidate.CandidateID,
		match.Candidate.RegionHint,
		match.Candidate.Title,
		strings.Join(match.Candidate.Artists, " | "),
		match.Candidate.ReleaseDate,
		match.Candidate.UPC,
		strings.Join(match.Reasons, " | "),
	}
}

func newVerboseSongCSVRows(resolution ariadne.SongResolution) [][]string {
	rows := [][]string{{
		resolution.InputURL,
		string(resolution.Source.Service),
		"source",
		resolution.Source.SourceURL,
		"true",
		"source",
		"",
		resolution.Source.SourceID,
		resolution.Source.RegionHint,
		resolution.Source.Title,
		strings.Join(resolution.Source.Artists, " | "),
		strconv.Itoa(resolution.Source.DurationMS),
		resolution.Source.ISRC,
		resolution.Source.AlbumTitle,
		strconv.Itoa(resolution.Source.TrackNumber),
		resolution.Source.ReleaseDate,
		"",
	}}

	services := make([]string, 0, len(resolution.Matches))
	for service := range resolution.Matches {
		services = append(services, string(service))
	}
	sort.Strings(services)
	for _, service := range services {
		result := resolution.Matches[ariadne.ServiceName(service)]
		if result.Best == nil {
			rows = append(rows, []string{
				resolution.InputURL,
				service,
				"best",
				"",
				"false",
				"not_found",
				"",
				"",
				"",
				"",
				"",
				"",
				"",
				"",
				"",
				"",
				"",
			})
			continue
		}
		rows = append(rows, newSongCSVMatchRow(resolution.InputURL, service, "best", true, scoreSummary(result.Best.Score), *result.Best))
		for _, alternate := range result.Alternates {
			rows = append(rows, newSongCSVMatchRow(resolution.InputURL, service, "alternate", true, scoreSummary(alternate.Score), alternate))
		}
	}
	return rows
}

func newSongCSVMatchRow(inputURL, service, kind string, found bool, summary string, match ariadne.SongScoredMatch) []string {
	return []string{
		inputURL,
		service,
		kind,
		match.URL,
		strconv.FormatBool(found),
		summary,
		strconv.Itoa(match.Score),
		match.Candidate.CandidateID,
		match.Candidate.RegionHint,
		match.Candidate.Title,
		strings.Join(match.Candidate.Artists, " | "),
		strconv.Itoa(match.Candidate.DurationMS),
		match.Candidate.ISRC,
		match.Candidate.AlbumTitle,
		strconv.Itoa(match.Candidate.TrackNumber),
		match.Candidate.ReleaseDate,
		strings.Join(match.Reasons, " | "),
	}
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func filterResolutionByStrength(resolution ariadne.Resolution, minStrength ariadne.MatchStrength) ariadne.Resolution {
	if minStrength == ariadne.MatchStrengthVeryWeak {
		return resolution
	}
	filtered := resolution
	filtered.Matches = make(map[ariadne.ServiceName]ariadne.MatchResult, len(resolution.Matches))
	for service, match := range resolution.Matches {
		pruned, ok := pruneAlbumMatchByStrength(match, minStrength)
		if !ok {
			continue
		}
		filtered.Matches[service] = pruned
	}
	return filtered
}

func pruneAlbumMatchByStrength(match ariadne.MatchResult, minStrength ariadne.MatchStrength) (ariadne.MatchResult, bool) {
	pruned := match
	pruned.Alternates = filterAlternatesByStrength(match.Alternates, minStrength)

	if match.Best == nil || !meetsMinimumStrength(match.Best.Score, minStrength) {
		return ariadne.MatchResult{}, false
	}
	return pruned, true
}

func filterAlternatesByStrength(alternates []ariadne.ScoredMatch, minStrength ariadne.MatchStrength) []ariadne.ScoredMatch {
	filtered := make([]ariadne.ScoredMatch, 0, len(alternates))
	for _, alternate := range alternates {
		if !meetsMinimumStrength(alternate.Score, minStrength) {
			continue
		}
		filtered = append(filtered, alternate)
	}
	return filtered
}

func filterSongResolutionByStrength(resolution ariadne.SongResolution, minStrength ariadne.MatchStrength) ariadne.SongResolution {
	if minStrength == ariadne.MatchStrengthVeryWeak {
		return resolution
	}

	filtered := resolution
	filtered.Matches = make(map[ariadne.ServiceName]ariadne.SongMatchResult, len(resolution.Matches))
	for service, match := range resolution.Matches {
		pruned, ok := pruneSongMatchByStrength(match, minStrength)
		if !ok {
			continue
		}
		filtered.Matches[service] = pruned
	}
	return filtered
}

func pruneSongMatchByStrength(match ariadne.SongMatchResult, minStrength ariadne.MatchStrength) (ariadne.SongMatchResult, bool) {
	pruned := match
	pruned.Alternates = filterSongAlternatesByStrength(match.Alternates, minStrength)

	if match.Best != nil && meetsMinimumStrength(match.Best.Score, minStrength) {
		best := *match.Best
		pruned.Best = &best
		return pruned, true
	}

	// Songs intentionally keep the service when strong alternates remain, even if
	// the original Best candidate falls below the threshold. Album output is
	// stricter and drops the whole service when Best is pruned.
	pruned.Best = nil
	if len(pruned.Alternates) == 0 {
		return ariadne.SongMatchResult{}, false
	}
	return pruned, true
}

func filterSongAlternatesByStrength(alternates []ariadne.SongScoredMatch, minStrength ariadne.MatchStrength) []ariadne.SongScoredMatch {
	filtered := make([]ariadne.SongScoredMatch, 0, len(alternates))
	for _, alternate := range alternates {
		if !meetsMinimumStrength(alternate.Score, minStrength) {
			continue
		}
		filtered = append(filtered, alternate)
	}
	return filtered
}

func meetsMinimumStrength(score int, minStrength ariadne.MatchStrength) bool {
	return matchStrengthRank(ariadne.MatchStrengthForScore(score)) >= matchStrengthRank(minStrength)
}

func matchStrengthRank(strength ariadne.MatchStrength) int {
	switch strength {
	case ariadne.MatchStrengthStrong:
		return 3
	case ariadne.MatchStrengthProbable:
		return 2
	case ariadne.MatchStrengthWeak:
		return 1
	default:
		return 0
	}
}

func newCLIAlbum(album ariadne.CanonicalAlbum) cliAlbum {
	return cliAlbum{
		Service:      string(album.Service),
		ID:           album.SourceID,
		URL:          album.SourceURL,
		RegionHint:   album.RegionHint,
		Title:        album.Title,
		Artists:      append([]string(nil), album.Artists...),
		ReleaseDate:  album.ReleaseDate,
		Label:        album.Label,
		UPC:          album.UPC,
		TrackCount:   album.TrackCount,
		ArtworkURL:   album.ArtworkURL,
		EditionHints: append([]string(nil), album.EditionHints...),
	}
}

func newCLISong(song ariadne.CanonicalSong) cliSong {
	return cliSong{
		Service:      string(song.Service),
		ID:           song.SourceID,
		URL:          song.SourceURL,
		RegionHint:   song.RegionHint,
		Title:        song.Title,
		Artists:      append([]string(nil), song.Artists...),
		DurationMS:   song.DurationMS,
		ISRC:         song.ISRC,
		Explicit:     song.Explicit,
		DiscNumber:   song.DiscNumber,
		TrackNumber:  song.TrackNumber,
		AlbumID:      song.AlbumID,
		AlbumTitle:   song.AlbumTitle,
		ReleaseDate:  song.ReleaseDate,
		ArtworkURL:   song.ArtworkURL,
		EditionHints: append([]string(nil), song.EditionHints...),
	}
}

func newCLIMatchResult(result ariadne.MatchResult) cliMatchResult {
	output := cliMatchResult{
		Found:      result.Best != nil,
		Summary:    "not_found",
		Alternates: make([]cliMatch, 0, len(result.Alternates)),
	}
	if result.Best != nil {
		best := newCLIMatch(*result.Best)
		output.Best = &best
		output.Summary = scoreSummary(result.Best.Score)
	}
	for _, alternate := range result.Alternates {
		output.Alternates = append(output.Alternates, newCLIMatch(alternate))
	}
	return output
}

func newCLISongMatchResult(result ariadne.SongMatchResult) cliSongMatchResult {
	output := cliSongMatchResult{
		Found:      result.Best != nil,
		Summary:    "not_found",
		Alternates: make([]cliSongMatch, 0, len(result.Alternates)),
	}
	if result.Best != nil {
		best := newCLISongMatch(*result.Best)
		output.Best = &best
		output.Summary = scoreSummary(result.Best.Score)
	}
	for _, alternate := range result.Alternates {
		output.Alternates = append(output.Alternates, newCLISongMatch(alternate))
	}
	return output
}

func scoreSummary(score int) string {
	return string(ariadne.MatchStrengthForScore(score))
}

func newCLIMatch(match ariadne.ScoredMatch) cliMatch {
	return cliMatch{
		URL:         match.URL,
		Score:       match.Score,
		Reasons:     append([]string(nil), match.Reasons...),
		AlbumID:     match.Candidate.CandidateID,
		RegionHint:  match.Candidate.RegionHint,
		Title:       match.Candidate.Title,
		Artists:     append([]string(nil), match.Candidate.Artists...),
		ReleaseDate: match.Candidate.ReleaseDate,
		UPC:         match.Candidate.UPC,
	}
}

func newCLISongMatch(match ariadne.SongScoredMatch) cliSongMatch {
	return cliSongMatch{
		URL:         match.URL,
		Score:       match.Score,
		Reasons:     append([]string(nil), match.Reasons...),
		SongID:      match.Candidate.CandidateID,
		RegionHint:  match.Candidate.RegionHint,
		Title:       match.Candidate.Title,
		Artists:     append([]string(nil), match.Candidate.Artists...),
		DurationMS:  match.Candidate.DurationMS,
		ISRC:        match.Candidate.ISRC,
		AlbumTitle:  match.Candidate.AlbumTitle,
		TrackNumber: match.Candidate.TrackNumber,
		ReleaseDate: match.Candidate.ReleaseDate,
	}
}
