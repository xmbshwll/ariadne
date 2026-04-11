package deezer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/normalize"
	"github.com/xmbshwll/ariadne/internal/parse"
)

const (
	defaultBaseURL        = "https://api.deezer.com"
	metadataSearchLimit   = 5
	identifierSearchLimit = 5
)

var (
	errUnexpectedDeezerService = errors.New("unexpected deezer service")
	errUnexpectedDeezerStatus  = errors.New("unexpected deezer status")
	errDeezerAlbumNotFound     = errors.New("deezer album not found")
	errDeezerTrackNotFound     = errors.New("deezer track not found")
)

// Adapter implements Deezer source operations.
type Adapter struct {
	baseURL string
	client  *http.Client
}

// New creates a Deezer adapter.
func New(client *http.Client) *Adapter {
	if client == nil {
		client = http.DefaultClient
	}
	return &Adapter{
		baseURL: defaultBaseURL,
		client:  client,
	}
}

// Service returns the service implemented by this adapter.
func (a *Adapter) Service() model.ServiceName {
	return model.ServiceDeezer
}

// ParseAlbumURL parses a Deezer album URL.
func (a *Adapter) ParseAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	parsed, err := parse.DeezerAlbumURL(raw)
	if err != nil {
		return nil, fmt.Errorf("parse deezer album url: %w", err)
	}
	return parsed, nil
}

// ParseSongURL parses a Deezer track URL.
func (a *Adapter) ParseSongURL(raw string) (*model.ParsedURL, error) {
	parsed, err := parse.DeezerSongURL(raw)
	if err != nil {
		return nil, fmt.Errorf("parse deezer song url: %w", err)
	}
	return parsed, nil
}

// FetchAlbum loads a Deezer album and its tracks, then converts them into the canonical model.
func (a *Adapter) FetchAlbum(ctx context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalAlbum, error) {
	if parsed.Service != model.ServiceDeezer {
		return nil, fmt.Errorf("%w: %s", errUnexpectedDeezerService, parsed.Service)
	}

	return a.fetchAlbumByID(ctx, parsed.ID)
}

// SearchByUPC resolves a Deezer album directly from a UPC when Deezer exposes the lookup path.
func (a *Adapter) SearchByUPC(ctx context.Context, upc string) ([]model.CandidateAlbum, error) {
	upc = strings.TrimSpace(upc)
	if upc == "" {
		return nil, nil
	}

	canonical, err := a.fetchAlbumByLookup(ctx, a.baseURL+"/album/upc:"+url.PathEscape(upc))
	if err != nil {
		return nil, fmt.Errorf("search deezer by upc %s: %w", upc, err)
	}

	return []model.CandidateAlbum{toCandidateAlbum(*canonical)}, nil
}

// SearchByISRC resolves Deezer albums from one or more track ISRCs.
func (a *Adapter) SearchByISRC(ctx context.Context, isrcs []string) ([]model.CandidateAlbum, error) {
	results := make([]model.CandidateAlbum, 0, len(isrcs))
	seenAlbumIDs := make(map[int]struct{}, len(isrcs))

	for _, isrc := range isrcs {
		isrc = strings.TrimSpace(isrc)
		if isrc == "" {
			continue
		}

		var track trackLookupResponse
		endpoint := a.baseURL + "/track/isrc:" + url.PathEscape(isrc)
		if err := a.getJSON(ctx, endpoint, &track); err != nil {
			return nil, fmt.Errorf("search deezer by isrc %s: %w", isrc, err)
		}
		if track.Album.ID == 0 {
			continue
		}
		albumID := track.Album.ID
		if _, ok := seenAlbumIDs[albumID]; ok {
			continue
		}
		seenAlbumIDs[albumID] = struct{}{}

		candidate, err := a.hydrateAlbumCandidate(ctx, albumID)
		if err != nil {
			return nil, fmt.Errorf("hydrate deezer album %d from isrc %s: %w", albumID, isrc, err)
		}
		results = append(results, candidate)
		if len(results) >= identifierSearchLimit {
			break
		}
	}

	return results, nil
}

