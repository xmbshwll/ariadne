package youtubemusic

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/xmbshwll/ariadne/internal/adapters/adapterutil"
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/normalize"
	"github.com/xmbshwll/ariadne/internal/parse"
)

const (
	defaultBaseURL = "https://music.youtube.com"
	searchLimit    = 5
)

var (
	canonicalURLPattern               = regexp.MustCompile(`(?i)<link rel="canonical" href="([^"]+)"`)
	ogTitlePattern                    = regexp.MustCompile(`(?i)<meta property="og:title" content="([^"]+)"`)
	ogImagePattern                    = regexp.MustCompile(`(?i)<meta property="og:image" content="([^"]+)"`)
	subtitleArtistPattern             = regexp.MustCompile(`subtitle\\x22:\\x7b\\x22runs\\x22:\\x5b\\x7b\\x22text\\x22:\\x22Album\\x22\\x7d,\\x7b\\x22text\\x22:\\x22 .*?\\x7d,\\x7b\\x22text\\x22:\\x22([^\\]+?)\\x22`)
	trackTitlePattern                 = regexp.MustCompile(`musicResponsiveListItemFlexColumnRenderer\\x22:\\x7b\\x22text\\x22:\\x7b\\x22runs\\x22:\\x5b\\x7b\\x22text\\x22:\\x22([^\\]+?)\\x22`)
	albumResultPattern                = regexp.MustCompile(`title\\x22:\\x7b\\x22runs\\x22:\\x5b\\x7b\\x22text\\x22:\\x22([^\\]+?)\\x22,\\x22navigationEndpoint\\x22:\\x7b.*?browseId\\x22:\\x22([^\\]+?)\\x22.*?pageType\\x22:\\x22MUSIC_PAGE_TYPE_ALBUM\\x22.*?subtitle\\x22:\\x7b\\x22runs\\x22:\\x5b\\x7b\\x22text\\x22:\\x22Album\\x22\\x7d,\\x7b\\x22text\\x22:\\x22 .*?\\x7d,\\x7b\\x22text\\x22:\\x22([^\\]+?)\\x22`)
	errUnexpectedYouTubeMusicService  = errors.New("unexpected youtube music service")
	errUnexpectedYouTubeMusicStatus   = errors.New("unexpected youtube music status")
	errMalformedYouTubeMusicPage      = errors.New("malformed youtube music page")
	errYouTubeMusicAlbumTitleNotFound = errors.New("youtube music album title not found")
)

type Option func(*Adapter)

func WithBaseURL(baseURL string) Option {
	return func(adapter *Adapter) {
		adapter.baseURL = strings.TrimRight(baseURL, "/")
	}
}

type Adapter struct {
	client  *http.Client
	baseURL string
}

func New(client *http.Client, opts ...Option) *Adapter {
	if client == nil {
		client = http.DefaultClient
	}
	adapter := &Adapter{client: client, baseURL: defaultBaseURL}
	for _, opt := range opts {
		opt(adapter)
	}
	return adapter
}

func (a *Adapter) Service() model.ServiceName {
	return model.ServiceYouTubeMusic
}

func (a *Adapter) ParseAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	parsed, err := parse.YouTubeMusicAlbumURL(raw)
	if err != nil {
		return nil, fmt.Errorf("parse youtube music album url: %w", err)
	}
	return parsed, nil
}

func (a *Adapter) FetchAlbum(ctx context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalAlbum, error) {
	if parsed.Service != model.ServiceYouTubeMusic {
		return nil, fmt.Errorf("%w: %s", errUnexpectedYouTubeMusicService, parsed.Service)
	}
	body, err := a.fetchPage(ctx, parsed.CanonicalURL)
	if err != nil {
		return nil, fmt.Errorf("fetch youtube music page: %w", err)
	}
	return extractAlbum(body, parsed.CanonicalURL)
}

func (a *Adapter) SearchByUPC(_ context.Context, _ string) ([]model.CandidateAlbum, error) {
	return nil, nil
}

func (a *Adapter) SearchByISRC(_ context.Context, _ []string) ([]model.CandidateAlbum, error) {
	return nil, nil
}

