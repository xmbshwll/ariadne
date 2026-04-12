package bandcamp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/xmbshwll/ariadne/internal/adapters/adapterutil"
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/normalize"
	"github.com/xmbshwll/ariadne/internal/parse"
	"github.com/xmbshwll/ariadne/internal/score"
)

const (
	searchLimit          = 5
	searchHydrationLimit = 8
)

var (
	jsonLDPattern                = regexp.MustCompile(`(?s)<script type="application/ld\+json">\s*(\{.*?\})\s*</script>`)
	errUnexpectedBandcampService = errors.New("unexpected bandcamp service")
	errUnexpectedBandcampStatus  = errors.New("unexpected bandcamp status")
	errBandcampJSONLDNotFound    = errors.New("bandcamp json-ld not found")
	errMalformedBandcampJSONLD   = errors.New("malformed bandcamp json-ld")
)

// Option configures the Bandcamp adapter.
type Option func(*Adapter)

// WithSearchBaseURL overrides the Bandcamp search base URL.
func WithSearchBaseURL(baseURL string) Option {
	return func(adapter *Adapter) {
		adapter.searchBaseURL = strings.TrimRight(baseURL, "/")
	}
}

// Adapter implements Bandcamp source and metadata target operations via HTML scraping.
type Adapter struct {
	client        *http.Client
	searchBaseURL string
}

// New creates a Bandcamp adapter.
func New(client *http.Client, opts ...Option) *Adapter {
	if client == nil {
		client = http.DefaultClient
	}
	adapter := &Adapter{
		client:        client,
		searchBaseURL: "https://bandcamp.com",
	}
	for _, opt := range opts {
		opt(adapter)
	}
	return adapter
}

// Service returns the service implemented by this adapter.
func (a *Adapter) Service() model.ServiceName {
	return model.ServiceBandcamp
}

// ParseAlbumURL parses a Bandcamp album URL.
func (a *Adapter) ParseAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	parsed, err := parse.BandcampAlbumURL(raw)
	if err != nil {
		return nil, fmt.Errorf("parse bandcamp album url: %w", err)
	}
	return parsed, nil
}

// ParseSongURL parses a Bandcamp track URL.
func (a *Adapter) ParseSongURL(raw string) (*model.ParsedURL, error) {
	parsed, err := parse.BandcampSongURL(raw)
	if err != nil {
		return nil, fmt.Errorf("parse bandcamp song url: %w", err)
	}
	return parsed, nil
}

// FetchAlbum loads a Bandcamp album page and extracts canonical metadata from schema.org JSON-LD.
func (a *Adapter) FetchAlbum(ctx context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalAlbum, error) {
	if parsed.Service != model.ServiceBandcamp {
		return nil, fmt.Errorf("%w: %s", errUnexpectedBandcampService, parsed.Service)
	}
	return a.fetchAlbumPage(ctx, parsed.CanonicalURL)
}

// SearchByUPC is not supported for Bandcamp.
func (a *Adapter) SearchByUPC(_ context.Context, _ string) ([]model.CandidateAlbum, error) {
	return nil, nil
}

// SearchByISRC is not supported for Bandcamp.
func (a *Adapter) SearchByISRC(_ context.Context, _ []string) ([]model.CandidateAlbum, error) {
	return nil, nil
}

