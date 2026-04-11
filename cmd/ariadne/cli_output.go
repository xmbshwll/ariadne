package main

import "github.com/xmbshwll/ariadne"

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

type cliMatchResult struct {
	Found      bool       `json:"found" yaml:"found"`
	Summary    string     `json:"summary" yaml:"summary"`
	Best       *cliMatch  `json:"best,omitempty" yaml:"best,omitempty"`
	Alternates []cliMatch `json:"alternates,omitempty" yaml:"alternates,omitempty"`
}

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

type cliSongMatchResult struct {
	Found      bool           `json:"found" yaml:"found"`
	Summary    string         `json:"summary" yaml:"summary"`
	Best       *cliSongMatch  `json:"best,omitempty" yaml:"best,omitempty"`
	Alternates []cliSongMatch `json:"alternates,omitempty" yaml:"alternates,omitempty"`
}

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

func newCLIOutput(resolution ariadne.Resolution, cfg resolveConfig) any {
	if cfg.verbose {
		return newCLIResolution(resolution)
	}
	return newCLILinks(resolution)
}
