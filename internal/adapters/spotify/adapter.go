package spotify

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/normalize"
	"github.com/xmbshwll/ariadne/internal/parse"
)

const (
	defaultWebBaseURL  = "https://open.spotify.com"
	defaultAPIBaseURL  = "https://api.spotify.com/v1"
	defaultAuthBaseURL = "https://accounts.spotify.com/api"
	searchLimit        = 5
)

var (
	initialStatePattern = regexp.MustCompile(`<script id="initialState" type="text/plain">([^<]+)</script>`)

	errUnexpectedSpotifyService     = errors.New("unexpected spotify service")
	errUnexpectedSpotifyStatus      = errors.New("unexpected spotify status")
	errSpotifyAlbumNotFound         = errors.New("spotify album not found")
	errSpotifyTrackNotFound         = errors.New("spotify track not found")
	errUnexpectedSpotifyAPIStatus   = errors.New("unexpected spotify api status")
	errUnexpectedSpotifyTokenStatus = errors.New("unexpected spotify token status")
	errEmptySpotifyAccessToken      = errors.New("empty spotify access token")
	errInitialStateScriptNotFound   = errors.New("initial state script not found")

	// ErrCredentialsNotConfigured indicates that a Web API operation requires Spotify credentials.
	ErrCredentialsNotConfigured = errors.New("spotify credentials not configured")
)

// Option configures the Spotify adapter.
type Option func(*Adapter)

// WithCredentials sets Spotify client credentials explicitly.
func WithCredentials(clientID string, clientSecret string) Option {
	return func(adapter *Adapter) {
		adapter.clientID = clientID
		adapter.clientSecret = clientSecret
	}
}

// WithAPIBaseURL overrides the Spotify Web API base URL.
func WithAPIBaseURL(baseURL string) Option {
	return func(adapter *Adapter) {
		adapter.apiBaseURL = strings.TrimRight(baseURL, "/")
	}
}

// WithAuthBaseURL overrides the Spotify auth API base URL.
func WithAuthBaseURL(baseURL string) Option {
	return func(adapter *Adapter) {
		adapter.authBaseURL = strings.TrimRight(baseURL, "/")
	}
}

// WithWebBaseURL overrides the Spotify web base URL used for bootstrap fetches.
func WithWebBaseURL(baseURL string) Option {
	return func(adapter *Adapter) {
		adapter.webBaseURL = strings.TrimRight(baseURL, "/")
	}
}

// Adapter implements Spotify source and target operations.
type Adapter struct {
	client       *http.Client
	clientID     string
	clientSecret string
	apiBaseURL   string
	authBaseURL  string
	webBaseURL   string

	tokenMu sync.Mutex
	token   cachedToken
}

// New creates a Spotify adapter.
func New(client *http.Client, opts ...Option) *Adapter {
	if client == nil {
		client = http.DefaultClient
	}
	adapter := &Adapter{
		client:      client,
		apiBaseURL:  defaultAPIBaseURL,
		authBaseURL: defaultAuthBaseURL,
		webBaseURL:  defaultWebBaseURL,
	}
	for _, opt := range opts {
		opt(adapter)
	}
	return adapter
}

// Service returns the service implemented by this adapter.
func (a *Adapter) Service() model.ServiceName {
	return model.ServiceSpotify
}

// ParseAlbumURL parses a Spotify album URL.
func (a *Adapter) ParseAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	parsed, err := parse.SpotifyAlbumURL(raw)
	if err != nil {
		return nil, fmt.Errorf("parse spotify album url: %w", err)
	}
	return parsed, nil
}

// ParseSongURL parses a Spotify track URL.
func (a *Adapter) ParseSongURL(raw string) (*model.ParsedAlbumURL, error) {
	parsed, err := parse.SpotifySongURL(raw)
	if err != nil {
		return nil, fmt.Errorf("parse spotify song url: %w", err)
	}
	return parsed, nil
}

