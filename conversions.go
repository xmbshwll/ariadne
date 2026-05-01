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
	return translateOwnedSlice(services, fromInternalServiceName)
}

// Score-weight structs have identical field layouts on both sides of the
// seam. Named-type conversions keep the two layouts locked together at
// compile time: any drift fails the build here.
func toInternalScoreWeights(weights ScoreWeights) score.Weights {
	return score.Weights(weights)
}

func fromInternalScoreWeights(weights score.Weights) ScoreWeights {
	return ScoreWeights(weights)
}

func toInternalSongScoreWeights(weights SongScoreWeights) score.SongWeights {
	return score.SongWeights(weights)
}

func fromInternalSongScoreWeights(weights score.SongWeights) SongScoreWeights {
	return SongScoreWeights(weights)
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
		Artists:         copyStrings(track.Artists),
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
		Artists:         copyStrings(track.Artists),
	}
}
