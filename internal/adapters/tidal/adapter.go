package tidal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/normalize"
	"github.com/xmbshwll/ariadne/internal/parse"
)

const (
	defaultAPIBaseURL  = "https://openapi.tidal.com/v2"
	defaultAuthBaseURL = "https://auth.tidal.com/v1"
	defaultCountryCode = "US"
	searchLimit        = 5
)

var (
	ErrCredentialsNotConfigured = errors.New("tidal credentials not configured")

	errUnexpectedTIDALService     = errors.New("unexpected tidal service")
	errTIDALAlbumNotFound         = errors.New("tidal album not found")
	errUnexpectedTIDALAPIStatus   = errors.New("unexpected api status")
	errUnexpectedTIDALTokenStatus = errors.New("unexpected token status")
	errEmptyTIDALAccessToken      = errors.New("empty tidal access token")
)

type Option func(*Adapter)

func WithCredentials(clientID string, clientSecret string) Option {
	return func(adapter *Adapter) {
		adapter.clientID = strings.TrimSpace(clientID)
		adapter.clientSecret = strings.TrimSpace(clientSecret)
	}
}

func WithAPIBaseURL(baseURL string) Option {
	return func(adapter *Adapter) {
		adapter.apiBaseURL = strings.TrimRight(baseURL, "/")
	}
}

func WithAuthBaseURL(baseURL string) Option {
	return func(adapter *Adapter) {
		adapter.authBaseURL = strings.TrimRight(baseURL, "/")
	}
}

func WithDefaultCountryCode(countryCode string) Option {
	return func(adapter *Adapter) {
		adapter.defaultCountryCode = normalizeCountryCode(countryCode)
	}
}

type Adapter struct {
	client             *http.Client
	clientID           string
	clientSecret       string
	apiBaseURL         string
	authBaseURL        string
	defaultCountryCode string

	tokenMu sync.Mutex
	token   cachedToken
}

type cachedToken struct {
	accessToken string
	expiresAt   time.Time
}

func New(client *http.Client, opts ...Option) *Adapter {
	if client == nil {
		client = http.DefaultClient
	}
	adapter := &Adapter{
		client:             client,
		apiBaseURL:         defaultAPIBaseURL,
		authBaseURL:        defaultAuthBaseURL,
		defaultCountryCode: defaultCountryCode,
	}
	for _, opt := range opts {
		opt(adapter)
	}
	return adapter
}

func (a *Adapter) Service() model.ServiceName {
	return model.ServiceTIDAL
}

func (a *Adapter) ParseAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	parsed, err := parse.TIDALAlbumURL(raw)
	if err != nil {
		return nil, fmt.Errorf("parse tidal album url: %w", err)
	}
	return parsed, nil
}

func (a *Adapter) ParseSongURL(raw string) (*model.ParsedAlbumURL, error) {
	parsed, err := parse.TIDALSongURL(raw)
	if err != nil {
		return nil, fmt.Errorf("parse tidal song url: %w", err)
	}
	return parsed, nil
}

func (a *Adapter) FetchAlbum(ctx context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalAlbum, error) {
	if parsed.Service != model.ServiceTIDAL {
		return nil, fmt.Errorf("%w: %s", errUnexpectedTIDALService, parsed.Service)
	}
	return a.fetchAlbumByID(ctx, parsed.ID, parsed.CanonicalURL, parsed.RegionHint)
}

func (a *Adapter) SearchByUPC(ctx context.Context, upc string) ([]model.CandidateAlbum, error) {
	upc = strings.TrimSpace(upc)
	if upc == "" {
		return nil, nil
	}
	if !a.hasCredentials() {
		return nil, ErrCredentialsNotConfigured
	}

	endpoint := fmt.Sprintf("%s/albums?countryCode=%s&filter[barcodeId]=%s", a.apiBaseURL, url.QueryEscape(a.defaultCountryCode), url.QueryEscape(upc))
	var document apiDocument
	if err := a.getAPIJSON(ctx, endpoint, &document); err != nil {
		return nil, fmt.Errorf("tidal search by upc: %w", err)
	}
	resources := documentData(document)
	results := make([]model.CandidateAlbum, 0, min(len(resources), searchLimit))
	for _, resource := range resources {
		canonical, err := a.fetchAlbumByID(ctx, resource.ID, canonicalAlbumURL(resource.ID), "")
		if err != nil {
			return nil, fmt.Errorf("hydrate tidal album %s from upc: %w", resource.ID, err)
		}
		results = append(results, toCandidateAlbum(*canonical))
		if len(results) >= searchLimit {
			break
		}
	}
	return results, nil
}