// FetchAlbum loads a Spotify album via the Web API when credentials are configured,
// otherwise falls back to the public album page bootstrap.
func (a *Adapter) FetchAlbum(ctx context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalAlbum, error) {
	if parsed.Service != model.ServiceSpotify {
		return nil, fmt.Errorf("%w: %s", errUnexpectedSpotifyService, parsed.Service)
	}

	if a.hasCredentials() {
		album, err := a.fetchAlbumAPI(ctx, parsed.ID)
		if err == nil {
			return toCanonicalAlbumAPI(parsed.CanonicalURL, album), nil
		}
	}

	return a.fetchAlbumBootstrap(ctx, parsed)
}

// SearchByUPC searches Spotify albums by UPC via the Web API.
func (a *Adapter) SearchByUPC(ctx context.Context, upc string) ([]model.CandidateAlbum, error) {
	if strings.TrimSpace(upc) == "" {
		return nil, nil
	}
	if !a.hasCredentials() {
		return nil, ErrCredentialsNotConfigured
	}

	endpoint := fmt.Sprintf("%s/search?q=%s&type=album&limit=%d", a.apiBaseURL, url.QueryEscape("upc:"+upc), searchLimit)
	var response apiAlbumSearchResponse
	if err := a.getAPIJSON(ctx, endpoint, &response); err != nil {
		return nil, fmt.Errorf("spotify search by upc: %w", err)
	}
	return a.hydrateAlbumCandidates(ctx, response.Albums.Items)
}

// SearchByISRC searches Spotify track results by ISRC, then hydrates the owning albums.
func (a *Adapter) SearchByISRC(ctx context.Context, isrcs []string) ([]model.CandidateAlbum, error) {
	if !a.hasCredentials() {
		return nil, ErrCredentialsNotConfigured
	}

	albumIDs := make([]string, 0, len(isrcs))
	seen := make(map[string]struct{}, len(isrcs))
	for _, isrc := range isrcs {
		isrc = strings.TrimSpace(isrc)
		if isrc == "" {
			continue
		}

		endpoint := fmt.Sprintf("%s/search?q=%s&type=track&limit=%d", a.apiBaseURL, url.QueryEscape("isrc:"+isrc), 1)
		var response apiTrackSearchResponse
		if err := a.getAPIJSON(ctx, endpoint, &response); err != nil {
			return nil, fmt.Errorf("spotify search by isrc %s: %w", isrc, err)
		}
		for _, item := range response.Tracks.Items {
			if item.Album.ID == "" {
				continue
			}
			if _, ok := seen[item.Album.ID]; ok {
				continue
			}
			seen[item.Album.ID] = struct{}{}
			albumIDs = append(albumIDs, item.Album.ID)
			if len(albumIDs) >= searchLimit {
				return a.hydrateAlbumCandidates(ctx, albumIDsToSummaries(albumIDs))
			}
		}
	}
	return a.hydrateAlbumCandidates(ctx, albumIDsToSummaries(albumIDs))
}

// SearchByMetadata searches Spotify albums by title and artist metadata.
func (a *Adapter) SearchByMetadata(ctx context.Context, album model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
	if !a.hasCredentials() {
		return nil, ErrCredentialsNotConfigured
	}

	queries := metadataQueries(album)
	if len(queries) == 0 {
		return nil, nil
	}

	items := make([]apiAlbumSummary, 0, searchLimit)
	seen := make(map[string]struct{}, searchLimit)
	for _, query := range queries {
		endpoint := fmt.Sprintf("%s/search?q=%s&type=album&limit=%d", a.apiBaseURL, url.QueryEscape(query), searchLimit)
		var response apiAlbumSearchResponse
		if err := a.getAPIJSON(ctx, endpoint, &response); err != nil {
			return nil, fmt.Errorf("spotify search by metadata %q: %w", query, err)
		}
		for _, item := range response.Albums.Items {
			if item.ID == "" {
				continue
			}
			if _, ok := seen[item.ID]; ok {
				continue
			}
			seen[item.ID] = struct{}{}
			items = append(items, item)
			if len(items) >= searchLimit {
				return a.hydrateAlbumCandidates(ctx, items)
			}
		}
	}
	return a.hydrateAlbumCandidates(ctx, items)
}

