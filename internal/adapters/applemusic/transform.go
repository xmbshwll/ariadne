package applemusic

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/normalize"
	"github.com/xmbshwll/ariadne/internal/parse"
)

func toCanonicalAlbum(parsed model.ParsedAlbumURL, items []lookupItem) *model.CanonicalAlbum {
	const explicitTrack = "explicit"

	if len(items) == 0 {
		return nil
	}

	collection := items[0]
	tracks := make([]model.CanonicalTrack, 0, len(items)-1)
	totalDurationMS := 0
	trackCount := 0
	explicit := false

	for _, item := range items[1:] {
		if item.WrapperType != wrapperTypeTrack || item.Kind != entitySong {
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
		Explicit:          explicit || collection.CollectionExplicitness == explicitTrack,
		EditionHints:      normalize.EditionHints(collection.CollectionName),
		Tracks:            tracks,
	}
}

func toCanonicalSong(parsed model.ParsedURL, track lookupItem) *model.CanonicalSong {
	const explicitTrack = "explicit"

	artists := []string{track.ArtistName}
	return &model.CanonicalSong{
		Service:                model.ServiceAppleMusic,
		SourceID:               strconv.FormatInt(track.TrackID, 10),
		SourceURL:              firstNonEmpty(canonicalTrackURL(track.CollectionViewURL, track.TrackID), parsed.CanonicalURL),
		RegionHint:             parsed.RegionHint,
		Title:                  track.TrackName,
		NormalizedTitle:        normalize.Text(track.TrackName),
		Artists:                artists,
		NormalizedArtists:      normalize.Artists(artists),
		DurationMS:             track.TrackTimeMillis,
		ISRC:                   firstNonEmpty(track.TrackISRC, track.ISRC),
		Explicit:               track.TrackExplicitness == explicitTrack,
		DiscNumber:             track.DiscNumber,
		TrackNumber:            track.TrackNumber,
		AlbumID:                strconv.FormatInt(track.CollectionID, 10),
		AlbumTitle:             track.CollectionName,
		AlbumNormalizedTitle:   normalize.Text(track.CollectionName),
		AlbumArtists:           artists,
		AlbumNormalizedArtists: normalize.Artists(artists),
		ReleaseDate:            dateOnly(track.ReleaseDate),
		ArtworkURL:             preferredArtworkURL(track),
		EditionHints:           normalize.EditionHints(track.TrackName),
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

func canonicalTrackURL(collectionURL string, trackID int64) string {
	base := canonicalCollectionURL(collectionURL, "")
	if base == "" || trackID == 0 {
		return base
	}
	return base + "?i=" + strconv.FormatInt(trackID, 10)
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

func toCandidateSong(song model.CanonicalSong) model.CandidateSong {
	return model.CandidateSong{
		CanonicalSong: song,
		CandidateID:   song.SourceID,
		MatchURL:      song.SourceURL,
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
