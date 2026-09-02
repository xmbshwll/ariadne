package main

// CLI output: result DTOs, field mapping, strength filtering, CSV and JSON/YAML rendering.

// CLI output: result DTOs, field mapping, strength filtering, CSV and JSON/YAML rendering.

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/xmbshwll/ariadne"
	"gopkg.in/yaml.v3"
)

type cliResolution struct {
	InputURL string                    `json:"input_url" yaml:"input_url"`
	Source   cliAlbum                  `json:"source" yaml:"source"`
	Links    map[string]cliMatchResult `json:"links,omitempty" yaml:"links,omitempty"`
}

type cliAlbum struct {
	Service      string   `json:"service" yaml:"service"`
	ID           string   `json:"id" yaml:"id"`
	URL          string   `json:"url" yaml:"url"`
	RegionHint   string   `json:"region_hint,omitempty" yaml:"region_hint,omitempty"`
	Title        string   `json:"title" yaml:"title"`
	Artists      []string `json:"artists" yaml:"artists"`
	ReleaseDate  string   `json:"release_date,omitempty" yaml:"release_date,omitempty"`
	Label        string   `json:"label,omitempty" yaml:"label,omitempty"`
	UPC          string   `json:"upc,omitempty" yaml:"upc,omitempty"`
	TrackCount   int      `json:"track_count,omitempty" yaml:"track_count,omitempty"`
	ArtworkURL   string   `json:"artwork_url,omitempty" yaml:"artwork_url,omitempty"`
	EditionHints []string `json:"edition_hints,omitempty" yaml:"edition_hints,omitempty"`
}

type cliMatchListing[M any] struct {
	Found      bool   `json:"found" yaml:"found"`
	Summary    string `json:"summary" yaml:"summary"`
	Best       *M     `json:"best,omitempty" yaml:"best,omitempty"`
	Alternates []M    `json:"alternates,omitempty" yaml:"alternates,omitempty"`
}

type cliMatchResult = cliMatchListing[cliMatch]

type cliMatch struct {
	URL         string   `json:"url" yaml:"url"`
	Score       int      `json:"score" yaml:"score"`
	Reasons     []string `json:"reasons,omitempty" yaml:"reasons,omitempty"`
	AlbumID     string   `json:"album_id,omitempty" yaml:"album_id,omitempty"`
	RegionHint  string   `json:"region_hint,omitempty" yaml:"region_hint,omitempty"`
	Title       string   `json:"title,omitempty" yaml:"title,omitempty"`
	Artists     []string `json:"artists,omitempty" yaml:"artists,omitempty"`
	ReleaseDate string   `json:"release_date,omitempty" yaml:"release_date,omitempty"`
	UPC         string   `json:"upc,omitempty" yaml:"upc,omitempty"`
}

type cliSongResolution struct {
	InputURL string                        `json:"input_url" yaml:"input_url"`
	Source   cliSong                       `json:"source" yaml:"source"`
	Links    map[string]cliSongMatchResult `json:"links,omitempty" yaml:"links,omitempty"`
}

type cliSong struct {
	Service      string   `json:"service" yaml:"service"`
	ID           string   `json:"id" yaml:"id"`
	URL          string   `json:"url" yaml:"url"`
	RegionHint   string   `json:"region_hint,omitempty" yaml:"region_hint,omitempty"`
	Title        string   `json:"title" yaml:"title"`
	Artists      []string `json:"artists" yaml:"artists"`
	DurationMS   int      `json:"duration_ms,omitempty" yaml:"duration_ms,omitempty"`
	ISRC         string   `json:"isrc,omitempty" yaml:"isrc,omitempty"`
	Explicit     bool     `json:"explicit,omitempty" yaml:"explicit,omitempty"`
	DiscNumber   int      `json:"disc_number,omitempty" yaml:"disc_number,omitempty"`
	TrackNumber  int      `json:"track_number,omitempty" yaml:"track_number,omitempty"`
	AlbumID      string   `json:"album_id,omitempty" yaml:"album_id,omitempty"`
	AlbumTitle   string   `json:"album_title,omitempty" yaml:"album_title,omitempty"`
	ReleaseDate  string   `json:"release_date,omitempty" yaml:"release_date,omitempty"`
	ArtworkURL   string   `json:"artwork_url,omitempty" yaml:"artwork_url,omitempty"`
	EditionHints []string `json:"edition_hints,omitempty" yaml:"edition_hints,omitempty"`
}

type cliSongMatchResult = cliMatchListing[cliSongMatch]