// FetchSong loads a Spotify track via the Web API.
func (a *Adapter) FetchSong(ctx context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalSong, error) {
	if parsed.Service != model.ServiceSpotify {
		return nil, fmt.Errorf("%w: %s", errUnexpectedSpotifyService, parsed.Service)
	}
	if !a.hasCredentials() {
		return nil, ErrCredentialsNotConfigured
	}

	track, err := a.fetchTrackAPI(ctx, parsed.ID)
	if err != nil {
		return nil, fmt.Errorf("spotify fetch song api %s: %w", parsed.ID, err)
	}
	return toCanonicalSongAPI(parsed.CanonicalURL, track), nil
}

// SearchSongByISRC searches Spotify tracks by ISRC.
func (a *Adapter) SearchSongByISRC(ctx context.Context, isrc string) ([]model.CandidateSong, error) {
	if strings.TrimSpace(isrc) == "" {
		return nil, nil
	}
	if !a.hasCredentials() {
		return nil, ErrCredentialsNotConfigured
	}

	endpoint := fmt.Sprintf("%s/search?q=%s&type=track&limit=%d", a.apiBaseURL, url.QueryEscape("isrc:"+strings.TrimSpace(isrc)), searchLimit)
	var response apiTrackSearchResponse
	if err := a.getAPIJSON(ctx, endpoint, &response); err != nil {
		return nil, fmt.Errorf("spotify song search by isrc %s: %w", isrc, err)
	}
	return a.hydrateSongCandidates(ctx, response.Tracks.Items)
}

// SearchSongByMetadata searches Spotify tracks by title and artist metadata.
func (a *Adapter) SearchSongByMetadata(ctx context.Context, song model.CanonicalSong) ([]model.CandidateSong, error) {
	queries := songMetadataQueries(song)
	if len(queries) == 0 {
		return nil, nil
	}
	if !a.hasCredentials() {
		return nil, ErrCredentialsNotConfigured
	}

	items := make([]apiTrackSearchItem, 0, searchLimit)
	seen := make(map[string]struct{}, searchLimit)
	for _, query := range queries {
		endpoint := fmt.Sprintf("%s/search?q=%s&type=track&limit=%d", a.apiBaseURL, url.QueryEscape(query), searchLimit)
		var response apiTrackSearchResponse
		if err := a.getAPIJSON(ctx, endpoint, &response); err != nil {
			return nil, fmt.Errorf("spotify song search by metadata %q: %w", query, err)
		}
		for _, item := range response.Tracks.Items {
			if item.ID == "" {
				continue
			}
			if _, ok := seen[item.ID]; ok {
				continue
			}
			seen[item.ID] = struct{}{}
			items = append(items, item)
			if len(items) >= searchLimit {
				return a.hydrateSongCandidates(ctx, items)
			}
		}
	}
	return a.hydrateSongCandidates(ctx, items)
}

func (a *Adapter) fetchAlbumAPI(ctx context.Context, albumID string) (*apiAlbumResponse, error) {
	var album apiAlbumResponse
	endpoint := a.apiBaseURL + "/albums/" + albumID
	if err := a.getAPIJSON(ctx, endpoint, &album); err != nil {
		return nil, fmt.Errorf("spotify fetch album api %s: %w", albumID, err)
	}
	if err := a.hydrateAlbumTrackDetails(ctx, &album); err != nil {
		return nil, fmt.Errorf("spotify hydrate track details %s: %w", albumID, err)
	}
	return &album, nil
}

