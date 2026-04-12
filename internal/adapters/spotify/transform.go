package spotify

import (
	"fmt"
	"sort"
	"strings"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/normalize"
)

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
	return buildMetadataQueries("album", album.Title, album.Artists)
}

func songMetadataQueries(song model.CanonicalSong) []string {
	return buildMetadataQueries("track", song.Title, song.Artists)
}

func buildMetadataQueries(prefix string, title string, artists []string) []string {
	if strings.TrimSpace(title) == "" {
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

	titleVariants := normalize.SearchTitleVariants(title)
	artistVariants := normalize.SearchArtistVariants(artists)
	for _, titleVariant := range titleVariants {
		for _, artistVariant := range artistVariants {
			appendUnique(strings.Join([]string{prefix + ":" + titleVariant, "artist:" + artistVariant}, " "))
		}
	}
	for _, titleVariant := range titleVariants {
		appendUnique(prefix + ":" + titleVariant)
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
