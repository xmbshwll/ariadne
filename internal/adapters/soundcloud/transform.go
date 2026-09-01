package soundcloud

import (
	"strings"

	"github.com/xmbshwll/ariadne/internal/canonical"
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/normalize"
)

func ToCanonicalAlbum(playlist soundPlaylist) *model.CanonicalAlbum {
	artists := canonical.SingleArtistList(canonical.FirstNonEmpty(playlist.User.Username, trackArtist(playlist.Tracks)))
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
		artistNames := canonical.SingleArtistList(canonical.FirstNonEmpty(track.PublisherMetadata.Artist, track.User.Username, playlist.User.Username))
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
	label := canonical.FirstNonEmpty(playlist.LabelName, trackLabel(playlist.Tracks), trackPLine(playlist.Tracks))
	canonicalURL := canonicalizeSoundCloudURL(playlist.PermalinkURL)
	sourceID := soundCloudSourceID(canonicalURL)
	releaseDate := canonical.FirstNonEmpty(canonical.DateOnly(playlist.ReleaseDate), canonical.DateOnly(playlist.PublishedAt), canonical.DateOnly(playlist.DisplayDate))
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
		ArtworkURL:        strings.TrimSpace(playlist.ArtworkURL),
		Explicit:          explicit,
		EditionHints:      normalize.EditionHints(playlist.Title),
		Tracks:            tracks,
	}
}

func ToCanonicalSong(track SoundTrack) *model.CanonicalSong {
	artists := canonical.SingleArtistList(canonical.FirstNonEmpty(track.PublisherMetadata.Artist, track.User.Username))
	durationMS := track.FullDuration
	if durationMS == 0 {
		durationMS = track.Duration
	}
	albumTitle := canonical.FirstNonEmpty(track.PublisherMetadata.AlbumTitle)
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
		ReleaseDate:            canonical.FirstNonEmpty(canonical.DateOnly(track.ReleaseDate), canonical.DateOnly(track.DisplayDate)),
		ArtworkURL:             strings.TrimSpace(track.ArtworkURL),
		EditionHints:           normalize.EditionHints(track.Title),
	}
}

func metadataQuery(album model.CanonicalAlbum) string {
	return normalize.SearchPrimaryQuery(album.Title, album.Artists)
}

func songMetadataQuery(song model.CanonicalSong) string {
	return normalize.SearchPrimaryQuery(song.Title, song.Artists)
}

func canonicalizeSoundCloudURL(raw string) string {
	if parsed, err := ParseAlbumURL(raw); err == nil {
		return parsed.CanonicalURL
	}
	if parsed, err := ParseSongURL(raw); err == nil {
		return parsed.CanonicalURL
	}
	return strings.TrimSpace(raw)
}

func consistentUPC(tracks []SoundTrack) string {
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

func trackArtist(tracks []SoundTrack) string {
	for _, track := range tracks {
		if artist := canonical.FirstNonEmpty(track.PublisherMetadata.Artist, track.User.Username); artist != "" {
			return artist
		}
	}
	return ""
}

func trackLabel(tracks []SoundTrack) string {
	for _, track := range tracks {
		if label := canonical.FirstNonEmpty(track.LabelName); label != "" {
			return label
		}
	}
	return ""
}

func trackPLine(tracks []SoundTrack) string {
	for _, track := range tracks {
		if pLine := canonical.FirstNonEmpty(track.PublisherMetadata.PLineForDisplay, track.PublisherMetadata.CLineForDisplay); pLine != "" {
			return pLine
		}
	}
	return ""
}

func extractClientID(body []byte) string {
	if matches := clientIDPattern.FindSubmatch(body); len(matches) == 2 {
		return string(matches[1])
	}
	return ""
}

func soundCloudSourceID(canonicalURL string) string {
	if parsed, err := ParseAlbumURL(canonicalURL); err == nil {
		return parsed.ID
	}
	if parsed, err := ParseSongURL(canonicalURL); err == nil {
		return parsed.ID
	}
	return canonicalURL
}