func (a *Adapter) SearchByISRC(ctx context.Context, isrcs []string) ([]model.CandidateAlbum, error) {
	if !a.hasCredentials() {
		return nil, ErrCredentialsNotConfigured
	}

	results := make([]model.CandidateAlbum, 0, len(isrcs))
	seen := make(map[string]struct{}, len(isrcs))
	for _, isrc := range isrcs {
		isrc = strings.TrimSpace(isrc)
		if isrc == "" {
			continue
		}
		endpoint := fmt.Sprintf("%s/tracks?countryCode=%s&filter[isrc]=%s&include=%s", a.apiBaseURL, url.QueryEscape(a.defaultCountryCode), url.QueryEscape(isrc), url.QueryEscape("albums"))
		var document apiDocument
		if err := a.getAPIJSON(ctx, endpoint, &document); err != nil {
			return nil, fmt.Errorf("tidal search by isrc %s: %w", isrc, err)
		}
		albumIDs := albumIDsFromTrackDocument(document)
		for _, albumID := range albumIDs {
			if _, ok := seen[albumID]; ok {
				continue
			}
			seen[albumID] = struct{}{}
			canonical, err := a.fetchAlbumByID(ctx, albumID, canonicalAlbumURL(albumID), "")
			if err != nil {
				return nil, fmt.Errorf("hydrate tidal album %s from isrc %s: %w", albumID, isrc, err)
			}
			results = append(results, toCandidateAlbum(*canonical))
			if len(results) >= searchLimit {
				return results, nil
			}
		}
	}
	return results, nil
}

func (a *Adapter) SearchByMetadata(ctx context.Context, album model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
	if !a.hasCredentials() {
		return nil, ErrCredentialsNotConfigured
	}
	query := metadataQuery(album)
	if query == "" {
		return nil, nil
	}
	countryCode := a.countryCodeFor(album.RegionHint)
	endpoint := fmt.Sprintf("%s/searchResults/%s/relationships/albums?countryCode=%s", a.apiBaseURL, url.PathEscape(query), url.QueryEscape(countryCode))
	var document apiDocument
	if err := a.getAPIJSON(ctx, endpoint, &document); err != nil {
		return nil, fmt.Errorf("tidal search by metadata: %w", err)
	}
	resources := documentData(document)
	results := make([]model.CandidateAlbum, 0, min(len(resources), searchLimit))
	for _, resource := range resources {
		canonical, err := a.fetchAlbumByID(ctx, resource.ID, canonicalAlbumURL(resource.ID), album.RegionHint)
		if err != nil {
			return nil, fmt.Errorf("hydrate tidal album %s from metadata: %w", resource.ID, err)
		}
		results = append(results, toCandidateAlbum(*canonical))
		if len(results) >= searchLimit {
			break
		}
	}
	return results, nil
}

func (a *Adapter) FetchSong(ctx context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalSong, error) {
	if parsed.Service != model.ServiceTIDAL {
		return nil, fmt.Errorf("%w: %s", errUnexpectedTIDALService, parsed.Service)
	}
	return a.fetchSongByID(ctx, parsed.ID, parsed.CanonicalURL, parsed.RegionHint)
}

