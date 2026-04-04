package soundcloud

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/normalize"
	"github.com/xmbshwll/ariadne/internal/parse"
)

const (
	defaultSiteBaseURL = "https://soundcloud.com"
	defaultAPIBaseURL  = "https://api-v2.soundcloud.com"
	searchLimit        = 5
)

var (
	hydrationPattern = regexp.MustCompile(`(?s)__sc_hydration\s*=\s*(\[.*?\]);`)
	scriptSrcPattern = regexp.MustCompile(`(?i)<script[^>]+src="([^"]+)"`)
	clientIDPattern  = regexp.MustCompile(`client_id[:=]\s*"([a-zA-Z0-9]{32})"`)
)

type Option func(*Adapter)

func WithSiteBaseURL(baseURL string) Option {
	return func(adapter *Adapter) {
		adapter.siteBaseURL = strings.TrimRight(baseURL, "/")
	}
}

func WithAPIBaseURL(baseURL string) Option {
	return func(adapter *Adapter) {
		adapter.apiBaseURL = strings.TrimRight(baseURL, "/")
	}
}

type Adapter struct {
	client      *http.Client
	siteBaseURL string
	apiBaseURL  string

	clientIDMu sync.Mutex
	clientID   string
}

func New(client *http.Client, opts ...Option) *Adapter {
	if client == nil {
		client = http.DefaultClient
	}
	adapter := &Adapter{
		client:      client,
		siteBaseURL: defaultSiteBaseURL,
		apiBaseURL:  defaultAPIBaseURL,
	}
	for _, opt := range opts {
		opt(adapter)
	}
	return adapter
}

func (a *Adapter) Service() model.ServiceName {
	return model.ServiceSoundCloud
}

func (a *Adapter) ParseAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	return parse.SoundCloudAlbumURL(raw)
}

func (a *Adapter) FetchAlbum(ctx context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalAlbum, error) {
	if parsed.Service != model.ServiceSoundCloud {
		return nil, fmt.Errorf("unexpected service: %s", parsed.Service)
	}
	body, err := a.fetchPage(ctx, parsed.CanonicalURL)
	if err != nil {
		return nil, err
	}
	playlist, err := extractPlaylistHydration(body, parsed.CanonicalURL)
	if err != nil {
		return nil, err
	}
	a.maybeCacheClientIDFromPage(body)
	return toCanonicalAlbum(*playlist), nil
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
	clientID, err := a.clientIdentifier(ctx)
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("%s/search/playlists?q=%s&client_id=%s&limit=%d", a.apiBaseURL, url.QueryEscape(query), url.QueryEscape(clientID), searchLimit)
	var payload searchResponse
	if err := a.getJSON(ctx, endpoint, &payload); err != nil {
		return nil, fmt.Errorf("search soundcloud metadata: %w", err)
	}
	results := make([]model.CandidateAlbum, 0, min(len(payload.Collection), searchLimit))
	for _, playlist := range payload.Collection {
		if playlist.Kind != "playlist" {
			continue
		}
		canonical := toCanonicalAlbum(playlist)
		results = append(results, toCandidateAlbum(*canonical))
		if len(results) >= searchLimit {
			break
		}
	}
	return results, nil
}

func (a *Adapter) fetchPage(ctx context.Context, requestURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build soundcloud request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute soundcloud request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("unexpected soundcloud status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read soundcloud response: %w", err)
	}
	return body, nil
}

func (a *Adapter) getJSON(ctx context.Context, requestURL string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return fmt.Errorf("build soundcloud api request: %w", err)
	}
	req.Header.Set("User-Agent", "ariadne/0.1 (+https://github.com/xmbshwll/ariadne)")
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("execute soundcloud api request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("unexpected soundcloud api status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode soundcloud api response: %w", err)
	}
	return nil
}

func (a *Adapter) clientIdentifier(ctx context.Context) (string, error) {
	a.clientIDMu.Lock()
	cachedClientID := a.clientID
	a.clientIDMu.Unlock()
	if cachedClientID != "" {
		return cachedClientID, nil
	}

	body, err := a.fetchPage(ctx, a.siteBaseURL)
	if err != nil {
		return "", err
	}
	clientID, err := a.findClientID(ctx, body)
	if err != nil {
		return "", err
	}
	a.clientIDMu.Lock()
	defer a.clientIDMu.Unlock()
	a.clientID = clientID
	return a.clientID, nil
}

func (a *Adapter) maybeCacheClientIDFromPage(body []byte) {
	clientID := extractClientID(body)
	if clientID == "" {
		return
	}
	a.clientIDMu.Lock()
	defer a.clientIDMu.Unlock()
	if a.clientID == "" {
		a.clientID = clientID
	}
}

func (a *Adapter) findClientID(ctx context.Context, body []byte) (string, error) {
	if clientID := extractClientID(body); clientID != "" {
		return clientID, nil
	}
	scriptMatches := scriptSrcPattern.FindAllSubmatch(body, -1)
	for _, match := range scriptMatches {
		if len(match) != 2 {
			continue
		}
		scriptURL := strings.TrimSpace(string(match[1]))
		if scriptURL == "" {
			continue
		}
		if strings.HasPrefix(scriptURL, "/") {
			scriptURL = a.siteBaseURL + scriptURL
		}
		assetBody, err := a.fetchPage(ctx, scriptURL)
		if err != nil {
			continue
		}
		if clientID := extractClientID(assetBody); clientID != "" {
			return clientID, nil
		}
	}
	return "", fmt.Errorf("soundcloud client id not found")
}

