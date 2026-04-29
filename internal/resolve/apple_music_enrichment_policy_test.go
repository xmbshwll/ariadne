package resolve

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/score"
)

func TestAppleMusicEnrichmentPolicyCopiesIdentifiersFromStrongMatches(t *testing.T) {
	source := model.CanonicalAlbum{
		Service: model.ServiceBandcamp,
		Title:   "High Horse Heaven",
		Tracks: []model.CanonicalTrack{
			{Title: "The Edge"},
			{Title: "Cherry Coke"},
		},
	}
	spotifyCandidate := model.CandidateAlbum{
		CandidateID: "spotify-high-horse-heaven",
		CanonicalAlbum: model.CanonicalAlbum{
			Service: model.ServiceSpotify,
			UPC:     "3618021182192",
			Tracks: []model.CanonicalTrack{
				{Title: "The Edge", ISRC: "QZHN92500001"},
				{Title: "Cherry Coke", ISRC: "QZHN92500002"},
			},
		},
	}
	weakDeezerCandidate := model.CandidateAlbum{
		CandidateID: "deezer-weak",
		CanonicalAlbum: model.CanonicalAlbum{
			Service: model.ServiceDeezer,
			UPC:     "SHOULD_NOT_COPY",
		},
	}
	appleCandidate := model.CandidateAlbum{
		CandidateID: "apple-strong",
		CanonicalAlbum: model.CanonicalAlbum{
			Service: model.ServiceAppleMusic,
			UPC:     "APPLE_SHOULD_NOT_COPY",
		},
	}
	matches := map[model.ServiceName]MatchResult{
		model.ServiceSpotify: {
			Best: &ScoredMatch{Score: appleMusicCascadeMinimumScore, Candidate: spotifyCandidate},
		},
		model.ServiceDeezer: {
			Best: &ScoredMatch{Score: appleMusicCascadeMinimumScore - 1, Candidate: weakDeezerCandidate},
		},
		model.ServiceAppleMusic: {
			Best: &ScoredMatch{Score: 999, Candidate: appleCandidate},
		},
	}

	enriched, changed := newAppleMusicEnrichmentPolicy(score.DefaultWeights()).enrichedSource(source, matches)

	require.True(t, changed)
	assert.Equal(t, "3618021182192", enriched.UPC)
	assert.Equal(t, "QZHN92500001", enriched.Tracks[0].ISRC)
	assert.Equal(t, "QZHN92500002", enriched.Tracks[1].ISRC)
	assert.Empty(t, source.UPC)
	assert.Empty(t, source.Tracks[0].ISRC)
}

func TestAppleMusicEnrichmentPolicyOnlyReplacesWithBetterResult(t *testing.T) {
	policy := newAppleMusicEnrichmentPolicy(score.DefaultWeights())
	existing := MatchResult{Best: &ScoredMatch{Score: 90}}

	assert.False(t, policy.shouldReplace(existing, MatchResult{}))
	assert.False(t, policy.shouldReplace(existing, MatchResult{Best: &ScoredMatch{Score: 80}}))
	assert.True(t, policy.shouldReplace(existing, MatchResult{Best: &ScoredMatch{Score: 91}}))
	assert.True(t, policy.shouldReplace(MatchResult{}, MatchResult{Best: &ScoredMatch{Score: 1}}))
}