// SearchByMetadata searches Deezer albums using album title and artist metadata.
func (a *Adapter) SearchByMetadata(ctx context.Context, album model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
	query := metadataQuery(album)
	if query == "" {
		return nil, nil
	}

	var searchResults albumSearchResponse
	endpoint := a.baseURL + "/search/album?q=" + url.QueryEscape(query)
	if err := a.getJSON(ctx, endpoint, &searchResults); err != nil {
		return nil, fmt.Errorf("search deezer by metadata %q: %w", query, err)
	}

	results := make([]model.CandidateAlbum, 0, min(len(searchResults.Data), metadataSearchLimit))
	for _, candidate := range searchResults.Data {
		hydrated, err := a.hydrateAlbumCandidate(ctx, candidate.ID)
		if err != nil {
			return nil, fmt.Errorf("hydrate deezer candidate %d: %w", candidate.ID, err)
		}
		results = append(results, hydrated)
		if len(results) >= metadataSearchLimit {
			break
		}
	}

	return results, nil
}

// FetchSong loads a Deezer track and converts it into the canonical song model.
func (a *Adapter) FetchSong(ctx context.Context, parsed model.ParsedURL) (*model.CanonicalSong, error) {
	if parsed.Service != model.ServiceDeezer {
		return nil, fmt.Errorf("%w: %s", errUnexpectedDeezerService, parsed.Service)
	}
	return a.fetchSongByID(ctx, parsed.ID)
}