func extractPlaylistHydration(body []byte, canonicalURL string) (*soundPlaylist, error) {
	matches := hydrationPattern.FindSubmatch(body)
	if len(matches) != 2 {
		return nil, fmt.Errorf("soundcloud hydration payload not found")
	}
	var entries []hydrationEnvelope
	if err := json.Unmarshal(matches[1], &entries); err != nil {
		return nil, fmt.Errorf("decode soundcloud hydration payload: %w", err)
	}
	var fallback *soundPlaylist
	for _, entry := range entries {
		if entry.Hydratable != "playlist" || entry.Data.PermalinkURL == "" {
			continue
		}
		playlist := entry.Data
		if fallback == nil {
			fallback = &playlist
		}
		if canonicalizeSoundCloudURL(playlist.PermalinkURL) == canonicalURL {
			return &playlist, nil
		}
	}
	if fallback != nil {
		return fallback, nil
	}
	return nil, fmt.Errorf("soundcloud playlist hydration not found")
}

func toCanonicalAlbum(playlist soundPlaylist) *model.CanonicalAlbum {
	artists := nonEmptyArtistList(firstNonEmpty(playlist.User.Username, trackArtist(playlist.Tracks)))
	tracks := make([]model.CanonicalTrack, 0, len(playlist.Tracks))
	totalDurationMS := playlist.Duration
	explicit := false
	for index, track := range playlist.Tracks {
		durationMS := track.FullDuration
		if durationMS == 0 {
			durationMS = track.Duration
		}
		if durationMS != 0 && playlist.Duration == 0 {
			totalDurationMS += durationMS
		}
		artistNames := nonEmptyArtistList(firstNonEmpty(track.PublisherMetadata.Artist, track.User.Username, playlist.User.Username))
		tracks = append(tracks, model.CanonicalTrack{
			TrackNumber:     index + 1,
			Title:           track.Title,
			NormalizedTitle: normalize.Text(track.Title),
			DurationMS:      durationMS,
			ISRC:            strings.TrimSpace(track.PublisherMetadata.ISRC),
			Artists:         artistNames,
		})
		if track.PublisherMetadata.Explicit {
			explicit = true
		}
	}
	if totalDurationMS == 0 {
		for _, track := range tracks {
			totalDurationMS += track.DurationMS
		}
	}
	upc := consistentUPC(playlist.Tracks)
	label := firstNonEmpty(playlist.LabelName, trackLabel(playlist.Tracks), trackPLine(playlist.Tracks))
	canonicalURL := canonicalizeSoundCloudURL(playlist.PermalinkURL)
	sourceID := soundCloudSourceID(canonicalURL)
	releaseDate := firstNonEmpty(dateOnly(playlist.ReleaseDate), dateOnly(playlist.PublishedAt), dateOnly(playlist.DisplayDate))
	return &model.CanonicalAlbum{
		Service:           model.ServiceSoundCloud,
		SourceID:          sourceID,
		SourceURL:         canonicalURL,
		Title:             playlist.Title,
		NormalizedTitle:   normalize.Text(playlist.Title),
		Artists:           artists,
		NormalizedArtists: normalize.Artists(artists),
		ReleaseDate:       releaseDate,
		Label:             label,
		UPC:               upc,
		TrackCount:        len(tracks),
		TotalDurationMS:   totalDurationMS,
		ArtworkURL:        playlist.ArtworkURL,
		Explicit:          explicit,
		EditionHints:      normalize.EditionHints(playlist.Title),
		Tracks:            tracks,
	}
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

func canonicalizeSoundCloudURL(raw string) string {
	parsed, err := parse.SoundCloudAlbumURL(raw)
	if err != nil {
		return strings.TrimSpace(raw)
	}
	return parsed.CanonicalURL
}

func consistentUPC(tracks []soundTrack) string {
	upc := ""
	for _, track := range tracks {
		candidate := strings.TrimSpace(track.PublisherMetadata.UPCOrEAN)
		if candidate == "" {
			continue
		}
		if upc == "" {
			upc = candidate
			continue
		}
		if upc != candidate {
			return ""
		}
	}
	return upc
}

func trackArtist(tracks []soundTrack) string {
	for _, track := range tracks {
		if artist := firstNonEmpty(track.PublisherMetadata.Artist, track.User.Username); artist != "" {
			return artist
		}
	}
	return ""
}

func trackLabel(tracks []soundTrack) string {
	for _, track := range tracks {
		if label := firstNonEmpty(track.LabelName); label != "" {
			return label
		}
	}
	return ""
}

func trackPLine(tracks []soundTrack) string {
	for _, track := range tracks {
		if pLine := firstNonEmpty(track.PublisherMetadata.PLineForDisplay, track.PublisherMetadata.CLineForDisplay); pLine != "" {
			return pLine
		}
	}
	return ""
}

func dateOnly(value string) string {
	if len(value) >= 10 {
		return value[:10]
	}
	return strings.TrimSpace(value)
}

func nonEmptyArtistList(artist string) []string {
	artist = strings.TrimSpace(artist)
	if artist == "" {
		return nil
	}
	return []string{artist}
}

func extractClientID(body []byte) string {
	if matches := clientIDPattern.FindSubmatch(body); len(matches) == 2 {
		return string(matches[1])
	}
	return ""
}

func toCandidateAlbum(album model.CanonicalAlbum) model.CandidateAlbum {
	return model.CandidateAlbum{
		CanonicalAlbum: album,
		CandidateID:    album.SourceID,
		MatchURL:       album.SourceURL,
	}
}

func soundCloudSourceID(canonicalURL string) string {
	parsed, err := parse.SoundCloudAlbumURL(canonicalURL)
	if err != nil {
		return canonicalURL
	}
	return parsed.ID
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