// SearchByMetadata searches Bandcamp HTML results and hydrates matching album pages.
func (a *Adapter) SearchByMetadata(ctx context.Context, album model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
	query := metadataQuery(album)
	if query == "" {
		return nil, nil
	}

	searchURL := fmt.Sprintf("%s/search?q=%s", a.searchBaseURL, url.QueryEscape(query))
	body, err := a.fetchPage(ctx, searchURL)
	if err != nil {
		return nil, fmt.Errorf("fetch bandcamp search page: %w", err)
	}

	searchCandidates := rankSearchCandidates(album, extractSearchCandidates(body))
	results, err := adapterutil.CollectCandidatesWithContext(
		ctx,
		searchCandidates,
		searchHydrationLimit,
		bandcampSearchCandidateURL,
		a.hydrateBandcampAlbumSearchCandidate,
	)
	if err != nil {
		return nil, fmt.Errorf("collect bandcamp album candidates: %w", err)
	}
	if len(results) == 0 {
		return nil, nil
	}

	ranking := score.RankAlbums(album, results, score.DefaultWeights())
	ordered := make([]model.CandidateAlbum, 0, minInt(len(ranking.Ranked), searchLimit))
	for i, ranked := range ranking.Ranked {
		if i >= searchLimit {
			break
		}
		ordered = append(ordered, ranked.Candidate)
	}
	return ordered, nil
}

// FetchSong loads a Bandcamp track page and extracts canonical metadata from schema.org JSON-LD.
func (a *Adapter) FetchSong(ctx context.Context, parsed model.ParsedURL) (*model.CanonicalSong, error) {
	if parsed.Service != model.ServiceBandcamp {
		return nil, fmt.Errorf("%w: %s", errUnexpectedBandcampService, parsed.Service)
	}
	return a.fetchSongPage(ctx, parsed.CanonicalURL)
}

// SearchSongByISRC is not supported for Bandcamp.
func (a *Adapter) SearchSongByISRC(_ context.Context, _ string) ([]model.CandidateSong, error) {
	return nil, nil
}

// SearchSongByMetadata searches Bandcamp HTML results and hydrates matching track pages.
func (a *Adapter) SearchSongByMetadata(ctx context.Context, song model.CanonicalSong) ([]model.CandidateSong, error) {
	query := songMetadataQuery(song)
	if query == "" {
		return nil, nil
	}

	searchURL := fmt.Sprintf("%s/search?q=%s", a.searchBaseURL, url.QueryEscape(query))
	body, err := a.fetchPage(ctx, searchURL)
	if err != nil {
		return nil, fmt.Errorf("fetch bandcamp search page: %w", err)
	}

	searchCandidates := rankSongSearchCandidates(song, extractSongSearchCandidates(body))
	results, err := adapterutil.CollectCandidatesWithContext(
		ctx,
		searchCandidates,
		searchHydrationLimit,
		bandcampSearchCandidateURL,
		a.hydrateBandcampSongSearchCandidate,
	)
	if err != nil {
		return nil, fmt.Errorf("collect bandcamp song candidates: %w", err)
	}
	if len(results) == 0 {
		return nil, nil
	}

	ranking := score.RankSongs(song, results, score.DefaultSongWeights())
	ordered := make([]model.CandidateSong, 0, minInt(len(ranking.Ranked), searchLimit))
	for i, ranked := range ranking.Ranked {
		if i >= searchLimit {
			break
		}
		ordered = append(ordered, ranked.Candidate)
	}
	return ordered, nil
}

func bandcampSearchCandidateURL(candidate searchCandidate) string {
	return candidate.URL
}

func (a *Adapter) hydrateBandcampAlbumSearchCandidate(ctx context.Context, candidate searchCandidate) (model.CandidateAlbum, error) {
	canonical, err := a.fetchAlbumPage(ctx, candidate.URL)
	if err != nil {
		return model.CandidateAlbum{}, fmt.Errorf("hydrate bandcamp album %s: %w", candidate.URL, err)
	}
	return model.CandidateAlbum{
		CanonicalAlbum: *canonical,
		CandidateID:    canonical.SourceID,
		MatchURL:       canonical.SourceURL,
	}, nil
}

func (a *Adapter) hydrateBandcampSongSearchCandidate(ctx context.Context, candidate searchCandidate) (model.CandidateSong, error) {
	canonical, err := a.fetchSongPage(ctx, candidate.URL)
	if err != nil {
		return model.CandidateSong{}, fmt.Errorf("hydrate bandcamp song %s: %w", candidate.URL, err)
	}
	return model.CandidateSong{
		CanonicalSong: *canonical,
		CandidateID:   canonical.SourceID,
		MatchURL:      canonical.SourceURL,
	}, nil
}

