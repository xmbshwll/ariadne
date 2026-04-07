package applemusic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xmbshwll/ariadne/internal/applemusicauth"
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/normalize"
	"github.com/xmbshwll/ariadne/internal/parse"
)

const (
	defaultLookupBaseURL = "https://itunes.apple.com"
	defaultAPIBaseURL    = "https://api.music.apple.com/v1"
	searchLimit          = 5
)

var (
	errUnexpectedAppleMusicService = errors.New("unexpected apple music service")
	errAppleMusicAlbumNotFound     = errors.New("apple music album not found")
	errUnexpectedAppleMusicStatus  = errors.New("unexpected apple music status")

	errUnexpectedAppleMusicOfficialStatus  = errors.New("unexpected apple music official status")
	errAppleMusicOfficialAuthNotConfigured = errors.New("apple music official auth not configured")
	errAppleMusicOfficialAlbumNotFound     = errors.New("apple music official album not found")
)

// Option configures the Apple Music adapter.
type Option func(*Adapter)

// WithLookupBaseURL overrides the iTunes lookup API base URL.
func WithLookupBaseURL(baseURL string) Option {
	return func(adapter *Adapter) {
		adapter.lookupBaseURL = strings.TrimRight(baseURL, "/")
	}
}

// WithDefaultStorefront sets the default Apple Music storefront used when the
// source album does not already carry a storefront hint.
func WithDefaultStorefront(storefront string) Option {
	return func(adapter *Adapter) {
		adapter.defaultStorefront = strings.ToLower(strings.TrimSpace(storefront))
	}
}

// WithAPIBaseURL overrides the official Apple Music API base URL.
func WithAPIBaseURL(baseURL string) Option {
	return func(adapter *Adapter) {
		adapter.apiBaseURL = strings.TrimRight(baseURL, "/")
	}
}

// WithDeveloperTokenAuth enables official Apple Music API calls by generating
// MusicKit developer tokens from the provided .p8 key material.
func WithDeveloperTokenAuth(keyID string, teamID string, privateKeyPath string) Option {
	return func(adapter *Adapter) {
		adapter.appleMusicKeyID = strings.TrimSpace(keyID)
		adapter.appleMusicTeamID = strings.TrimSpace(teamID)
		adapter.appleMusicPrivateKeyPath = strings.TrimSpace(privateKeyPath)
	}
}

// Adapter implements Apple Music source operations using the public lookup API.
type Adapter struct {
	client                   *http.Client
	lookupBaseURL            string
	apiBaseURL               string
	defaultStorefront        string
	appleMusicKeyID          string
	appleMusicTeamID         string
	appleMusicPrivateKeyPath string
	tokenMu                  sync.Mutex
	cachedToken              string
	tokenExpiresAt           time.Time
}

// New creates an Apple Music adapter.
func New(client *http.Client, opts ...Option) *Adapter {
	if client == nil {
		client = http.DefaultClient
	}
	adapter := &Adapter{
		client:            client,
		lookupBaseURL:     defaultLookupBaseURL,
		apiBaseURL:        defaultAPIBaseURL,
		defaultStorefront: "us",
	}
	for _, opt := range opts {
		opt(adapter)
	}
	return adapter
}

// Service returns the service implemented by this adapter.
func (a *Adapter) Service() model.ServiceName {
	return model.ServiceAppleMusic
}

// ParseAlbumURL parses an Apple Music album URL.
func (a *Adapter) ParseAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	parsed, err := parse.AppleMusicAlbumURL(raw)
	if err != nil {
		return nil, fmt.Errorf("parse apple music album url: %w", err)
	}
	return parsed, nil
}

// FetchAlbum loads Apple Music album metadata from the lookup API and maps it into the canonical model.
func (a *Adapter) FetchAlbum(ctx context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalAlbum, error) {
	if parsed.Service != model.ServiceAppleMusic {
		return nil, fmt.Errorf("%w: %s", errUnexpectedAppleMusicService, parsed.Service)
	}
	return a.fetchAlbumByID(ctx, parsed.ID, parsed.CanonicalURL, a.storefrontFor(parsed.RegionHint))
}