// SearchSongByISRC resolves Deezer songs from an ISRC.
func (a *Adapter) SearchSongByISRC(ctx context.Context, isrc string) ([]model.CandidateSong, error) {
	isrc = strings.TrimSpace(isrc)
	if isrc == "" {
		return nil, nil
	}

	track, err := a.fetchTrackLookup(ctx, a.baseURL+"/track/isrc:"+url.PathEscape(isrc))
	if err != nil {
		if errors.Is(err, errDeezerTrackNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("search deezer song by isrc %s: %w", isrc, err)
	}
	return []model.CandidateSong{toCandidateSong(*a.toCanonicalSong(*track))}, nil
}

// SearchSongByMetadata searches Deezer tracks using song title and artist metadata.
func (a *Adapter) SearchSongByMetadata(ctx context.Context, song model.CanonicalSong) ([]model.CandidateSong, error) {
	query := songMetadataQuery(song)
	if query == "" {
		return nil, nil
	}

	var searchResults tracksResponse
	endpoint := a.baseURL + "/search/track?q=" + url.QueryEscape(query)
	if err := a.getJSON(ctx, endpoint, &searchResults); err != nil {
		return nil, fmt.Errorf("search deezer song by metadata %q: %w", query, err)
	}

	results := make([]model.CandidateSong, 0, min(len(searchResults.Data), metadataSearchLimit))
	for _, candidate := range searchResults.Data {
		hydrated, err := a.hydrateSongCandidate(ctx, candidate.ID)
		if err != nil {
			return nil, fmt.Errorf("hydrate deezer song candidate %d: %w", candidate.ID, err)
		}
		results = append(results, hydrated)
		if len(results) >= metadataSearchLimit {
			break
		}
	}

	return results, nil
}

func (a *Adapter) hydrateAlbumCandidate(ctx context.Context, albumID int) (model.CandidateAlbum, error) {
	canonical, err := a.fetchAlbumByID(ctx, strconv.Itoa(albumID))
	if err != nil {
		return model.CandidateAlbum{}, err
	}
	return toCandidateAlbum(*canonical), nil
}

func (a *Adapter) hydrateSongCandidate(ctx context.Context, trackID int) (model.CandidateSong, error) {
	canonical, err := a.fetchSongByID(ctx, strconv.Itoa(trackID))
	if err != nil {
		return model.CandidateSong{}, err
	}
	return toCandidateSong(*canonical), nil
}

func (a *Adapter) fetchAlbumByID(ctx context.Context, albumID string) (*model.CanonicalAlbum, error) {
	parsed := model.ParsedAlbumURL{
		Service:      model.ServiceDeezer,
		EntityType:   "album",
		ID:           albumID,
		CanonicalURL: canonicalAlbumURLString(albumID),
		RawURL:       canonicalAlbumURLString(albumID),
	}
	return a.fetchAlbumByLookup(ctx, a.baseURL+"/album/"+albumID, parsed)
}

func (a *Adapter) fetchSongByID(ctx context.Context, trackID string) (*model.CanonicalSong, error) {
	track, err := a.fetchTrackLookup(ctx, a.baseURL+"/track/"+trackID)
	if err != nil {
		return nil, fmt.Errorf("fetch deezer song %s: %w", trackID, err)
	}
	canonical := a.toCanonicalSong(*track)
	return canonical, nil
}

func (a *Adapter) fetchTrackLookup(ctx context.Context, endpoint string) (*trackLookupResponse, error) {
	var track trackLookupResponse
	if err := a.getJSON(ctx, endpoint, &track); err != nil {
		return nil, err
	}
	if track.ID == 0 {
		return nil, errDeezerTrackNotFound
	}
	return &track, nil
}

func (a *Adapter) fetchAlbumByLookup(ctx context.Context, endpoint string, parsedOverride ...model.ParsedAlbumURL) (*model.CanonicalAlbum, error) {
	var album albumResponse
	if err := a.getJSON(ctx, endpoint, &album); err != nil {
		return nil, err
	}

	if album.ID == 0 || album.TracklistURL == "" {
		return nil, errDeezerAlbumNotFound
	}

	var tracks tracksResponse
	if err := a.getJSON(ctx, album.TracklistURL, &tracks); err != nil {
		return nil, fmt.Errorf("fetch deezer album tracks %d: %w", album.ID, err)
	}

	parsed := model.ParsedAlbumURL{
		Service:      model.ServiceDeezer,
		EntityType:   "album",
		ID:           strconv.Itoa(album.ID),
		CanonicalURL: canonicalAlbumURL(album.ID),
		RawURL:       canonicalAlbumURL(album.ID),
	}
	if len(parsedOverride) > 0 {
		parsed = parsedOverride[0]
	}

	return a.toCanonicalAlbum(parsed, album, tracks), nil
}

func (a *Adapter) getJSON(ctx context.Context, endpoint string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		closeErr := resp.Body.Close()
		if closeErr != nil {
			return fmt.Errorf("unexpected status %d and close body: %w", resp.StatusCode, closeErr)
		}
		return fmt.Errorf("%w: %d", errUnexpectedDeezerStatus, resp.StatusCode)
	}

	decodeErr := json.NewDecoder(resp.Body).Decode(target)
	closeErr := resp.Body.Close()
	if decodeErr != nil {
		if closeErr != nil {
			return errors.Join(
				fmt.Errorf("decode response: %w", decodeErr),
				fmt.Errorf("close response body: %w", closeErr),
			)
		}
		return fmt.Errorf("decode response: %w", decodeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close response body: %w", closeErr)
	}
	return nil
}

func (a *Adapter) toCanonicalAlbum(parsed model.ParsedAlbumURL, album albumResponse, tracks tracksResponse) *model.CanonicalAlbum {
	artists := contributorNames(album)
	if len(artists) == 0 && album.Artist.Name != "" {
		artists = []string{album.Artist.Name}
	}

	canonicalTracks := make([]model.CanonicalTrack, 0, len(tracks.Data))
	for _, track := range tracks.Data {
		trackArtists := []string{}
		if track.Artist.Name != "" {
			trackArtists = append(trackArtists, track.Artist.Name)
		}
		canonicalTracks = append(canonicalTracks, model.CanonicalTrack{
			DiscNumber:      track.DiskNumber,
			TrackNumber:     track.TrackPosition,
			Title:           track.Title,
			NormalizedTitle: normalize.Text(track.Title),
			DurationMS:      track.Duration * 1000,
			ISRC:            track.ISRC,
			Artists:         trackArtists,
		})
	}

	trackCount := album.NBTracks
	if trackCount == 0 {
		trackCount = len(canonicalTracks)
	}

	artworkURL := firstNonEmpty(album.CoverXL, album.CoverBig, album.CoverMedium, album.Cover)

	return &model.CanonicalAlbum{
		Service:           model.ServiceDeezer,
		SourceID:          strconv.Itoa(album.ID),
		SourceURL:         parsed.CanonicalURL,
		RegionHint:        parsed.RegionHint,
		Title:             album.Title,
		NormalizedTitle:   normalize.Text(album.Title),
		Artists:           artists,
		NormalizedArtists: normalize.Artists(artists),
		ReleaseDate:       album.ReleaseDate,
		Label:             album.Label,
		UPC:               album.UPC,
		TrackCount:        trackCount,
		TotalDurationMS:   album.Duration * 1000,
		ArtworkURL:        artworkURL,
		Explicit:          album.ExplicitLyrics,
		EditionHints:      normalize.EditionHints(album.Title),
		Tracks:            canonicalTracks,
	}
}

func contributorNames(album albumResponse) []string {
	artists := make([]string, 0, len(album.Contributors))
	for _, contributor := range album.Contributors {
		if contributor.Name == "" {
			continue
		}
		if slices.Contains(artists, contributor.Name) {
			continue
		}
		artists = append(artists, contributor.Name)
	}
	return artists
}

func toCandidateAlbum(album model.CanonicalAlbum) model.CandidateAlbum {
	return model.CandidateAlbum{
		CanonicalAlbum: album,
		CandidateID:    album.SourceID,
		MatchURL:       album.SourceURL,
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

func canonicalAlbumURL(albumID int) string {
	return fmt.Sprintf("https://www.deezer.com/album/%d", albumID)
}

func canonicalAlbumURLString(albumID string) string {
	return "https://www.deezer.com/album/" + albumID
}

func canonicalTrackURL(trackID int) string {
	return fmt.Sprintf("https://www.deezer.com/track/%d", trackID)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func (a *Adapter) toCanonicalSong(track trackLookupResponse) *model.CanonicalSong {
	artists := []string{}
	if track.Artist.Name != "" {
		artists = append(artists, track.Artist.Name)
	}
	artworkURL := firstNonEmpty(track.Album.CoverXL, track.Album.CoverBig, track.Album.CoverMedium, track.Album.Cover)
	return &model.CanonicalSong{
		Service:                model.ServiceDeezer,
		SourceID:               strconv.Itoa(track.ID),
		SourceURL:              firstNonEmpty(track.Link, canonicalTrackURL(track.ID)),
		Title:                  track.Title,
		NormalizedTitle:        normalize.Text(track.Title),
		Artists:                artists,
		NormalizedArtists:      normalize.Artists(artists),
		DurationMS:             track.Duration * 1000,
		ISRC:                   track.ISRC,
		Explicit:               track.ExplicitLyrics,
		DiscNumber:             track.DiskNumber,
		TrackNumber:            track.TrackPosition,
		AlbumID:                strconv.Itoa(track.Album.ID),
		AlbumTitle:             track.Album.Title,
		AlbumNormalizedTitle:   normalize.Text(track.Album.Title),
		AlbumArtists:           append([]string(nil), artists...),
		AlbumNormalizedArtists: normalize.Artists(artists),
		ReleaseDate:            track.Album.ReleaseDate,
		ArtworkURL:             artworkURL,
		EditionHints:           normalize.EditionHints(track.Title),
	}
}

func toCandidateSong(song model.CanonicalSong) model.CandidateSong {
	return model.CandidateSong{
		CanonicalSong: song,
		CandidateID:   song.SourceID,
		MatchURL:      song.SourceURL,
	}
}