func (a *Adapter) fetchAlbumPage(ctx context.Context, rawURL string) (*model.CanonicalAlbum, error) {
	body, err := a.fetchPage(ctx, rawURL)
	if err != nil {
		return nil, fmt.Errorf("fetch bandcamp album page: %w", err)
	}

	parsed, err := parse.BandcampAlbumURL(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse bandcamp album url: %w", err)
	}

	schema, err := extractSchemaAlbum(body)
	if err != nil {
		return nil, fmt.Errorf("extract bandcamp schema album: %w", err)
	}
	return toCanonicalAlbum(*parsed, schema), nil
}

func (a *Adapter) fetchSongPage(ctx context.Context, rawURL string) (*model.CanonicalSong, error) {
	body, err := a.fetchPage(ctx, rawURL)
	if err != nil {
		return nil, fmt.Errorf("fetch bandcamp song page: %w", err)
	}

	parsed, err := parse.BandcampSongURL(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse bandcamp song url: %w", err)
	}

	schema, err := extractSchemaTrack(body)
	if err != nil {
		return nil, fmt.Errorf("extract bandcamp schema track: %w", err)
	}
	return toCanonicalSong(*parsed, schema), nil
}

func (a *Adapter) fetchPage(ctx context.Context, requestURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build bandcamp request: %w", err)
	}
	req.Header.Set("User-Agent", "ariadne/0.1 (+https://github.com/xmbshwll/ariadne)")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute bandcamp request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("%w %d: %s", errUnexpectedBandcampStatus, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read bandcamp response: %w", err)
	}
	return body, nil
}

func extractSchemaAlbum(body []byte) (*schemaAlbum, error) {
	schema, err := extractSchema(body)
	if err != nil {
		return nil, err
	}
	return schema, nil
}

func extractSchemaTrack(body []byte) (*schemaAlbum, error) {
	schema, err := extractSchema(body)
	if err != nil {
		return nil, err
	}
	return schema, nil
}

func extractSchema(body []byte) (*schemaAlbum, error) {
	matches := jsonLDPattern.FindSubmatch(body)
	if len(matches) != 2 {
		return nil, errBandcampJSONLDNotFound
	}

	var schema schemaAlbum
	if err := json.Unmarshal(matches[1], &schema); err != nil {
		return nil, errors.Join(errMalformedBandcampJSONLD, fmt.Errorf("unmarshal bandcamp json-ld: %w", err))
	}
	return &schema, nil
}

func toCanonicalAlbum(parsed model.ParsedAlbumURL, album *schemaAlbum) *model.CanonicalAlbum {
	tracks := make([]model.CanonicalTrack, 0, len(album.Track.ItemListElement))
	totalDurationMS := 0
	for _, item := range album.Track.ItemListElement {
		durationMS := parseISODurationMilliseconds(item.Item.Duration)
		totalDurationMS += durationMS
		tracks = append(tracks, model.CanonicalTrack{
			TrackNumber:     item.Position,
			Title:           item.Item.Name,
			NormalizedTitle: normalize.Text(item.Item.Name),
			DurationMS:      durationMS,
			Artists:         []string{album.ByArtist.Name},
		})
	}

	imageURL := schemaImageURL(album.Image)
	return &model.CanonicalAlbum{
		Service:           model.ServiceBandcamp,
		SourceID:          parsed.ID,
		SourceURL:         parsed.CanonicalURL,
		Title:             album.Name,
		NormalizedTitle:   normalize.Text(album.Name),
		Artists:           []string{album.ByArtist.Name},
		NormalizedArtists: normalize.Artists([]string{album.ByArtist.Name}),
		ReleaseDate:       dateOnly(album.DatePublished),
		Label:             album.Publisher.Name,
		TrackCount:        len(tracks),
		TotalDurationMS:   totalDurationMS,
		ArtworkURL:        imageURL,
		EditionHints:      normalize.EditionHints(album.Name),
		Tracks:            tracks,
	}
}

