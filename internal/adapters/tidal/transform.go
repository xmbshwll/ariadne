package tidal

import (
	"sort"
	"strings"
	"time"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/normalize"
)

func toCanonicalAlbum(resource apiResource, included []apiResource, canonicalURL string, regionHint string) *model.CanonicalAlbum {
	resourceByID := includedResourceIndex(included)
	artistNames := includedArtistNames(resourceByID, resource.Relationships.Artists.Data)
	tracks := tracksFromIncluded(included, resource.Relationships.Items.Data, artistNames)
	artworkURL := artworkURLFromIncluded(resourceByID, resource.Relationships.CoverArt.Data)
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
	resourceByID := includedResourceIndex(included)
	artistNames := includedArtistNames(resourceByID, resource.Relationships.Artists.Data)
	albumResource := firstRelatedResource(resourceByID, resource.Relationships.Albums.Data, "albums")
	albumTitle := ""
	albumNormalizedTitle := ""
	albumArtists := []string{}
	albumNormalizedArtists := []string{}
	releaseDate := resource.Attributes.ReleaseDate
	artworkURL := ""
	if albumResource != nil {
		albumTitle = albumResource.Attributes.Title
		albumNormalizedTitle = normalize.Text(albumTitle)
		albumArtists = includedArtistNames(resourceByID, albumResource.Relationships.Artists.Data)
		albumNormalizedArtists = normalize.Artists(albumArtists)
		if releaseDate == "" {
			releaseDate = albumResource.Attributes.ReleaseDate
		}
		artworkURL = artworkURLFromIncluded(resourceByID, albumResource.Relationships.CoverArt.Data)
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

func includedArtistNames(resourceByID map[string]apiResource, relations []relationshipData) []string {
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

func firstRelatedResource(resourceByID map[string]apiResource, relations []relationshipData, typ string) *apiResource {
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
	resourceByID := includedResourceIndex(included)
	tracks := make([]model.CanonicalTrack, 0, len(relations))
	for _, relation := range relations {
		if relation.Type != "tracks" {
			continue
		}
		resource, ok := resourceByID[includedRelationKey(relation)]
		if !ok {
			continue
		}
		trackArtists := includedArtistNames(resourceByID, resource.Relationships.Artists.Data)
		if len(trackArtists) == 0 {
			trackArtists = append([]string(nil), fallbackArtists...)
		}
		tracks = append(tracks, model.CanonicalTrack{
			DiscNumber:      relation.Meta.VolumeNumber,
			TrackNumber:     relation.Meta.TrackNumber,
			Title:           resource.Attributes.Title,
			NormalizedTitle: normalize.Text(resource.Attributes.Title),
			DurationMS:      parseISODurationMilliseconds(resource.Attributes.Duration),
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

func artworkURLFromIncluded(resourceByID map[string]apiResource, relations []relationshipData) string {
	for _, relation := range relations {
		if relation.Type != "artworks" {
			continue
		}
		resource, ok := resourceByID[includedRelationKey(relation)]
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

func includedResourceIndex(included []apiResource) map[string]apiResource {
	resourceByID := make(map[string]apiResource, len(included))
	for _, resource := range included {
		resourceByID[includedResourceKey(resource.Type, resource.ID)] = resource
	}
	return resourceByID
}

func includedResourceKey(resourceType string, resourceID string) string {
	return resourceType + ":" + resourceID
}

func includedRelationKey(relation relationshipData) string {
	return includedResourceKey(relation.Type, relation.ID)
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