type cliSongMatch struct {
	URL         string   `json:"url" yaml:"url"`
	Score       int      `json:"score" yaml:"score"`
	Reasons     []string `json:"reasons,omitempty" yaml:"reasons,omitempty"`
	SongID      string   `json:"song_id,omitempty" yaml:"song_id,omitempty"`
	RegionHint  string   `json:"region_hint,omitempty" yaml:"region_hint,omitempty"`
	Title       string   `json:"title,omitempty" yaml:"title,omitempty"`
	Artists     []string `json:"artists,omitempty" yaml:"artists,omitempty"`
	DurationMS  int      `json:"duration_ms,omitempty" yaml:"duration_ms,omitempty"`
	ISRC        string   `json:"isrc,omitempty" yaml:"isrc,omitempty"`
	AlbumTitle  string   `json:"album_title,omitempty" yaml:"album_title,omitempty"`
	TrackNumber int      `json:"track_number,omitempty" yaml:"track_number,omitempty"`
	ReleaseDate string   `json:"release_date,omitempty" yaml:"release_date,omitempty"`
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
	return newCLIMatchListing(result, newCLIMatch)
}

func newCLISongMatchResult(result ariadne.SongMatchResult) cliSongMatchResult {
	return newCLIMatchListing(result, newCLISongMatch)
}