func (a *Adapter) SearchSongByISRC(ctx context.Context, isrc string) ([]model.CandidateSong, error) {
	if !a.hasCredentials() {
		return nil, ErrCredentialsNotConfigured
	}
	isrc = strings.TrimSpace(isrc)
	if isrc == "" {
		return nil, nil
	}

	endpoint := fmt.Sprintf("%s/tracks?countryCode=%s&filter[isrc]=%s&include=%s", a.apiBaseURL, url.QueryEscape(a.defaultCountryCode), url.QueryEscape(isrc), url.QueryEscape("artists,albums,coverArt"))
	var document apiDocument
	if err := a.getAPIJSON(ctx, endpoint, &document); err != nil {
		return nil, fmt.Errorf("tidal song search by isrc %s: %w", isrc, err)
	}
	resources := documentData(document)
	results := make([]model.CandidateSong, 0, min(len(resources), searchLimit))
	for _, resource := range resources {
		canonical, err := a.fetchSongByID(ctx, resource.ID, canonicalTrackURL(resource.ID), "")
		if err != nil {
			return nil, fmt.Errorf("hydrate tidal song %s from isrc: %w", resource.ID, err)
		}
		results = append(results, toCandidateSong(*canonical))
		if len(results) >= searchLimit {
			break
		}
	}
	return results, nil
}

func (a *Adapter) SearchSongByMetadata(ctx context.Context, song model.CanonicalSong) ([]model.CandidateSong, error) {
	if !a.hasCredentials() {
		return nil, ErrCredentialsNotConfigured
	}
	query := songMetadataQuery(song)
	if query == "" {
		return nil, nil
	}
	countryCode := a.countryCodeFor(song.RegionHint)
	endpoint := fmt.Sprintf("%s/searchResults/%s/relationships/tracks?countryCode=%s", a.apiBaseURL, url.PathEscape(query), url.QueryEscape(countryCode))
	var document apiDocument
	if err := a.getAPIJSON(ctx, endpoint, &document); err != nil {
		return nil, fmt.Errorf("tidal song search by metadata: %w", err)
	}
	resources := documentData(document)
	results := make([]model.CandidateSong, 0, min(len(resources), searchLimit))
	for _, resource := range resources {
		canonical, err := a.fetchSongByID(ctx, resource.ID, canonicalTrackURL(resource.ID), song.RegionHint)
		if err != nil {
			return nil, fmt.Errorf("hydrate tidal song %s from metadata: %w", resource.ID, err)
		}
		results = append(results, toCandidateSong(*canonical))
		if len(results) >= searchLimit {
			break
		}
	}
	return results, nil
}

func (a *Adapter) fetchAlbumByID(ctx context.Context, albumID string, canonicalURL string, regionHint string) (*model.CanonicalAlbum, error) {
	var document apiDocument
	countryCode := a.countryCodeFor(regionHint)
	endpoint := fmt.Sprintf("%s/albums/%s?countryCode=%s&include=%s", a.apiBaseURL, url.PathEscape(albumID), url.QueryEscape(countryCode), url.QueryEscape("artists,items,coverArt"))
	if err := a.getAPIJSON(ctx, endpoint, &document); err != nil {
		return nil, err
	}
	resource := firstDataResource(document)
	if resource == nil {
		return nil, fmt.Errorf("%w: %s", errTIDALAlbumNotFound, albumID)
	}
	return toCanonicalAlbum(*resource, document.Included, canonicalURL, regionHint), nil
}

func (a *Adapter) fetchSongByID(ctx context.Context, trackID string, canonicalURL string, regionHint string) (*model.CanonicalSong, error) {
	var document apiDocument
	countryCode := a.countryCodeFor(regionHint)
	endpoint := fmt.Sprintf("%s/tracks/%s?countryCode=%s&include=%s", a.apiBaseURL, url.PathEscape(trackID), url.QueryEscape(countryCode), url.QueryEscape("artists,albums,coverArt"))
	if err := a.getAPIJSON(ctx, endpoint, &document); err != nil {
		return nil, err
	}
	resource := firstDataResource(document)
	if resource == nil {
		return nil, fmt.Errorf("%w: %s", errTIDALAlbumNotFound, trackID)
	}
	return toCanonicalSong(*resource, document.Included, canonicalURL, regionHint), nil
}