// SearchByUPC uses the official Apple Music catalog API when MusicKit auth is configured.
func (a *Adapter) SearchByUPC(ctx context.Context, upc string) ([]model.CandidateAlbum, error) {
	upc = strings.TrimSpace(upc)
	if upc == "" || !a.authEnabled() {
		return nil, nil
	}

	storefront := a.defaultStorefront
	endpoint := fmt.Sprintf("%s/catalog/%s/albums?filter[upc]=%s", a.apiBaseURL, url.PathEscape(storefront), url.QueryEscape(upc))
	var payload map[string]any
	if err := a.getOfficialJSON(ctx, endpoint, &payload); err != nil {
		return nil, fmt.Errorf("search apple music by upc: %w", err)
	}
	albumIDs := officialAlbumIDs(payload)
	return a.hydrateOfficialAlbums(ctx, albumIDs, storefront)
}

// SearchByISRC uses the official Apple Music catalog API when MusicKit auth is configured.
func (a *Adapter) SearchByISRC(ctx context.Context, isrcs []string) ([]model.CandidateAlbum, error) {
	if !a.authEnabled() {
		return nil, nil
	}

	storefront := a.defaultStorefront
	seenAlbumIDs := make(map[string]struct{}, len(isrcs))
	albumIDs := make([]string, 0, len(isrcs))
	for _, isrc := range isrcs {
		isrc = strings.TrimSpace(isrc)
		if isrc == "" {
			continue
		}
		endpoint := fmt.Sprintf("%s/catalog/%s/songs?filter[isrc]=%s", a.apiBaseURL, url.PathEscape(storefront), url.QueryEscape(isrc))
		var payload map[string]any
		if err := a.getOfficialJSON(ctx, endpoint, &payload); err != nil {
			return nil, fmt.Errorf("search apple music by isrc: %w", err)
		}
		for _, albumID := range officialAlbumIDsFromSongs(payload) {
			if _, ok := seenAlbumIDs[albumID]; ok {
				continue
			}
			seenAlbumIDs[albumID] = struct{}{}
			albumIDs = append(albumIDs, albumID)
			if len(albumIDs) >= searchLimit {
				return a.hydrateOfficialAlbums(ctx, albumIDs, storefront)
			}
		}
	}
	return a.hydrateOfficialAlbums(ctx, albumIDs, storefront)
}

// SearchByMetadata searches Apple Music albums by title and artist metadata via the public search API.
func (a *Adapter) SearchByMetadata(ctx context.Context, album model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
	queries := metadataQueries(album)
	if len(queries) == 0 {
		return nil, nil
	}

	storefront := a.storefrontFor(album.RegionHint)
	results := make([]model.CandidateAlbum, 0, searchLimit)
	seen := make(map[int64]struct{}, searchLimit)

	for _, query := range queries {
		searchURL := fmt.Sprintf("%s/search?term=%s&entity=album&limit=%d&country=%s", a.lookupBaseURL, url.QueryEscape(query), searchLimit, url.QueryEscape(storefront))
		var payload lookupResponse
		if err := a.getJSON(ctx, searchURL, &payload); err != nil {
			return nil, fmt.Errorf("search apple music metadata %q: %w", query, err)
		}

		for _, item := range payload.Results {
			if item.WrapperType != "collection" || item.CollectionType != "Album" {
				continue
			}
			if _, ok := seen[item.CollectionID]; ok {
				continue
			}
			seen[item.CollectionID] = struct{}{}

			canonical, err := a.fetchAlbumByID(ctx, strconv.FormatInt(item.CollectionID, 10), canonicalCollectionURL(item.CollectionViewURL, ""), storefront)
			if err != nil {
				return nil, fmt.Errorf("hydrate apple music album %d: %w", item.CollectionID, err)
			}
			results = append(results, toCandidateAlbum(*canonical))
			if len(results) >= searchLimit {
				return results, nil
			}
		}
	}
	return results, nil
}

func (a *Adapter) fetchAlbumByID(ctx context.Context, albumID string, canonicalURL string, storefront string) (*model.CanonicalAlbum, error) {
	lookupURL := fmt.Sprintf("%s/lookup?id=%s&entity=song&country=%s", a.lookupBaseURL, url.QueryEscape(albumID), url.QueryEscape(a.storefrontFor(storefront)))
	var payload lookupResponse
	if err := a.getJSON(ctx, lookupURL, &payload); err != nil {
		return nil, err
	}
	if len(payload.Results) == 0 {
		return nil, fmt.Errorf("%w: %s", errAppleMusicAlbumNotFound, albumID)
	}

	parsed := model.ParsedAlbumURL{
		Service:      model.ServiceAppleMusic,
		EntityType:   "album",
		ID:           albumID,
		CanonicalURL: canonicalURL,
		RegionHint:   a.storefrontFor(storefront),
	}
	if parsed.CanonicalURL == "" {
		parsed.CanonicalURL = canonicalCollectionURL(payload.Results[0].CollectionViewURL, "")
	}
	return toCanonicalAlbum(parsed, payload.Results), nil
}

