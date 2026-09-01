package tidal

import (
	"sort"
	"strings"

	"github.com/xmbshwll/ariadne/internal/canonical"
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/normalize"
)

func ToCanonicalAlbum(resource APIResource, included []APIResource, canonicalURL string, regionHint string) *model.CanonicalAlbum {
	resourceByID := IncludedResourceIndex(included)
	artistNames := IncludedArtistNames(resourceByID, resource.Relationships.Artists.Data)
	tracks := tracksFromIncluded(included, resource.Relationships.Items.Data, artistNames)
	artworkURL := ArtworkURLFromIncluded(resourceByID, resource.Relationships.CoverArt.Data)
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
		UPC:               canonical.FirstNonEmpty(resource.Attributes.BarcodeID, resource.Attributes.UPC),
		TrackCount:        trackCount,
		TotalDurationMS:   canonical.ISODurationMilliseconds(resource.Attributes.Duration),
		ArtworkURL:        artworkURL,
		Explicit:          resource.Attributes.Explicit,
		EditionHints:      normalize.EditionHints(resource.Attributes.Title),
		Tracks:            tracks,
	}
}

func toCanonicalSong(resource APIResource, included []APIResource, canonicalURL string, regionHint string) *model.CanonicalSong {
	resourceByID := IncludedResourceIndex(included)
	artistNames := IncludedArtistNames(resourceByID, resource.Relationships.Artists.Data)
	albumResource := FirstRelatedResource(resourceByID, resource.Relationships.Albums.Data, "albums")
	albumTitle := ""
	albumNormalizedTitle := ""
	albumArtists := []string{}
	albumNormalizedArtists := []string{}
	releaseDate := resource.Attributes.ReleaseDate
	artworkURL := ""
	if albumResource != nil {
		albumTitle = albumResource.Attributes.Title
		albumNormalizedTitle = normalize.Text(albumTitle)
		albumArtists = IncludedArtistNames(resourceByID, albumResource.Relationships.Artists.Data)
		albumNormalizedArtists = normalize.Artists(albumArtists)
		if releaseDate == "" {
			releaseDate = albumResource.Attributes.ReleaseDate
		}
		artworkURL = ArtworkURLFromIncluded(resourceByID, albumResource.Relationships.CoverArt.Data)
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
		DurationMS:             canonical.ISODurationMilliseconds(resource.Attributes.Duration),
		ISRC:                   strings.TrimSpace(resource.Attributes.ISRC),
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

func IncludedArtistNames(resourceByID map[string]APIResource, relations []RelationshipData) []string {
	results := make([]string, 0, len(relations))
	seen := make(map[string]struct{}, len(relations))
	for _, relation := range relations {
		if relation.Type != "artists" {
			continue
		}
		resource, ok := resourceByID[includedRelationKey(relation)]
		if !ok {
			continue
		}
		Name := canonical.FirstNonEmpty(resource.Attributes.Name, resource.Attributes.Title)
		if Name == "" {
			continue
		}
		if _, ok := seen[Name]; ok {
			continue
		}
		seen[Name] = struct{}{}
		results = append(results, Name)
	}
	return results
}

func FirstRelatedResource(resourceByID map[string]APIResource, relations []RelationshipData, typ string) *APIResource {
	for _, relation := range relations {
		if relation.Type != typ {
			continue
		}
		resource, ok := resourceByID[includedRelationKey(relation)]
		if !ok {
			continue
		}
		relatedResource := resource
		return &relatedResource
	}
	return nil
}

func firstRelatedID(relations []RelationshipData, typ string) string {
	for _, relation := range relations {
		if relation.Type == typ && relation.ID != "" {
			return relation.ID
		}
	}
	return ""
}

func firstTrackNumber(relations []RelationshipData) int {
	for _, relation := range relations {
		if relation.Meta.TrackNumber > 0 {
			return relation.Meta.TrackNumber
		}
	}
	return 0
}

func firstTrackVolumeNumber(relations []RelationshipData) int {
	for _, relation := range relations {
		if relation.Meta.VolumeNumber > 0 {
			return relation.Meta.VolumeNumber
		}
	}
	return 0
}

func tracksFromIncluded(included []APIResource, relations []RelationshipData, fallbackArtists []string) []model.CanonicalTrack {
	resourceByID := IncludedResourceIndex(included)
	tracks := make([]model.CanonicalTrack, 0, len(relations))
	for _, relation := range relations {
		if relation.Type != "tracks" {
			continue
		}
		resource, ok := resourceByID[includedRelationKey(relation)]
		if !ok {
			continue
		}
		trackArtists := IncludedArtistNames(resourceByID, resource.Relationships.Artists.Data)
		if len(trackArtists) == 0 {
			trackArtists = append([]string(nil), fallbackArtists...)
		}
		tracks = append(tracks, model.CanonicalTrack{
			DiscNumber:      relation.Meta.VolumeNumber,
			TrackNumber:     relation.Meta.TrackNumber,
			Title:           resource.Attributes.Title,
			NormalizedTitle: normalize.Text(resource.Attributes.Title),
			DurationMS:      canonical.ISODurationMilliseconds(resource.Attributes.Duration),
			ISRC:            strings.TrimSpace(resource.Attributes.ISRC),
			Artists:         trackArtists,
		})
	}
	sort.SliceStable(tracks, func(i, j int) bool {
		if tracks[i].DiscNumber == tracks[j].DiscNumber {
			return tracks[i].TrackNumber < tracks[j].TrackNumber
		}
		return tracks[i].DiscNumber < tracks[j].DiscNumber
	})
	return tracks
}

func ArtworkURLFromIncluded(resourceByID map[string]APIResource, relations []RelationshipData) string {
	for _, relation := range relations {
		if relation.Type != "artworks" {
			continue
		}
		resource, ok := resourceByID[includedRelationKey(relation)]
		if !ok {
			continue
		}
		files := append([]ResourceFile(nil), resource.Attributes.Files...)
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

func IncludedResourceIndex(included []APIResource) map[string]APIResource {
	resourceByID := make(map[string]APIResource, len(included))
	for _, resource := range included {
		resourceByID[includedResourceKey(resource.Type, resource.ID)] = resource
	}
	return resourceByID
}

func includedResourceKey(resourceType string, resourceID string) string {
	return resourceType + ":" + resourceID
}

func includedRelationKey(relation RelationshipData) string {
	return includedResourceKey(relation.Type, relation.ID)
}

func canonicalAlbumURL(albumID string) string {
	return "https://tidal.com/album/" + albumID
}

func canonicalTrackURL(trackID string) string {
	return "https://tidal.com/track/" + trackID
}
