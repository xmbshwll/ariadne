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
		} else {
			rows = append(rows, newCSVMatchRow(resolution.InputURL, service, "best", true, scoreSummary(result.Best.Score), *result.Best))
		}
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
		} else {
			rows = append(rows, newSongCSVMatchRow(resolution.InputURL, service, "best", true, scoreSummary(result.Best.Score), *result.Best))
		}
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