func (a *Adapter) getAPIJSON(ctx context.Context, endpoint string, target any) error {
	token, err := a.accessToken(ctx)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build api request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.api+json")
	req.Header.Set("User-Agent", "ariadne/0.1 (+https://github.com/xmbshwll/ariadne)")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("execute api request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("%w %d: %s", errUnexpectedTIDALAPIStatus, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode api response: %w", err)
	}
	return nil
}

func (a *Adapter) accessToken(ctx context.Context) (string, error) {
	if !a.hasCredentials() {
		return "", ErrCredentialsNotConfigured
	}
	a.tokenMu.Lock()
	defer a.tokenMu.Unlock()
	if a.token.accessToken != "" && time.Now().Before(a.token.expiresAt) {
		return a.token.accessToken, nil
	}

	form := url.Values{}
	form.Set("client_id", a.clientID)
	form.Set("client_secret", a.clientSecret)
	form.Set("grant_type", "client_credentials")
	endpoint := a.authBaseURL + "/oauth2/token"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("User-Agent", "ariadne/0.1 (+https://github.com/xmbshwll/ariadne)")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute token request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read token response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w %d: %s", errUnexpectedTIDALTokenStatus, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var token tokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}
	if strings.TrimSpace(token.AccessToken) == "" {
		return "", errEmptyTIDALAccessToken
	}
	a.token = cachedToken{
		accessToken: token.AccessToken,
		expiresAt:   time.Now().Add(time.Duration(token.ExpiresIn-30) * time.Second),
	}
	return a.token.accessToken, nil
}

func (a *Adapter) hasCredentials() bool {
	return strings.TrimSpace(a.clientID) != "" && strings.TrimSpace(a.clientSecret) != ""
}

func (a *Adapter) countryCodeFor(regionHint string) string {
	countryCode := normalizeCountryCode(regionHint)
	if countryCode == "" {
		return a.defaultCountryCode
	}
	return countryCode
}