func (a *Adapter) hydrateAlbumTrackDetails(ctx context.Context, album *apiAlbumResponse) error {
	trackIDs := make([]string, 0, len(album.Tracks.Items))
	for _, track := range album.Tracks.Items {
		if track.ID == "" {
			continue
		}
		trackIDs = append(trackIDs, track.ID)
	}
	if len(trackIDs) == 0 {
		return nil
	}

	trackDetails, err := a.fetchTrackDetailsAPI(ctx, trackIDs)
	if err != nil {
		return err
	}
	byID := make(map[string]apiTrack, len(trackDetails))
	for _, track := range trackDetails {
		if track.ID == "" {
			continue
		}
		byID[track.ID] = track
	}
	for i := range album.Tracks.Items {
		track := album.Tracks.Items[i]
		detail, ok := byID[track.ID]
		if !ok {
			continue
		}
		album.Tracks.Items[i].ExternalIDs = detail.ExternalIDs
		if len(detail.Artists) > 0 {
			album.Tracks.Items[i].Artists = detail.Artists
		}
		if detail.DurationMS > 0 {
			album.Tracks.Items[i].DurationMS = detail.DurationMS
		}
		album.Tracks.Items[i].Explicit = detail.Explicit
	}
	return nil
}

func (a *Adapter) fetchTrackAPI(ctx context.Context, trackID string) (*apiTrack, error) {
	tracks, err := a.fetchTrackDetailsAPI(ctx, []string{trackID})
	if err != nil {
		return nil, err
	}
	if len(tracks) == 0 {
		return nil, fmt.Errorf("%w: %s", errSpotifyTrackNotFound, trackID)
	}
	return &tracks[0], nil
}

func (a *Adapter) fetchTrackDetailsAPI(ctx context.Context, trackIDs []string) ([]apiTrack, error) {
	if len(trackIDs) == 0 {
		return nil, nil
	}

	tracks := make([]apiTrack, 0, len(trackIDs))
	for _, trackID := range trackIDs {
		if trackID == "" {
			continue
		}
		endpoint := a.apiBaseURL + "/tracks/" + trackID
		var track apiTrack
		if err := a.getAPIJSON(ctx, endpoint, &track); err != nil {
			return nil, err
		}
		tracks = append(tracks, track)
	}
	return tracks, nil
}

func (a *Adapter) fetchAlbumBootstrap(ctx context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalAlbum, error) {
	requestURL := parsed.CanonicalURL
	if parsed.CanonicalURL == "https://open.spotify.com/album/"+parsed.ID && a.webBaseURL != defaultWebBaseURL {
		requestURL = a.webBaseURL + "/album/" + parsed.ID
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build spotify request: %w", err)
	}
	req.Header.Set("User-Agent", "ariadne/0.1 (+https://github.com/xmbshwll/ariadne)")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute spotify request: %w", err)
	}

	body, err := io.ReadAll(resp.Body)
	closeErr := resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("read spotify response: %w", err)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close spotify response body: %w", closeErr)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d", errUnexpectedSpotifyStatus, resp.StatusCode)
	}

	payload, err := parseInitialState(body)
	if err != nil {
		return nil, fmt.Errorf("parse spotify initial state: %w", err)
	}

	entityKey := "spotify:album:" + parsed.ID
	album, ok := payload.Entities.Items[entityKey]
	if !ok {
		return nil, fmt.Errorf("%w: %s", errSpotifyAlbumNotFound, entityKey)
	}

	return toCanonicalAlbumBootstrap(parsed, album), nil
}

