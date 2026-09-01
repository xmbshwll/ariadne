package deezer

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/xmbshwll/ariadne/internal/adapters/canonical"
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/normalize"
)

func (a *Adapter) ToCanonicalAlbum(parsed model.ParsedAlbumURL, album AlbumResponse, tracks TracksResponse) *model.CanonicalAlbum {
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

	artworkURL := canonical.FirstNonEmpty(album.CoverXL, album.CoverBig, album.CoverMedium, album.Cover)

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

func (a *Adapter) toCanonicalSong(track trackLookupResponse) *model.CanonicalSong {
	artists := []string{}
	if track.Artist.Name != "" {
		artists = append(artists, track.Artist.Name)
	}
	artworkURL := canonical.FirstNonEmpty(track.Album.CoverXL, track.Album.CoverBig, track.Album.CoverMedium, track.Album.Cover)
	return &model.CanonicalSong{
		Service:                model.ServiceDeezer,
		SourceID:               strconv.Itoa(track.ID),
		SourceURL:              canonical.FirstNonEmpty(track.Link, canonicalTrackURL(track.ID)),
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

func contributorNames(album AlbumResponse) []string {
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

func canonicalAlbumURL(albumID int) string {
	return fmt.Sprintf("https://www.deezer.com/album/%d", albumID)
}

func canonicalAlbumURLString(albumID string) string {
	return "https://www.deezer.com/album/" + albumID
}

func canonicalTrackURL(trackID int) string {
	return fmt.Sprintf("https://www.deezer.com/track/%d", trackID)
}
