package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/xmbshwll/ariadne"
)

func writeCompactCSV(w io.Writer, resolution ariadne.Resolution) error {
	return writeCSVRows(w, []string{"service", "url"}, linkRows(newCLILinks(resolution)))
}

func writeVerboseCSV(w io.Writer, resolution ariadne.Resolution) error {
	headers := []string{"input_url", "service", "kind", "url", "found", "summary", "score", "album_id", "region_hint", "title", "artists", "release_date", "upc", "reasons"}
	return writeCSVRows(w, headers, newVerboseCSVRows(resolution))
}

func writeCompactSongCSV(w io.Writer, resolution ariadne.SongResolution) error {
	return writeCSVRows(w, []string{"service", "url"}, linkRows(newCLISongLinks(resolution)))
}

func writeVerboseSongCSV(w io.Writer, resolution ariadne.SongResolution) error {
	headers := []string{"input_url", "service", "kind", "url", "found", "summary", "score", "song_id", "region_hint", "title", "artists", "duration_ms", "isrc", "album_title", "track_number", "release_date", "reasons"}
	return writeCSVRows(w, headers, newVerboseSongCSVRows(resolution))
}

func linkRows(links map[string]string) [][]string {
	rows := make([][]string, 0, len(links))
	for _, service := range sortedKeys(links) {
		rows = append(rows, []string{service, links[service]})
	}
	return rows
}

func writeCSVRows(w io.Writer, header []string, rows [][]string) error {
	writer := csv.NewWriter(w)
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("write csv header: %w", err)
	}
	for _, row := range rows {
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

	return appendMatchRows(rows, resolution.InputURL, 7, resolution.Matches, newCSVMatchRow)
}

// appendMatchRows appends one not_found row or best+alternate rows per service,
// in sorted service order. Row shape stays per entity shape via matchRow;
// trailingEmpty pads not_found rows to the entity's column count.
func appendMatchRows[C any](
	rows [][]string,
	inputURL string,
	trailingEmpty int,
	matches map[ariadne.ServiceName]ariadne.MatchResultOf[C],
	matchRow func(inputURL, service, kind string, found bool, summary string, match ariadne.ScoredMatchOf[C]) []string,
) [][]string {
	services := make([]string, 0, len(matches))
	for service := range matches {
		services = append(services, string(service))
	}
	sort.Strings(services)
	for _, service := range services {
		result := matches[ariadne.ServiceName(service)]
		if result.Best == nil {
			row := []string{inputURL, service, "best", "", "false", "not_found", ""}
			rows = append(rows, append(row, make([]string, trailingEmpty)...))
		} else {
			rows = append(rows, matchRow(inputURL, service, "best", true, scoreSummary(result.Best.Score), *result.Best))
		}
		for _, alternate := range result.Alternates {
			rows = append(rows, matchRow(inputURL, service, "alternate", true, scoreSummary(alternate.Score), alternate))
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

	return appendMatchRows(rows, resolution.InputURL, 10, resolution.Matches, newSongCSVMatchRow)
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