func (a *Adapter) hydrateAlbumCandidates(ctx context.Context, summaries []apiAlbumSummary) ([]model.CandidateAlbum, error) {
	results := make([]model.CandidateAlbum, 0, len(summaries))
	seen := make(map[string]struct{}, len(summaries))
	for _, summary := range summaries {
		if summary.ID == "" {
			continue
		}
		if _, ok := seen[summary.ID]; ok {
			continue
		}
		seen[summary.ID] = struct{}{}

		album, err := a.fetchAlbumAPI(ctx, summary.ID)
		if err != nil {
			return nil, fmt.Errorf("hydrate spotify album %s: %w", summary.ID, err)
		}
		canonical := toCanonicalAlbumAPI(canonicalAlbumURL(summary.ID), album)
		results = append(results, model.CandidateAlbum{
			CanonicalAlbum: *canonical,
			CandidateID:    canonical.SourceID,
			MatchURL:       canonical.SourceURL,
		})
	}
	return results, nil
}

func (a *Adapter) hydrateSongCandidates(ctx context.Context, items []apiTrackSearchItem) ([]model.CandidateSong, error) {
	results := make([]model.CandidateSong, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		if item.ID == "" {
			continue
		}
		if _, ok := seen[item.ID]; ok {
			continue
		}
		seen[item.ID] = struct{}{}

		track, err := a.fetchTrackAPI(ctx, item.ID)
		if err != nil {
			if errors.Is(err, errSpotifyTrackNotFound) {
				continue
			}
			return nil, fmt.Errorf("hydrate spotify track %s: %w", item.ID, err)
		}
		canonical := toCanonicalSongAPI(canonicalTrackURL(item.ID), track)
		results = append(results, model.CandidateSong{
			CanonicalSong: *canonical,
			CandidateID:   canonical.SourceID,
			MatchURL:      canonical.SourceURL,
		})
	}
	return results, nil
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
	req.Header.Set("User-Agent", "ariadne/0.1 (+https://github.com/xmbshwll/ariadne)")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("execute api request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("%w %d: %s", errUnexpectedSpotifyAPIStatus, resp.StatusCode, strings.TrimSpace(string(body)))
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

	if a.token.AccessToken != "" && time.Now().Before(a.token.ExpiresAt) {
		return a.token.AccessToken, nil
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	endpoint := a.authBaseURL + "/token"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBufferString(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(a.clientID+":"+a.clientSecret)))
	req.Header.Set("User-Agent", "ariadne/0.1 (+https://github.com/xmbshwll/ariadne)")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute token request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("%w %d: %s", errUnexpectedSpotifyTokenStatus, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var token tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}
	if token.AccessToken == "" {
		return "", errEmptySpotifyAccessToken
	}

	a.token = cachedToken{
		AccessToken: token.AccessToken,
		ExpiresAt:   time.Now().Add(time.Duration(token.ExpiresIn-30) * time.Second),
	}
	return a.token.AccessToken, nil
}

func (a *Adapter) hasCredentials() bool {
	return strings.TrimSpace(a.clientID) != "" && strings.TrimSpace(a.clientSecret) != ""
}

func parseInitialState(body []byte) (*initialState, error) {
	matches := initialStatePattern.FindSubmatch(body)
	if len(matches) != 2 {
		return nil, errInitialStateScriptNotFound
	}

	decoded, err := base64.StdEncoding.DecodeString(string(matches[1]))
	if err != nil {
		return nil, fmt.Errorf("decode initial state: %w", err)
	}

	var state initialState
	if err := json.Unmarshal(decoded, &state); err != nil {
		return nil, fmt.Errorf("unmarshal initial state: %w", err)
	}
	return &state, nil
}

func toCanonicalAlbumBootstrap(parsed model.ParsedAlbumURL, album spotifyAlbumEntity) *model.CanonicalAlbum {
	artists := spotifyArtistNamesBootstrap(album.Artists)
	tracks := make([]model.CanonicalTrack, 0, len(album.TracksV2.Items))
	totalDurationMS := 0
	for _, wrapped := range album.TracksV2.Items {
		trackArtists := spotifyArtistNamesBootstrap(wrapped.Track.Artists)
		durationMS := wrapped.Track.Duration.TotalMilliseconds
		totalDurationMS += durationMS
		tracks = append(tracks, model.CanonicalTrack{
			DiscNumber:      wrapped.Track.DiscNumber,
			TrackNumber:     wrapped.Track.TrackNumber,
			Title:           wrapped.Track.Name,
			NormalizedTitle: normalize.Text(wrapped.Track.Name),
			DurationMS:      durationMS,
			Artists:         trackArtists,
		})
	}

	label := spotifyLabelBootstrap(album.Copyright)
	artworkURL := spotifyArtworkURLBootstrap(album.CoverArt)
	releaseDate := spotifyReleaseDateStringBootstrap(album.Date)

	return &model.CanonicalAlbum{
		Service:           model.ServiceSpotify,
		SourceID:          album.ID,
		SourceURL:         parsed.CanonicalURL,
		RegionHint:        parsed.RegionHint,
		Title:             album.Name,
		NormalizedTitle:   normalize.Text(album.Name),
		Artists:           artists,
		NormalizedArtists: normalize.Artists(artists),
		ReleaseDate:       releaseDate,
		Label:             label,
		TrackCount:        len(tracks),
		TotalDurationMS:   totalDurationMS,
		ArtworkURL:        artworkURL,
		EditionHints:      normalize.EditionHints(album.Name),
		Tracks:            tracks,
	}
}

func toCanonicalAlbumAPI(sourceURL string, album *apiAlbumResponse) *model.CanonicalAlbum {
	artists := spotifyArtistNamesAPI(album.Artists)
	tracks := make([]model.CanonicalTrack, 0, len(album.Tracks.Items))
	totalDurationMS := 0
	explicit := false
	for _, track := range album.Tracks.Items {
		totalDurationMS += track.DurationMS
		if track.Explicit {
			explicit = true
		}
		tracks = append(tracks, model.CanonicalTrack{
			DiscNumber:      track.DiscNumber,
			TrackNumber:     track.TrackNumber,
			Title:           track.Name,
			NormalizedTitle: normalize.Text(track.Name),
			DurationMS:      track.DurationMS,
			ISRC:            track.ExternalIDs.ISRC,
			Artists:         spotifyArtistNamesAPI(track.Artists),
		})
	}

	if album.TotalTracks > 0 && len(tracks) == 0 {
		tracks = []model.CanonicalTrack{}
	}
	trackCount := album.TotalTracks
	if trackCount == 0 {
		trackCount = len(tracks)
	}

	return &model.CanonicalAlbum{
		Service:           model.ServiceSpotify,
		SourceID:          album.ID,
		SourceURL:         sourceURL,
		Title:             album.Name,
		NormalizedTitle:   normalize.Text(album.Name),
		Artists:           artists,
		NormalizedArtists: normalize.Artists(artists),
		ReleaseDate:       album.ReleaseDate,
		Label:             album.Label,
		UPC:               album.ExternalIDs.UPC,
		TrackCount:        trackCount,
		TotalDurationMS:   totalDurationMS,
		ArtworkURL:        spotifyArtworkURLAPI(album.Images),
		Explicit:          explicit,
		EditionHints:      normalize.EditionHints(album.Name),
		Tracks:            tracks,
	}
}

func toCanonicalSongAPI(sourceURL string, track *apiTrack) *model.CanonicalSong {
	artists := spotifyArtistNamesAPI(track.Artists)
	albumArtists := spotifyArtistNamesAPI(track.Album.Artists)
	albumTitle := track.Album.Name
	return &model.CanonicalSong{
		Service:                model.ServiceSpotify,
		SourceID:               track.ID,
		SourceURL:              sourceURL,
		Title:                  track.Name,
		NormalizedTitle:        normalize.Text(track.Name),
		Artists:                artists,
		NormalizedArtists:      normalize.Artists(artists),
		DurationMS:             track.DurationMS,
		ISRC:                   track.ExternalIDs.ISRC,
		Explicit:               track.Explicit,
		DiscNumber:             track.DiscNumber,
		TrackNumber:            track.TrackNumber,
		AlbumID:                track.Album.ID,
		AlbumTitle:             albumTitle,
		AlbumNormalizedTitle:   normalize.Text(albumTitle),
		AlbumArtists:           albumArtists,
		AlbumNormalizedArtists: normalize.Artists(albumArtists),
		ReleaseDate:            track.Album.ReleaseDate,
		ArtworkURL:             spotifyArtworkURLAPI(track.Album.Images),
		EditionHints:           normalize.EditionHints(track.Name),
	}
}

func spotifyArtistNamesBootstrap(list spotifyArtistList) []string {
	out := make([]string, 0, len(list.Items))
	for _, item := range list.Items {
		if item.Profile.Name == "" {
			continue
		}
		out = append(out, item.Profile.Name)
	}
	return out
}

func spotifyArtistNamesAPI(artists []apiArtist) []string {
	out := make([]string, 0, len(artists))
	for _, artist := range artists {
		if artist.Name == "" {
			continue
		}
		out = append(out, artist.Name)
	}
	return out
}

func spotifyReleaseDateStringBootstrap(date spotifyReleaseDate) string {
	if date.Year == 0 {
		return ""
	}
	if date.Month == 0 || date.Day == 0 {
		return fmt.Sprintf("%04d", date.Year)
	}
	return fmt.Sprintf("%04d-%02d-%02d", date.Year, date.Month, date.Day)
}

func spotifyLabelBootstrap(group spotifyCopyrightGroup) string {
	parts := make([]string, 0, len(group.Items))
	for _, item := range group.Items {
		text := strings.TrimSpace(item.Text)
		text = strings.TrimPrefix(text, "℗ ")
		text = strings.TrimPrefix(text, "© ")
		if text == "" {
			continue
		}
		parts = append(parts, text)
	}
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func spotifyArtworkURLBootstrap(cover spotifyCoverArt) string {
	if len(cover.Sources) == 0 {
		return ""
	}
	sorted := append([]spotifyImage(nil), cover.Sources...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Width > sorted[j].Width
	})
	return sorted[0].URL
}

func spotifyArtworkURLAPI(images []apiImage) string {
	if len(images) == 0 {
		return ""
	}
	sorted := append([]apiImage(nil), images...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Width > sorted[j].Width
	})
	return sorted[0].URL
}