func newCLIMatchListing[C any, M any](result ariadne.MatchResultOf[C], convert func(ariadne.ScoredMatchOf[C]) M) cliMatchListing[M] {
	output := cliMatchListing[M]{
		Found:      result.Best != nil,
		Summary:    "not_found",
		Alternates: make([]M, 0, len(result.Alternates)),
	}
	if result.Best != nil {
		best := convert(*result.Best)
		output.Best = &best
		output.Summary = scoreSummary(result.Best.Score)
	}
	for _, alternate := range result.Alternates {
		output.Alternates = append(output.Alternates, convert(alternate))
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
	return newMatchLinks(string(resolution.Source.Service), resolution.Source.SourceURL, resolution.Matches)
}

func newMatchLinks[C any](sourceService string, sourceURL string, matches map[ariadne.ServiceName]ariadne.MatchResultOf[C]) map[string]string {
	links := map[string]string{}
	if sourceService != "" && sourceURL != "" {
		links[sourceService] = sourceURL
	}
	for service, match := range matches {
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
	return newMatchLinks(string(resolution.Source.Service), resolution.Source.SourceURL, resolution.Matches)
}

func filterResolutionByStrength(resolution ariadne.Resolution, minStrength ariadne.MatchStrength) ariadne.Resolution {
	filtered := resolution
	filtered.Matches = filterMatchesByStrength(resolution.Matches, minStrength, pruneAlbumMatchByStrength)
	return filtered
}

func filterSongResolutionByStrength(resolution ariadne.SongResolution, minStrength ariadne.MatchStrength) ariadne.SongResolution {
	filtered := resolution
	filtered.Matches = filterMatchesByStrength(resolution.Matches, minStrength, pruneSongMatchByStrength)
	return filtered
}

func filterMatchesByStrength[C any](
	matches map[ariadne.ServiceName]ariadne.MatchResultOf[C],
	minStrength ariadne.MatchStrength,
	prune func(ariadne.MatchResultOf[C], ariadne.MatchStrength) (ariadne.MatchResultOf[C], bool),
) map[ariadne.ServiceName]ariadne.MatchResultOf[C] {
	if minStrength == ariadne.MatchStrengthVeryWeak {
		return matches
	}
	filtered := make(map[ariadne.ServiceName]ariadne.MatchResultOf[C], len(matches))
	for service, match := range matches {
		pruned, ok := prune(match, minStrength)
		if !ok {
			continue
		}
		filtered[service] = pruned
	}
	return filtered
}

// pruneMatchByStrength filters alternates below the threshold, then applies the
// entity shape's keep policy for a best that no longer qualifies: album output
// drops the whole service, while songs keep the service when strong alternates
// remain by promoting the best alternate into the best slot.
func pruneMatchByStrength[C any](match ariadne.MatchResultOf[C], minStrength ariadne.MatchStrength, promoteAlternate bool) (ariadne.MatchResultOf[C], bool) {
	pruned := match
	pruned.Alternates = filterScoredByStrength(match.Alternates, minStrength)

	if match.Best != nil && meetsMinimumStrength(match.Best.Score, minStrength) {
		best := *match.Best
		pruned.Best = &best
		return pruned, true
	}
	if !promoteAlternate || len(pruned.Alternates) == 0 {
		return ariadne.MatchResultOf[C]{}, false
	}

	best, alternates := promoteBestAlternate(pruned.Alternates)
	pruned.Best = &best
	pruned.Alternates = alternates
	return pruned, true
}

func pruneAlbumMatchByStrength(match ariadne.MatchResult, minStrength ariadne.MatchStrength) (ariadne.MatchResult, bool) {
	return pruneMatchByStrength(match, minStrength, false)
}

func pruneSongMatchByStrength(match ariadne.SongMatchResult, minStrength ariadne.MatchStrength) (ariadne.SongMatchResult, bool) {
	return pruneMatchByStrength(match, minStrength, true)
}

// promoteBestAlternate requires at least one entry; callers check beforehand.
func promoteBestAlternate[C any](alternates []ariadne.ScoredMatchOf[C]) (ariadne.ScoredMatchOf[C], []ariadne.ScoredMatchOf[C]) {
	bestIndex := 0
	for i := 1; i < len(alternates); i++ {
		if alternates[i].Score > alternates[bestIndex].Score {
			bestIndex = i
		}
	}

	best := alternates[bestIndex]
	remaining := make([]ariadne.ScoredMatchOf[C], 0, len(alternates)-1)
	remaining = append(remaining, alternates[:bestIndex]...)
	remaining = append(remaining, alternates[bestIndex+1:]...)
	return best, remaining
}

func filterScoredByStrength[C any](alternates []ariadne.ScoredMatchOf[C], minStrength ariadne.MatchStrength) []ariadne.ScoredMatchOf[C] {
	filtered := make([]ariadne.ScoredMatchOf[C], 0, len(alternates))
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

// resolutionOutput[DTO] is the per-Entity-Shape output adapter: one shape for
// the renderers, one links map for compact mode, and the two CSV writers. The
// dispatch (filter by strength, pick compact or verbose, format) is written
// once in writeResolutionOutput.
type resolutionOutput[DTO any] struct {
	compact    func() any // the links map for JSON/YAML compact mode
	verbose    func() any // the full DTO for JSON/YAML verbose mode
	compactCSV func(io.Writer) error
	verboseCSV func(io.Writer) error
}

func writeCLIOutput(w io.Writer, resolution ariadne.Resolution, cfg resolveConfig) error {
	filtered := filterResolutionByStrength(resolution, cfg.minStrength)
	return writeResolutionOutput(w, cfg, resolutionOutput[cliResolution]{
		compact:    func() any { return newCLILinks(filtered) },
		verbose:    func() any { return newCLIResolution(filtered) },
		compactCSV: func(w io.Writer) error { return writeCompactCSV(w, filtered) },
		verboseCSV: func(w io.Writer) error { return writeVerboseCSV(w, filtered) },
	})
}

func writeCLISongOutput(w io.Writer, resolution ariadne.SongResolution, cfg resolveConfig) error {
	filtered := filterSongResolutionByStrength(resolution, cfg.minStrength)
	return writeResolutionOutput(w, cfg, resolutionOutput[cliSongResolution]{
		compact:    func() any { return newCLISongLinks(filtered) },
		verbose:    func() any { return newCLISongResolution(filtered) },
		compactCSV: func(w io.Writer) error { return writeCompactSongCSV(w, filtered) },
		verboseCSV: func(w io.Writer) error { return writeVerboseSongCSV(w, filtered) },
	})
}

// writeResolutionOutput applies the one dispatch every entity shape shares:
// compact renders links only, verbose renders the full DTO, and the format
// decides between JSON, YAML, and the matching CSV writer.
func writeResolutionOutput[DTO any](w io.Writer, cfg resolveConfig, output resolutionOutput[DTO]) error {
	if cfg.verbose {
		return writeFormattedOutput(w, output.verbose(), cfg.format, func() error { return output.verboseCSV(w) })
	}
	return writeFormattedOutput(w, output.compact(), cfg.format, func() error { return output.compactCSV(w) })
}

func writeFormattedOutput(w io.Writer, output any, format string, writeCSV func() error) error {
	switch format {
	case outputFormatJSON:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		// Titles and URLs carry raw text, not HTML; escaping would turn < and &
		// into \u003c and \u0026 for every script reading the output.
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(output); err != nil {
			return fmt.Errorf("encode output json: %w", err)
		}
		return nil
	case outputFormatYAML:
		data, err := yaml.Marshal(output)
		if err != nil {
			return fmt.Errorf("encode output yaml: %w", err)
		}
		if _, err := w.Write(data); err != nil {
			return fmt.Errorf("write output yaml: %w", err)
		}
		return nil
	case outputFormatCSV:
		return writeCSV()
	default:
		return fmt.Errorf("%w: %q", errUnsupportedFormat, format)
	}
}