func (a *Adapter) SearchByMetadata(ctx context.Context, album model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
	query := metadataQuery(album)
	if query == "" {
		return nil, nil
	}
	searchURL := fmt.Sprintf("%s/search?q=%s", a.baseURL, url.QueryEscape(query))
	body, err := a.fetchPage(ctx, searchURL)
	if err != nil {
		return nil, fmt.Errorf("fetch youtube music search page: %w", err)
	}
	candidates := extractSearchCandidates(body)
	results, err := adapterutil.CollectCandidates(
		candidates,
		searchLimit,
		youTubeMusicSearchCandidateID,
		func(candidate searchCandidate) (model.CandidateAlbum, error) {
			return a.hydrateYouTubeMusicAlbumSearchCandidate(ctx, candidate)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("collect youtube music candidates: %w", err)
	}
	return results, nil
}

func youTubeMusicSearchCandidateID(candidate searchCandidate) string {
	return candidate.BrowseID
}

func (a *Adapter) hydrateYouTubeMusicAlbumSearchCandidate(ctx context.Context, candidate searchCandidate) (model.CandidateAlbum, error) {
	canonical, err := a.fetchAlbumByBrowseID(ctx, candidate.BrowseID)
	if err != nil {
		return model.CandidateAlbum{}, fmt.Errorf("hydrate youtube music album %s: %w", candidate.BrowseID, err)
	}
	return toCandidateAlbum(*canonical), nil
}

func (a *Adapter) fetchAlbumByBrowseID(ctx context.Context, browseID string) (*model.CanonicalAlbum, error) {
	browseURL := a.baseURL + "/browse/" + browseID
	body, err := a.fetchPage(ctx, browseURL)
	if err != nil {
		return nil, err
	}
	return extractAlbum(body, browseURL)
}

func (a *Adapter) fetchPage(ctx context.Context, requestURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build youtube music request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute youtube music request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("%w %d: %s", errUnexpectedYouTubeMusicStatus, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read youtube music response: %w", err)
	}
	return body, nil
}

type searchCandidate struct {
	Title    string
	BrowseID string
	Artist   string
}

func extractAlbum(body []byte, fallbackURL string) (*model.CanonicalAlbum, error) {
	canonicalURL := extractFirstGroup(canonicalURLPattern, body)
	if canonicalURL == "" {
		canonicalURL = strings.TrimSpace(fallbackURL)
	}
	title := cleanAlbumTitle(extractFirstGroup(ogTitlePattern, body))
	if title == "" {
		return nil, errors.Join(errMalformedYouTubeMusicPage, errYouTubeMusicAlbumTitleNotFound)
	}
	artist := html.UnescapeString(extractFirstGroup(subtitleArtistPattern, body))
	trackTitles := extractTrackTitles(body)
	parsed, _ := parse.YouTubeMusicAlbumURL(canonicalURL)
	sourceID := canonicalURL
	if parsed != nil {
		sourceID = parsed.ID
	}
	artists := nonEmptyArtistList(artist)
	tracks := make([]model.CanonicalTrack, 0, len(trackTitles))
	for index, trackTitle := range trackTitles {
		tracks = append(tracks, model.CanonicalTrack{
			TrackNumber:     index + 1,
			Title:           trackTitle,
			NormalizedTitle: normalize.Text(trackTitle),
			Artists:         artists,
		})
	}
	return &model.CanonicalAlbum{
		Service:           model.ServiceYouTubeMusic,
		SourceID:          sourceID,
		SourceURL:         canonicalURL,
		Title:             title,
		NormalizedTitle:   normalize.Text(title),
		Artists:           artists,
		NormalizedArtists: normalize.Artists(artists),
		TrackCount:        len(tracks),
		ArtworkURL:        extractFirstGroup(ogImagePattern, body),
		EditionHints:      normalize.EditionHints(title),
		Tracks:            tracks,
	}, nil
}

func extractSearchCandidates(body []byte) []searchCandidate {
	matches := albumResultPattern.FindAllSubmatch(body, -1)
	results := make([]searchCandidate, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		if len(match) != 4 {
			continue
		}
		browseID := html.UnescapeString(string(match[2]))
		if browseID == "" {
			continue
		}
		if _, ok := seen[browseID]; ok {
			continue
		}
		seen[browseID] = struct{}{}
		results = append(results, searchCandidate{
			Title:    html.UnescapeString(string(match[1])),
			BrowseID: browseID,
			Artist:   html.UnescapeString(string(match[3])),
		})
	}
	return results
}

func extractTrackTitles(body []byte) []string {
	matches := trackTitlePattern.FindAllSubmatch(body, -1)
	titles := make([]string, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		if len(match) != 2 {
			continue
		}
		title := html.UnescapeString(string(match[1]))
		if shouldSkipTrackTitle(title) {
			continue
		}
		if _, ok := seen[title]; ok {
			continue
		}
		seen[title] = struct{}{}
		titles = append(titles, title)
	}
	return titles
}

func shouldSkipTrackTitle(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true
	}
	lower := strings.ToLower(value)
	if strings.Contains(lower, "wiedergaben") || strings.Contains(lower, "views") {
		return true
	}
	return false
}

func cleanAlbumTitle(value string) string {
	value = html.UnescapeString(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "\u00a0", " ")
	if index := strings.Index(value, " – "); index > 0 {
		return strings.TrimSpace(value[:index])
	}
	return value
}

func extractFirstGroup(pattern *regexp.Regexp, body []byte) string {
	matches := pattern.FindSubmatch(body)
	if len(matches) != 2 {
		return ""
	}
	return html.UnescapeString(string(matches[1]))
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

func toCandidateAlbum(album model.CanonicalAlbum) model.CandidateAlbum {
	return model.CandidateAlbum{CanonicalAlbum: album, CandidateID: album.SourceID, MatchURL: album.SourceURL}
}