func metadataQueries(album model.CanonicalAlbum) []string {
	if strings.TrimSpace(album.Title) == "" {
		return nil
	}

	queries := make([]string, 0, 8)
	seen := make(map[string]struct{}, 8)
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

	for _, title := range normalize.SearchTitleVariants(album.Title) {
		for _, artist := range normalize.SearchArtistVariants(album.Artists) {
			appendUnique(strings.Join([]string{"album:" + title, "artist:" + artist}, " "))
		}
		appendUnique("album:" + title)
	}
	return queries
}

func songMetadataQueries(song model.CanonicalSong) []string {
	if strings.TrimSpace(song.Title) == "" {
		return nil
	}

	queries := make([]string, 0, 8)
	seen := make(map[string]struct{}, 8)
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

	for _, title := range normalize.SearchTitleVariants(song.Title) {
		for _, artist := range normalize.SearchArtistVariants(song.Artists) {
			appendUnique(strings.Join([]string{"track:" + title, "artist:" + artist}, " "))
		}
		appendUnique("track:" + title)
	}
	return queries
}

func albumIDsToSummaries(ids []string) []apiAlbumSummary {
	items := make([]apiAlbumSummary, 0, len(ids))
	for _, id := range ids {
		items = append(items, apiAlbumSummary{ID: id})
	}
	return items
}

func canonicalAlbumURL(albumID string) string {
	return "https://open.spotify.com/album/" + albumID
}

func canonicalTrackURL(trackID string) string {
	return "https://open.spotify.com/track/" + trackID
}
