package soundcloud

import (
	"strings"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/normalize"
	"github.com/xmbshwll/ariadne/internal/parse"
)

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

func toCanonicalSong(track soundTrack) *model.CanonicalSong {
	artists := nonEmptyArtistList(firstNonEmpty(track.PublisherMetadata.Artist, track.User.Username))
	durationMS := track.FullDuration
	if durationMS == 0 {
		durationMS = track.Duration
	}
	albumTitle := firstNonEmpty(track.PublisherMetadata.AlbumTitle)
	albumArtists := []string(nil)
	albumNormalizedArtists := []string(nil)
	if albumTitle != "" {
		albumArtists = artists
		albumNormalizedArtists = normalize.Artists(artists)
	}
	canonicalURL := canonicalizeSoundCloudURL(track.PermalinkURL)
	return &model.CanonicalSong{
		Service:                model.ServiceSoundCloud,
		SourceID:               soundCloudSourceID(canonicalURL),
		SourceURL:              canonicalURL,
		Title:                  track.Title,
		NormalizedTitle:        normalize.Text(track.Title),
		Artists:                artists,
		NormalizedArtists:      normalize.Artists(artists),
		DurationMS:             durationMS,
		ISRC:                   strings.TrimSpace(track.PublisherMetadata.ISRC),
		Explicit:               track.PublisherMetadata.Explicit,
		AlbumTitle:             albumTitle,
		AlbumNormalizedTitle:   normalize.Text(albumTitle),
		AlbumArtists:           albumArtists,
		AlbumNormalizedArtists: albumNormalizedArtists,
		ReleaseDate:            firstNonEmpty(dateOnly(track.ReleaseDate), dateOnly(track.DisplayDate)),
		ArtworkURL:             strings.TrimSpace(track.ArtworkURL),
		EditionHints:           normalize.EditionHints(track.Title),
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

func canonicalizeSoundCloudURL(raw string) string {
	if parsed, err := parse.SoundCloudAlbumURL(raw); err == nil {
		return parsed.CanonicalURL
	}
	if parsed, err := parse.SoundCloudSongURL(raw); err == nil {
		return parsed.CanonicalURL
	}
	return strings.TrimSpace(raw)
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

func toCandidateSong(song model.CanonicalSong) model.CandidateSong {
	return model.CandidateSong{
		CanonicalSong: song,
		CandidateID:   song.SourceID,
		MatchURL:      song.SourceURL,
	}
}

func soundCloudSourceID(canonicalURL string) string {
	if parsed, err := parse.SoundCloudAlbumURL(canonicalURL); err == nil {
		return parsed.ID
	}
	if parsed, err := parse.SoundCloudSongURL(canonicalURL); err == nil {
		return parsed.ID
	}
	return canonicalURL
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