func toCanonicalSong(parsed model.ParsedURL, track *schemaAlbum) *model.CanonicalSong {
	artists := nonEmptyArtistList(track.ByArtist.Name)
	albumID := ""
	if parsedAlbum, err := parse.BandcampAlbumURL(track.InAlbum.ID); err == nil {
		albumID = parsedAlbum.ID
	}
	return &model.CanonicalSong{
		Service:                model.ServiceBandcamp,
		SourceID:               parsed.ID,
		SourceURL:              parsed.CanonicalURL,
		Title:                  track.Name,
		NormalizedTitle:        normalize.Text(track.Name),
		Artists:                artists,
		NormalizedArtists:      normalize.Artists(artists),
		DurationMS:             parseISODurationMilliseconds(track.Duration),
		AlbumID:                albumID,
		AlbumTitle:             track.InAlbum.Name,
		AlbumNormalizedTitle:   normalize.Text(track.InAlbum.Name),
		AlbumArtists:           artists,
		AlbumNormalizedArtists: normalize.Artists(artists),
		ReleaseDate:            dateOnly(track.DatePublished),
		ArtworkURL:             schemaImageURL(track.Image),
		EditionHints:           normalize.EditionHints(track.Name),
	}
}

func songMetadataQuery(song model.CanonicalSong) string {
	parts := make([]string, 0, 2)
	if song.Title != "" {
		parts = append(parts, song.Title)
	}
	if len(song.Artists) > 0 {
		parts = append(parts, song.Artists[0])
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func schemaImageURL(value any) string {
	switch image := value.(type) {
	case string:
		return image
	case []any:
		for _, entry := range image {
			if urlValue, ok := entry.(string); ok && urlValue != "" {
				return urlValue
			}
		}
	}
	return ""
}

func parseISODurationMilliseconds(value string) int {
	if value == "" {
		return 0
	}
	value = strings.TrimPrefix(value, "P")
	value = strings.TrimPrefix(value, "T")
	var hours, minutes int
	var seconds float64
	for len(value) > 0 {
		index := strings.IndexAny(value, "HMS")
		if index <= 0 {
			break
		}
		number := value[:index]
		unit := value[index]
		value = value[index+1:]
		switch unit {
		case 'H':
			parsed, _ := time.ParseDuration(number + "h")
			hours = int(parsed.Hours())
		case 'M':
			parsed, _ := time.ParseDuration(number + "m")
			minutes = int(parsed.Minutes()) % 60
		case 'S':
			parsed, err := time.ParseDuration(number + "s")
			if err == nil {
				seconds = parsed.Seconds()
			}
		}
	}
	return int((float64(hours*3600+minutes*60) + seconds) * 1000)
}

func dateOnly(value string) string {
	if len(value) < 10 {
		return value
	}
	parsed, err := time.Parse(time.RFC1123, value)
	if err == nil {
		return parsed.Format("2006-01-02")
	}
	parsed, err = time.Parse("02 Jan 2006 15:04:05 MST", value)
	if err == nil {
		return parsed.Format("2006-01-02")
	}
	return value[:10]
}

func metadataQuery(album model.CanonicalAlbum) string {
	parts := make([]string, 0, 2)
	if album.Title != "" {
		parts = append(parts, album.Title)
	}
	if len(album.Artists) > 0 {
		parts = append(parts, album.Artists[0])
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func nonEmptyArtistList(artist string) []string {
	artist = strings.TrimSpace(artist)
	if artist == "" {
		return nil
	}
	return []string{artist}
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}
