package ariadne

import (
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/score"
)

func toInternalServiceName(service ServiceName) model.ServiceName {
	return model.ServiceName(service)
}

func fromInternalServiceName(service model.ServiceName) ServiceName {
	return ServiceName(service)
}

func fromInternalServiceNames(services []model.ServiceName) []ServiceName {
	if services == nil {
		return nil
	}

	converted := make([]ServiceName, 0, len(services))
	for _, service := range services {
		converted = append(converted, fromInternalServiceName(service))
	}
	return converted
}

func toInternalScoreWeights(weights ScoreWeights) score.Weights {
	return score.Weights{
		UPCExact:             weights.UPCExact,
		ISRCStrongOverlap:    weights.ISRCStrongOverlap,
		ISRCPartialScale:     weights.ISRCPartialScale,
		TrackTitleStrong:     weights.TrackTitleStrong,
		TrackTitlePartial:    weights.TrackTitlePartial,
		TitleExact:           weights.TitleExact,
		CoreTitleExact:       weights.CoreTitleExact,
		PrimaryArtistExact:   weights.PrimaryArtistExact,
		ArtistOverlap:        weights.ArtistOverlap,
		TrackCountExact:      weights.TrackCountExact,
		TrackCountNear:       weights.TrackCountNear,
		TrackCountMismatch:   weights.TrackCountMismatch,
		ReleaseDateExact:     weights.ReleaseDateExact,
		ReleaseYearExact:     weights.ReleaseYearExact,
		DurationNear:         weights.DurationNear,
		LabelExact:           weights.LabelExact,
		ExplicitMismatch:     weights.ExplicitMismatch,
		EditionMismatch:      weights.EditionMismatch,
		EditionMarkerPenalty: weights.EditionMarkerPenalty,
	}
}

func fromInternalScoreWeights(weights score.Weights) ScoreWeights {
	return ScoreWeights{
		UPCExact:             weights.UPCExact,
		ISRCStrongOverlap:    weights.ISRCStrongOverlap,
		ISRCPartialScale:     weights.ISRCPartialScale,
		TrackTitleStrong:     weights.TrackTitleStrong,
		TrackTitlePartial:    weights.TrackTitlePartial,
		TitleExact:           weights.TitleExact,
		CoreTitleExact:       weights.CoreTitleExact,
		PrimaryArtistExact:   weights.PrimaryArtistExact,
		ArtistOverlap:        weights.ArtistOverlap,
		TrackCountExact:      weights.TrackCountExact,
		TrackCountNear:       weights.TrackCountNear,
		TrackCountMismatch:   weights.TrackCountMismatch,
		ReleaseDateExact:     weights.ReleaseDateExact,
		ReleaseYearExact:     weights.ReleaseYearExact,
		DurationNear:         weights.DurationNear,
		LabelExact:           weights.LabelExact,
		ExplicitMismatch:     weights.ExplicitMismatch,
		EditionMismatch:      weights.EditionMismatch,
		EditionMarkerPenalty: weights.EditionMarkerPenalty,
	}
}

func toInternalSongScoreWeights(weights SongScoreWeights) score.SongWeights {
	return score.SongWeights{
		ISRCExact:            weights.ISRCExact,
		TitleExact:           weights.TitleExact,
		CoreTitleExact:       weights.CoreTitleExact,
		PrimaryArtistExact:   weights.PrimaryArtistExact,
		ArtistOverlap:        weights.ArtistOverlap,
		DurationNear:         weights.DurationNear,
		AlbumTitleExact:      weights.AlbumTitleExact,
		ReleaseDateExact:     weights.ReleaseDateExact,
		ReleaseYearExact:     weights.ReleaseYearExact,
		TrackNumberExact:     weights.TrackNumberExact,
		ExplicitMismatch:     weights.ExplicitMismatch,
		EditionMismatch:      weights.EditionMismatch,
		EditionMarkerPenalty: weights.EditionMarkerPenalty,
	}
}

func fromInternalSongScoreWeights(weights score.SongWeights) SongScoreWeights {
	return SongScoreWeights{
		ISRCExact:            weights.ISRCExact,
		TitleExact:           weights.TitleExact,
		CoreTitleExact:       weights.CoreTitleExact,
		PrimaryArtistExact:   weights.PrimaryArtistExact,
		ArtistOverlap:        weights.ArtistOverlap,
		DurationNear:         weights.DurationNear,
		AlbumTitleExact:      weights.AlbumTitleExact,
		ReleaseDateExact:     weights.ReleaseDateExact,
		ReleaseYearExact:     weights.ReleaseYearExact,
		TrackNumberExact:     weights.TrackNumberExact,
		ExplicitMismatch:     weights.ExplicitMismatch,
		EditionMismatch:      weights.EditionMismatch,
		EditionMarkerPenalty: weights.EditionMarkerPenalty,
	}
}

func toInternalParsedURL(parsed ParsedURL) model.ParsedURL {
	return model.ParsedURL{
		Service:      toInternalServiceName(parsed.Service),
		EntityType:   parsed.EntityType,
		ID:           parsed.ID,
		CanonicalURL: parsed.CanonicalURL,
		RegionHint:   parsed.RegionHint,
		RawURL:       parsed.RawURL,
	}
}

func fromInternalParsedURL(parsed model.ParsedURL) ParsedURL {
	return ParsedURL{
		Service:      fromInternalServiceName(parsed.Service),
		EntityType:   parsed.EntityType,
		ID:           parsed.ID,
		CanonicalURL: parsed.CanonicalURL,
		RegionHint:   parsed.RegionHint,
		RawURL:       parsed.RawURL,
	}
}

func toInternalCanonicalTrack(track CanonicalTrack) model.CanonicalTrack {
	return model.CanonicalTrack{
		DiscNumber:      track.DiscNumber,
		TrackNumber:     track.TrackNumber,
		Title:           track.Title,
		NormalizedTitle: track.NormalizedTitle,
		DurationMS:      track.DurationMS,
		ISRC:            track.ISRC,
		Artists:         append([]string(nil), track.Artists...),
	}
}

func fromInternalCanonicalTrack(track model.CanonicalTrack) CanonicalTrack {
	return CanonicalTrack{
		DiscNumber:      track.DiscNumber,
		TrackNumber:     track.TrackNumber,
		Title:           track.Title,
		NormalizedTitle: track.NormalizedTitle,
		DurationMS:      track.DurationMS,
		ISRC:            track.ISRC,
		Artists:         append([]string(nil), track.Artists...),
	}
}