func (a *Adapter) getJSON(ctx context.Context, requestURL string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return fmt.Errorf("build apple music request: %w", err)
	}
	req.Header.Set("User-Agent", "ariadne/0.1 (+https://github.com/xmbshwll/ariadne)")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("execute apple music request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("%w %d: %s", errUnexpectedAppleMusicStatus, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode apple music response: %w", err)
	}
	return nil
}

func (a *Adapter) getOfficialJSON(ctx context.Context, requestURL string, target any) error {
	developerToken, err := a.developerToken()
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return fmt.Errorf("build apple music official request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+developerToken)
	req.Header.Set("User-Agent", "ariadne/0.1 (+https://github.com/xmbshwll/ariadne)")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("execute apple music official request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("%w %d: %s", errUnexpectedAppleMusicOfficialStatus, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode apple music official response: %w", err)
	}
	return nil
}

func toCanonicalAlbum(parsed model.ParsedAlbumURL, items []lookupItem) *model.CanonicalAlbum {
	const explicitTrack = "explicit"

	collection := items[0]
	tracks := make([]model.CanonicalTrack, 0, len(items)-1)
	totalDurationMS := 0
	trackCount := 0
	explicit := false

	for _, item := range items[1:] {
		if item.WrapperType != "track" || item.Kind != "song" {
			continue
		}
		trackCount++
		totalDurationMS += item.TrackTimeMillis
		if item.TrackExplicitness == explicitTrack {
			explicit = true
		}
		tracks = append(tracks, model.CanonicalTrack{
			DiscNumber:      item.DiscNumber,
			TrackNumber:     item.TrackNumber,
			Title:           item.TrackName,
			NormalizedTitle: normalize.Text(item.TrackName),
			DurationMS:      item.TrackTimeMillis,
			Artists:         []string{item.ArtistName},
		})
	}

	if trackCount == 0 {
		trackCount = collection.TrackCount
	}

	return &model.CanonicalAlbum{
		Service:           model.ServiceAppleMusic,
		SourceID:          strconv.FormatInt(collection.CollectionID, 10),
		SourceURL:         canonicalCollectionURL(collection.CollectionViewURL, parsed.CanonicalURL),
		RegionHint:        parsed.RegionHint,
		Title:             collection.CollectionName,
		NormalizedTitle:   normalize.Text(collection.CollectionName),
		Artists:           []string{collection.ArtistName},
		NormalizedArtists: normalize.Artists([]string{collection.ArtistName}),
		ReleaseDate:       dateOnly(collection.ReleaseDate),
		Label:             collection.Copyright,
		TrackCount:        trackCount,
		TotalDurationMS:   totalDurationMS,
		ArtworkURL:        preferredArtworkURL(collection),
		Explicit:          explicit || collection.CollectionExplicitness == "explicit",
		EditionHints:      normalize.EditionHints(collection.CollectionName),
		Tracks:            tracks,
	}
}