func normalizeCountryCode(value string) string {
	countryCode := strings.ToUpper(strings.TrimSpace(value))
	if len(countryCode) != 2 {
		return ""
	}
	return countryCode
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

func firstDataResource(document apiDocument) *apiResource {
	resources := documentData(document)
	if len(resources) == 0 {
		return nil
	}
	resource := resources[0]
	return &resource
}

func documentData(document apiDocument) []apiResource {
	switch data := document.Data.(type) {
	case nil:
		return nil
	case map[string]any:
		content, _ := json.Marshal(data)
		var resource apiResource
		if err := json.Unmarshal(content, &resource); err != nil {
			return nil
		}
		return []apiResource{resource}
	case []any:
		resources := make([]apiResource, 0, len(data))
		for _, item := range data {
			content, _ := json.Marshal(item)
			var resource apiResource
			if err := json.Unmarshal(content, &resource); err != nil {
				continue
			}
			resources = append(resources, resource)
		}
		return resources
	default:
		return nil
	}
}

func albumIDsFromTrackDocument(document apiDocument) []string {
	resources := documentData(document)
	seen := make(map[string]struct{})
	ids := make([]string, 0, len(resources))
	for _, included := range document.Included {
		if included.Type != "albums" {
			continue
		}
		if included.ID == "" {
			continue
		}
		if _, ok := seen[included.ID]; ok {
			continue
		}
		seen[included.ID] = struct{}{}
		ids = append(ids, included.ID)
	}
	if len(ids) > 0 {
		return ids
	}
	for _, resource := range resources {
		for _, relation := range resource.Relationships.Albums.Data {
			if relation.ID == "" {
				continue
			}
			if _, ok := seen[relation.ID]; ok {
				continue
			}
			seen[relation.ID] = struct{}{}
			ids = append(ids, relation.ID)
		}
	}
	return ids
}

func toCanonicalAlbum(resource apiResource, included []apiResource, canonicalURL string, regionHint string) *model.CanonicalAlbum {
	artistNames := includedArtistNames(included, resource.Relationships.Artists.Data)
	tracks := tracksFromIncluded(included, resource.Relationships.Items.Data, artistNames)
	artworkURL := artworkURLFromIncluded(included, resource.Relationships.CoverArt.Data)
	trackCount := resource.Attributes.NumberOfItems
	if trackCount == 0 {
		trackCount = len(tracks)
	}
	if canonicalURL == "" {
		canonicalURL = canonicalAlbumURL(resource.ID)
	}
	return &model.CanonicalAlbum{
		Service:           model.ServiceTIDAL,
		SourceID:          resource.ID,
		SourceURL:         canonicalURL,
		RegionHint:        strings.ToUpper(strings.TrimSpace(regionHint)),
		Title:             resource.Attributes.Title,
		NormalizedTitle:   normalize.Text(resource.Attributes.Title),
		Artists:           artistNames,
		NormalizedArtists: normalize.Artists(artistNames),
		ReleaseDate:       resource.Attributes.ReleaseDate,
		Label:             resource.Attributes.Copyright.Text,
		UPC:               firstNonEmpty(resource.Attributes.BarcodeID, resource.Attributes.UPC),
		TrackCount:        trackCount,
		TotalDurationMS:   parseISODurationMilliseconds(resource.Attributes.Duration),
		ArtworkURL:        artworkURL,
		Explicit:          resource.Attributes.Explicit,
		EditionHints:      normalize.EditionHints(resource.Attributes.Title),
		Tracks:            tracks,
	}
}

func toCanonicalSong(resource apiResource, included []apiResource, canonicalURL string, regionHint string) *model.CanonicalSong {
	artistNames := includedArtistNames(included, resource.Relationships.Artists.Data)
	albumResource := firstRelatedResource(included, resource.Relationships.Albums.Data, "albums")
	albumTitle := ""
	albumNormalizedTitle := ""
	albumArtists := []string{}
	albumNormalizedArtists := []string{}
	releaseDate := resource.Attributes.ReleaseDate
	artworkURL := ""
	if albumResource != nil {
		albumTitle = albumResource.Attributes.Title
		albumNormalizedTitle = normalize.Text(albumTitle)
		albumArtists = includedArtistNames(included, albumResource.Relationships.Artists.Data)
		albumNormalizedArtists = normalize.Artists(albumArtists)
		if releaseDate == "" {
			releaseDate = albumResource.Attributes.ReleaseDate
		}
		artworkURL = artworkURLFromIncluded(included, albumResource.Relationships.CoverArt.Data)
	}
	if canonicalURL == "" {
		canonicalURL = canonicalTrackURL(resource.ID)
	}
	return &model.CanonicalSong{
		Service:                model.ServiceTIDAL,
		SourceID:               resource.ID,
		SourceURL:              canonicalURL,
		RegionHint:             strings.ToUpper(strings.TrimSpace(regionHint)),
		Title:                  resource.Attributes.Title,
		NormalizedTitle:        normalize.Text(resource.Attributes.Title),
		Artists:                artistNames,
		NormalizedArtists:      normalize.Artists(artistNames),
		DurationMS:             parseISODurationMilliseconds(resource.Attributes.Duration),
		ISRC:                   resource.Attributes.ISRC,
		Explicit:               resource.Attributes.Explicit,
		DiscNumber:             firstTrackVolumeNumber(resource.Relationships.Albums.Data),
		TrackNumber:            firstTrackNumber(resource.Relationships.Albums.Data),
		AlbumID:                firstRelatedID(resource.Relationships.Albums.Data, "albums"),
		AlbumTitle:             albumTitle,
		AlbumNormalizedTitle:   albumNormalizedTitle,
		AlbumArtists:           albumArtists,
		AlbumNormalizedArtists: albumNormalizedArtists,
		ReleaseDate:            releaseDate,
		ArtworkURL:             artworkURL,
		EditionHints:           normalize.EditionHints(resource.Attributes.Title),
	}
}

func includedArtistNames(included []apiResource, relations []relationshipData) []string {
	resourceByID := make(map[string]apiResource, len(included))
	for _, resource := range included {
		resourceByID[resource.ID] = resource
	}
	results := make([]string, 0, len(relations))
	seen := make(map[string]struct{}, len(relations))
	for _, relation := range relations {
		if relation.Type != "artists" {
			continue
		}
		resource, ok := resourceByID[relation.ID]
		if !ok {
			continue
		}
		name := firstNonEmpty(resource.Attributes.Name, resource.Attributes.Title)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		results = append(results, name)
	}
	return results
}

func firstRelatedResource(included []apiResource, relations []relationshipData, typ string) *apiResource {
	resourceByID := make(map[string]apiResource, len(included))
	for _, resource := range included {
		resourceByID[resource.ID] = resource
	}
	for _, relation := range relations {
		if relation.Type != typ {
			continue
		}
		resource, ok := resourceByID[relation.ID]
		if !ok {
			continue
		}
		relatedResource := resource
		return &relatedResource
	}
	return nil
}

func firstRelatedID(relations []relationshipData, typ string) string {
	for _, relation := range relations {
		if relation.Type == typ && relation.ID != "" {
			return relation.ID
		}
	}
	return ""
}

func firstTrackNumber(relations []relationshipData) int {
	for _, relation := range relations {
		if relation.Meta.TrackNumber > 0 {
			return relation.Meta.TrackNumber
		}
	}
	return 0
}

func firstTrackVolumeNumber(relations []relationshipData) int {
	for _, relation := range relations {
		if relation.Meta.VolumeNumber > 0 {
			return relation.Meta.VolumeNumber
		}
	}
	return 0
}

func tracksFromIncluded(included []apiResource, relations []relationshipData, fallbackArtists []string) []model.CanonicalTrack {
	resourceByID := make(map[string]apiResource, len(included))
	for _, resource := range included {
		resourceByID[resource.ID] = resource
	}
	tracks := make([]model.CanonicalTrack, 0, len(relations))
	for _, relation := range relations {
		if relation.Type != "tracks" {
			continue
		}
		resource, ok := resourceByID[relation.ID]
		if !ok {
			continue
		}
		trackArtists := includedArtistNames(included, resource.Relationships.Artists.Data)
		if len(trackArtists) == 0 {
			trackArtists = append([]string(nil), fallbackArtists...)
		}
		tracks = append(tracks, model.CanonicalTrack{
			DiscNumber:      relation.Meta.VolumeNumber,
			TrackNumber:     relation.Meta.TrackNumber,
			Title:           resource.Attributes.Title,
			NormalizedTitle: normalize.Text(resource.Attributes.Title),
			DurationMS:      parseISODurationMilliseconds(resource.Attributes.Duration),
			ISRC:            resource.Attributes.ISRC,
			Artists:         trackArtists,
		})
	}
	return tracks
}

func artworkURLFromIncluded(included []apiResource, relations []relationshipData) string {
	resourceByID := make(map[string]apiResource, len(included))
	for _, resource := range included {
		resourceByID[resource.ID] = resource
	}
	for _, relation := range relations {
		if relation.Type != "artworks" {
			continue
		}
		resource, ok := resourceByID[relation.ID]
		if !ok {
			continue
		}
		files := append([]resourceFile(nil), resource.Attributes.Files...)
		sort.Slice(files, func(i, j int) bool {
			return files[i].Meta.Width > files[j].Meta.Width
		})
		for _, file := range files {
			if file.Href != "" {
				return file.Href
			}
		}
	}
	return ""
}

func parseISODurationMilliseconds(value string) int {
	if value == "" {
		return 0
	}
	duration, err := time.ParseDuration(strings.ToLower(strings.TrimPrefix(strings.TrimPrefix(value, "P"), "T")))
	if err == nil {
		return int(duration.Milliseconds())
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

func canonicalAlbumURL(albumID string) string {
	return "https://tidal.com/album/" + albumID
}

func canonicalTrackURL(trackID string) string {
	return "https://tidal.com/track/" + trackID
}

func toCandidateAlbum(album model.CanonicalAlbum) model.CandidateAlbum {
	return model.CandidateAlbum{CanonicalAlbum: album, CandidateID: album.SourceID, MatchURL: album.SourceURL}
}

func toCandidateSong(song model.CanonicalSong) model.CandidateSong {
	return model.CandidateSong{CanonicalSong: song, CandidateID: song.SourceID, MatchURL: song.SourceURL}
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
