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

func writeCompactSongCSV(w io.Writer, resolution ariadne.SongResolution) error {
	return writeCSVRows(w, []string{"service", "url"}, linkRows(newCLISongLinks(resolution)))
}

func writeVerboseCSV(w io.Writer, resolution ariadne.Resolution) error {
	return writeCSVRows(w, columnHeaders(albumCSVColumns), verboseCSVRows(albumCSVColumns, resolution))
}

func writeVerboseSongCSV(w io.Writer, resolution ariadne.SongResolution) error {
	return writeCSVRows(w, columnHeaders(songCSVColumns), verboseSongCSVRows(songCSVColumns, resolution))
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

// csvColumn is one verbose-CSV column: its header and the value it renders
// from a row's fields. The header list and the rendered values come from the
// same table, so the column order cannot drift between them.
type csvColumn[Row any] struct {
	header string
	value  func(Row) string
}

func columnHeaders[Row any](columns []csvColumn[Row]) []string {
	headers := make([]string, 0, len(columns))
	for _, column := range columns {
		headers = append(headers, column.header)
	}
	return headers
}

// renderRow projects one row through every column, in table order.
func renderRow[Row any](columns []csvColumn[Row], row Row) []string {
	values := make([]string, 0, len(columns))
	for _, column := range columns {
		values = append(values, column.value(row))
	}
	return values
}

// verboseRow carries the fields every verbose CSV row shares, whatever the
// entity shape: the positional prefix, then the shape's own trailing columns.
type verboseRow struct {
	inputURL string
	service  string
	kind     string
	url      string
	found    bool
	summary  string
	score    string
	// row holds the entity-shape-specific trailing columns.
	row any
}

// albumCSVColumns is the album verbose CSV shape: the shared prefix columns,
// then the album's own fields. Headers and values stay in one table.
var albumCSVColumns = []csvColumn[verboseRow]{
	{header: "input_url", value: func(r verboseRow) string { return r.inputURL }},
	{header: "service", value: func(r verboseRow) string { return r.service }},
	{header: "kind", value: func(r verboseRow) string { return r.kind }},
	{header: "url", value: func(r verboseRow) string { return r.url }},
	{header: "found", value: func(r verboseRow) string { return strconv.FormatBool(r.found) }},
	{header: "summary", value: func(r verboseRow) string { return r.summary }},
	{header: "score", value: func(r verboseRow) string { return r.score }},
	{header: "album_id", value: func(r verboseRow) string { return albumFields(r.row).candidateID }},
	{header: "region_hint", value: func(r verboseRow) string { return albumFields(r.row).regionHint }},
	{header: "title", value: func(r verboseRow) string { return albumFields(r.row).title }},
	{header: "artists", value: func(r verboseRow) string { return albumFields(r.row).artists }},
	{header: "release_date", value: func(r verboseRow) string { return albumFields(r.row).releaseDate }},
	{header: "upc", value: func(r verboseRow) string { return albumFields(r.row).uniqueID }},
	{header: "reasons", value: func(r verboseRow) string { return albumFields(r.row).reasons }},
}

// songCSVColumns is the song verbose CSV shape.
var songCSVColumns = []csvColumn[verboseRow]{
	{header: "input_url", value: func(r verboseRow) string { return r.inputURL }},
	{header: "service", value: func(r verboseRow) string { return r.service }},
	{header: "kind", value: func(r verboseRow) string { return r.kind }},
	{header: "url", value: func(r verboseRow) string { return r.url }},
	{header: "found", value: func(r verboseRow) string { return strconv.FormatBool(r.found) }},
	{header: "summary", value: func(r verboseRow) string { return r.summary }},
	{header: "score", value: func(r verboseRow) string { return r.score }},
	{header: "song_id", value: func(r verboseRow) string { return songFields(r.row).candidateID }},
	{header: "region_hint", value: func(r verboseRow) string { return songFields(r.row).regionHint }},
	{header: "title", value: func(r verboseRow) string { return songFields(r.row).title }},
	{header: "artists", value: func(r verboseRow) string { return songFields(r.row).artists }},
	{header: "duration_ms", value: func(r verboseRow) string { return songFields(r.row).durationMS }},
	{header: "isrc", value: func(r verboseRow) string { return songFields(r.row).isrc }},
	{header: "album_title", value: func(r verboseRow) string { return songFields(r.row).albumTitle }},
	{header: "track_number", value: func(r verboseRow) string { return songFields(r.row).trackNumber }},
	{header: "release_date", value: func(r verboseRow) string { return songFields(r.row).releaseDate }},
	{header: "reasons", value: func(r verboseRow) string { return songFields(r.row).reasons }},
}

// albumShape adapts the album's source and candidate payloads to the shared
// trailing-column names, so one table serves both the source row and match rows.
type csvShape struct {
	candidateID string
	regionHint  string
	title       string
	artists     string
	albumTitle  string
	durationMS  string
	isrc        string
	trackNumber string
	releaseDate string
	uniqueID    string
	reasons     string
}

func albumFields(payload any) csvShape {
	switch payload := payload.(type) {
	case ariadne.CanonicalAlbum:
		return csvShape{
			candidateID: payload.SourceID,
			regionHint:  payload.RegionHint,
			title:       payload.Title,
			artists:     strings.Join(payload.Artists, " | "),
			releaseDate: payload.ReleaseDate,
			uniqueID:    payload.UPC,
		}
	case ariadne.ScoredMatch:
		return csvShape{
			candidateID: payload.Candidate.CandidateID,
			regionHint:  payload.Candidate.RegionHint,
			title:       payload.Candidate.Title,
			artists:     strings.Join(payload.Candidate.Artists, " | "),
			releaseDate: payload.Candidate.ReleaseDate,
			uniqueID:    payload.Candidate.UPC,
			reasons:     strings.Join(payload.Reasons, " | "),
		}
	}
	return csvShape{}
}

func songFields(payload any) csvShape {
	switch payload := payload.(type) {
	case ariadne.CanonicalSong:
		return csvShape{
			candidateID: payload.SourceID,
			regionHint:  payload.RegionHint,
			title:       payload.Title,
			artists:     strings.Join(payload.Artists, " | "),
			albumTitle:  payload.AlbumTitle,
			durationMS:  strconv.Itoa(payload.DurationMS),
			isrc:        payload.ISRC,
			trackNumber: strconv.Itoa(payload.TrackNumber),
			releaseDate: payload.ReleaseDate,
		}
	case ariadne.SongScoredMatch:
		return csvShape{
			candidateID: payload.Candidate.CandidateID,
			regionHint:  payload.Candidate.RegionHint,
			title:       payload.Candidate.Title,
			artists:     strings.Join(payload.Candidate.Artists, " | "),
			albumTitle:  payload.Candidate.AlbumTitle,
			durationMS:  strconv.Itoa(payload.Candidate.DurationMS),
			isrc:        payload.Candidate.ISRC,
			trackNumber: strconv.Itoa(payload.Candidate.TrackNumber),
			releaseDate: payload.Candidate.ReleaseDate,
			reasons:     strings.Join(payload.Reasons, " | "),
		}
	}
	return csvShape{}
}

func verboseCSVRows(columns []csvColumn[verboseRow], resolution ariadne.Resolution) [][]string {
	return appendSourceAndMatchRows(columns, verboseRow{
		inputURL: resolution.InputURL,
		service:  string(resolution.Source.Service),
		kind:     "source",
		url:      resolution.Source.SourceURL,
		found:    true,
		summary:  "source",
		row:      resolution.Source,
	}, resolution.InputURL, resolution.Matches)
}

func verboseSongCSVRows(columns []csvColumn[verboseRow], resolution ariadne.SongResolution) [][]string {
	return appendSourceAndMatchRows(columns, verboseRow{
		inputURL: resolution.InputURL,
		service:  string(resolution.Source.Service),
		kind:     "source",
		url:      resolution.Source.SourceURL,
		found:    true,
		summary:  "source",
		row:      resolution.Source,
	}, resolution.InputURL, resolution.Matches)
}

// appendSourceAndMatchRows renders the source row then one row per best and
// alternate match, in sorted service order.
func appendSourceAndMatchRows[C any](
	columns []csvColumn[verboseRow],
	sourceRow verboseRow,
	inputURL string,
	matches map[ariadne.ServiceName]ariadne.MatchResultOf[C],
) [][]string {
	rows := [][]string{renderRow(columns, sourceRow)}
	return appendMatchRows(rows, columns, inputURL, matches)
}

// appendMatchRows appends one not_found row or best+alternate rows per service,
// in sorted service order. The column table renders every row, so not_found
// rows get the entity's columns for free.
func appendMatchRows[C any](
	rows [][]string,
	columns []csvColumn[verboseRow],
	inputURL string,
	matches map[ariadne.ServiceName]ariadne.MatchResultOf[C],
) [][]string {
	services := make([]string, 0, len(matches))
	for service := range matches {
		services = append(services, string(service))
	}
	sort.Strings(services)
	for _, service := range services {
		result := matches[ariadne.ServiceName(service)]
		if result.Best != nil {
			rows = append(rows, renderRow(columns, matchVerboseRow(inputURL, service, "best", *result.Best)))
		} else {
			empty := verboseRow{inputURL: inputURL, service: service, kind: "best"}
			rows = append(rows, renderRow(columns, empty))
		}
		// Alternates render even without a best, matching the long-standing
		// output contract that a failing best must not hide ranked alternates.
		for _, alternate := range result.Alternates {
			rows = append(rows, renderRow(columns, matchVerboseRow(inputURL, service, "alternate", alternate)))
		}
	}
	return rows
}

func matchVerboseRow[C any](inputURL, service, kind string, match ariadne.ScoredMatchOf[C]) verboseRow {
	return verboseRow{
		inputURL: inputURL,
		service:  service,
		kind:     kind,
		url:      match.URL,
		found:    true,
		summary:  scoreSummary(match.Score),
		score:    strconv.Itoa(match.Score),
		row:      match,
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