func canonicalCollectionURL(raw string, fallback string) string {
	if raw == "" {
		return fallback
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	parsed.RawQuery = ""
	return parsed.String()
}

func preferredArtworkURL(item lookupItem) string {
	if item.ArtworkURL100 != "" {
		return strings.Replace(item.ArtworkURL100, "100x100bb", "1000x1000bb", 1)
	}
	return item.ArtworkURL60
}

func dateOnly(value string) string {
	if len(value) >= 10 {
		return value[:10]
	}
	return value
}

func (a *Adapter) storefrontFor(regionHint string) string {
	if strings.TrimSpace(regionHint) == "" {
		return a.defaultStorefront
	}
	return strings.ToLower(regionHint)
}

func metadataQueries(album model.CanonicalAlbum) []string {
	if strings.TrimSpace(album.Title) == "" {
		return nil
	}

	queries := make([]string, 0, 4)
	seen := make(map[string]struct{}, 4)
	appendUnique := func(query string) {
		query = strings.TrimSpace(query)
		if query == "" {
			return
		}
		key := normalize.Text(query)
		if key == "" {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		queries = append(queries, query)
	}

	for _, artist := range normalize.SearchArtistVariants(album.Artists) {
		appendUnique(strings.TrimSpace(strings.Join([]string{album.Title, artist}, " ")))
	}
	appendUnique(album.Title)
	return queries
}

func (a *Adapter) authEnabled() bool {
	return a.appleMusicKeyID != "" && a.appleMusicTeamID != "" && a.appleMusicPrivateKeyPath != ""
}

func (a *Adapter) developerToken() (string, error) {
	if !a.authEnabled() {
		return "", errAppleMusicOfficialAuthNotConfigured
	}

	a.tokenMu.Lock()
	defer a.tokenMu.Unlock()
	now := time.Now()
	if a.cachedToken != "" && now.Before(a.tokenExpiresAt) {
		return a.cachedToken, nil
	}

	token, err := applemusicauth.GenerateDeveloperToken(applemusicauth.Config{
		KeyID:          a.appleMusicKeyID,
		TeamID:         a.appleMusicTeamID,
		PrivateKeyPath: a.appleMusicPrivateKeyPath,
		TTL:            time.Hour,
	}, now.UTC())
	if err != nil {
		return "", fmt.Errorf("generate apple music developer token: %w", err)
	}
	a.cachedToken = token
	a.tokenExpiresAt = now.Add(55 * time.Minute)
	return a.cachedToken, nil
}

func (a *Adapter) hydrateOfficialAlbums(ctx context.Context, albumIDs []string, storefront string) ([]model.CandidateAlbum, error) {
	results := make([]model.CandidateAlbum, 0, len(albumIDs))
	seen := make(map[string]struct{}, len(albumIDs))
	for _, albumID := range albumIDs {
		albumID = strings.TrimSpace(albumID)
		if albumID == "" {
			continue
		}
		if _, ok := seen[albumID]; ok {
			continue
		}
		seen[albumID] = struct{}{}
		album, err := a.fetchOfficialAlbumByID(ctx, albumID, storefront)
		if err != nil {
			return nil, err
		}
		results = append(results, toCandidateAlbum(*album))
		if len(results) >= searchLimit {
			break
		}
	}
	return results, nil
}

func (a *Adapter) fetchOfficialAlbumByID(ctx context.Context, albumID string, storefront string) (*model.CanonicalAlbum, error) {
	endpoint := fmt.Sprintf("%s/catalog/%s/albums/%s?include=tracks", a.apiBaseURL, url.PathEscape(storefront), url.PathEscape(albumID))
	var payload map[string]any
	if err := a.getOfficialJSON(ctx, endpoint, &payload); err != nil {
		return nil, fmt.Errorf("fetch apple music official album %s: %w", albumID, err)
	}
	resource := firstOfficialResource(payload)
	if resource == nil {
		return nil, fmt.Errorf("%w: %s", errAppleMusicOfficialAlbumNotFound, albumID)
	}
	return officialResourceToCanonicalAlbum(resource, storefront), nil
}

func firstOfficialResource(payload map[string]any) map[string]any {
	data, _ := payload["data"].([]any)
	if len(data) == 0 {
		return nil
	}
	resource, _ := data[0].(map[string]any)
	return resource
}

func officialAlbumIDs(payload map[string]any) []string {
	data, _ := payload["data"].([]any)
	ids := make([]string, 0, len(data))
	seen := make(map[string]struct{}, len(data))
	for _, item := range data {
		resource, ok := item.(map[string]any)
		if !ok {
			continue
		}
		ids = appendUniqueString(ids, seen, officialAlbumID(resource))
	}
	return ids
}

func officialAlbumIDsFromSongs(payload map[string]any) []string {
	data, _ := payload["data"].([]any)
	ids := make([]string, 0, len(data))
	seen := make(map[string]struct{}, len(data))
	for _, item := range data {
		resource, ok := item.(map[string]any)
		if !ok {
			continue
		}
		relationships := officialMap(resource, "relationships")
		albums := officialMap(relationships, "albums")
		albumData, _ := albums["data"].([]any)
		for _, candidate := range albumData {
			albumResource, ok := candidate.(map[string]any)
			if !ok {
				continue
			}
			ids = appendUniqueString(ids, seen, officialString(albumResource, "id"))
		}
	}
	return ids
}

func officialAlbumID(resource map[string]any) string {
	attributes := officialMap(resource, "attributes")
	if parsed := parseOfficialAlbumURL(officialString(attributes, "url")); parsed != nil {
		return parsed.ID
	}
	playParams := officialMap(attributes, "playParams")
	if id := officialString(playParams, "id"); id != "" {
		return id
	}
	return officialString(resource, "id")
}

func officialResourceToCanonicalAlbum(resource map[string]any, storefront string) *model.CanonicalAlbum {
	attributes := officialMap(resource, "attributes")
	title := officialString(attributes, "name")
	artist := officialString(attributes, "artistName")
	canonicalURL := officialString(attributes, "url")
	sourceID := officialAlbumID(resource)
	if parsed := parseOfficialAlbumURL(canonicalURL); parsed != nil {
		canonicalURL = parsed.CanonicalURL
	}
	tracks := officialTracks(resource)
	totalDurationMS := 0
	for _, track := range tracks {
		totalDurationMS += track.DurationMS
	}
	trackCount := officialInt(attributes, "trackCount")
	if trackCount == 0 {
		trackCount = len(tracks)
	}
	label := officialString(attributes, "recordLabel")
	if label == "" {
		label = officialString(attributes, "copyright")
	}
	artists := nonEmptyArtistList(artist)
	releaseDate := officialString(attributes, "releaseDate")
	upc := officialString(attributes, "upc")
	artworkURL := officialArtworkURL(officialMap(attributes, "artwork"))
	explicit := officialString(attributes, "contentRating") == "explicit"
	return &model.CanonicalAlbum{
		Service:           model.ServiceAppleMusic,
		SourceID:          sourceID,
		SourceURL:         canonicalURL,
		RegionHint:        storefront,
		Title:             title,
		NormalizedTitle:   normalize.Text(title),
		Artists:           artists,
		NormalizedArtists: normalize.Artists(artists),
		ReleaseDate:       releaseDate,
		Label:             label,
		UPC:               upc,
		TrackCount:        trackCount,
		TotalDurationMS:   totalDurationMS,
		ArtworkURL:        artworkURL,
		Explicit:          explicit,
		EditionHints:      normalize.EditionHints(title),
		Tracks:            tracks,
	}
}

func officialTracks(resource map[string]any) []model.CanonicalTrack {
	relationships := officialMap(resource, "relationships")
	tracksResource := officialMap(relationships, "tracks")
	data, _ := tracksResource["data"].([]any)
	tracks := make([]model.CanonicalTrack, 0, len(data))
	for _, item := range data {
		trackResource, ok := item.(map[string]any)
		if !ok {
			continue
		}
		attributes := officialMap(trackResource, "attributes")
		title := officialString(attributes, "name")
		artist := officialString(attributes, "artistName")
		tracks = append(tracks, model.CanonicalTrack{
			DiscNumber:      officialInt(attributes, "discNumber"),
			TrackNumber:     officialInt(attributes, "trackNumber"),
			Title:           title,
			NormalizedTitle: normalize.Text(title),
			DurationMS:      officialInt(attributes, "durationInMillis"),
			ISRC:            officialString(attributes, "isrc"),
			Artists:         nonEmptyArtistList(artist),
		})
	}
	return tracks
}

func parseOfficialAlbumURL(raw string) *model.ParsedAlbumURL {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parsed, err := parse.AppleMusicAlbumURL(raw)
	if err != nil {
		return nil
	}
	return parsed
}

func officialArtworkURL(artwork map[string]any) string {
	template := officialString(artwork, "url")
	if template == "" {
		return ""
	}
	replacer := strings.NewReplacer("{w}", "1000", "{h}", "1000")
	return replacer.Replace(template)
}

func officialMap(root map[string]any, key string) map[string]any {
	value, _ := root[key].(map[string]any)
	return value
}

func officialString(root map[string]any, key string) string {
	value, _ := root[key].(string)
	return strings.TrimSpace(value)
}

func officialInt(root map[string]any, key string) int {
	switch value := root[key].(type) {
	case float64:
		return int(value)
	case int:
		return value
	default:
		return 0
	}
}

func nonEmptyArtistList(artist string) []string {
	artist = strings.TrimSpace(artist)
	if artist == "" {
		return nil
	}
	return []string{artist}
}

func toCandidateAlbum(album model.CanonicalAlbum) model.CandidateAlbum {
	return model.CandidateAlbum{
		CanonicalAlbum: album,
		CandidateID:    album.SourceID,
		MatchURL:       album.SourceURL,
	}
}

func appendUniqueString(values []string, seen map[string]struct{}, value string) []string {
	if value == "" {
		return values
	}
	if _, ok := seen[value]; ok {
		return values
	}
	seen[value] = struct{}{}
	return append(values, value)
}
