package resolve

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xmbshwll/ariadne/internal/model"
)

// metadataOnlyAlbumTarget implements the metadata Capability and nothing else.
type metadataOnlyAlbumTarget struct{ calls *[]string }

func (t metadataOnlyAlbumTarget) Service() model.ServiceName { return model.ServiceDeezer }

func (t metadataOnlyAlbumTarget) SearchByMetadata(_ context.Context, _ model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
	*t.calls = append(*t.calls, "metadata")
	return []model.CandidateAlbum{{CandidateID: "from-metadata"}}, nil
}

// fullAlbumTarget implements every album Capability.
type fullAlbumTarget struct{ calls *[]string }

func (t fullAlbumTarget) Service() model.ServiceName { return model.ServiceSpotify }

func (t fullAlbumTarget) SearchByUPC(_ context.Context, _ string) ([]model.CandidateAlbum, error) {
	*t.calls = append(*t.calls, "upc")
	return []model.CandidateAlbum{{CandidateID: "from-upc"}}, nil
}

func (t fullAlbumTarget) SearchByISRC(_ context.Context, _ []string) ([]model.CandidateAlbum, error) {
	*t.calls = append(*t.calls, "isrc")
	return []model.CandidateAlbum{{CandidateID: "from-isrc"}}, nil
}

func (t fullAlbumTarget) SearchByMetadata(_ context.Context, _ model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
	*t.calls = append(*t.calls, "metadata")
	return []model.CandidateAlbum{{CandidateID: "from-metadata"}}, nil
}

func TestAlbumTargetSearchPlanProbesCapabilities(t *testing.T) {
	var calls []string
	candidates, err := albumTargetSearchPlan(metadataOnlyAlbumTarget{&calls},
		model.CanonicalAlbum{Title: "Abbey Road", UPC: "602547670342"}, nil).Collect(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []string{"metadata"}, calls, "layers the adapter cannot serve must not be searched")
	assert.Equal(t, "from-metadata", candidates[0].CandidateID)
}

func TestAlbumTargetSearchPlanOrderAndIdentifierGates(t *testing.T) {
	var calls []string
	plan := albumTargetSearchPlan(fullAlbumTarget{&calls}, model.CanonicalAlbum{
		Title:  "Abbey Road",
		UPC:    "602547670342",
		Tracks: []model.CanonicalTrack{{ISRC: "GBAYE0601690"}},
	}, nil)
	require.Len(t, plan.Layers, 3)
	assert.Equal(t, []string{"SearchByUPC", "SearchByISRC", "SearchByMetadata"},
		[]string{plan.Layers[0].Name, plan.Layers[1].Name, plan.Layers[2].Name})
	assert.True(t, plan.Layers[0].Enabled)
	assert.True(t, plan.Layers[1].Enabled)
	assert.True(t, plan.Layers[2].Enabled)

	// Without a UPC or any track ISRC only the metadata layer can run.
	identiferless := albumTargetSearchPlan(fullAlbumTarget{&calls}, model.CanonicalAlbum{Title: "Abbey Road"}, nil)
	assert.False(t, identiferless.Layers[0].Enabled, "UPC layer needs source.UPC")
	assert.False(t, identiferless.Layers[1].Enabled, "ISRC layer needs track ISRCs")
	assert.True(t, identiferless.Layers[2].Enabled)

	candidates, err := identiferless.Collect(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "from-metadata", candidates[0].CandidateID)
}

// fullSongTarget implements the whole song Target Search Capability set.
type fullSongTarget struct{}

func (fullSongTarget) Service() model.ServiceName { return model.ServiceDeezer }

func (fullSongTarget) SearchSongByISRC(_ context.Context, _ string) ([]model.CandidateSong, error) {
	return []model.CandidateSong{{CandidateID: "from-isrc"}}, nil
}

func (fullSongTarget) SearchSongByMetadata(_ context.Context, _ model.CanonicalSong) ([]model.CandidateSong, error) {
	return []model.CandidateSong{{CandidateID: "from-metadata"}}, nil
}

func TestSongTargetSearchPlanOrderAndIdentifierGates(t *testing.T) {
	plan := songTargetSearchPlan(fullSongTarget{}, model.CanonicalSong{Title: "Let It Be"})
	require.Len(t, plan.Layers, 2)
	assert.Equal(t, []string{"SearchSongByISRC", "SearchSongByMetadata"},
		[]string{plan.Layers[0].Name, plan.Layers[1].Name})
	assert.False(t, plan.Layers[0].Enabled, "ISRC layer needs source.ISRC")
	assert.True(t, plan.Layers[1].Enabled)

	withISRC := songTargetSearchPlan(fullSongTarget{}, model.CanonicalSong{Title: "Let It Be", ISRC: "GBAYE0601690"})
	assert.True(t, withISRC.Layers[0].Enabled)
}

func TestStrongIntermediateAlbumMatchesMergeInDeterministicOrder(t *testing.T) {
	match := func(service model.ServiceName, candidateID, upc string) MatchResult {
		return MatchResult{Service: service, Best: &ScoredMatch{
			Score: AppleMusicCascadeMinimumScore,
			Candidate: model.CandidateAlbum{
				CanonicalAlbum: model.CanonicalAlbum{Service: service, UPC: upc},
				CandidateID:    candidateID,
			},
		}}
	}
	matches := map[model.ServiceName]MatchResult{
		model.ServiceTIDAL:   match(model.ServiceTIDAL, "t-1", "upc-from-tidal"),
		model.ServiceSpotify: match(model.ServiceSpotify, "s-9", "upc-from-spotify"),
	}

	// Equal scores over a map: without the stable sort the merged UPC would
	// change between runs.
	for range 32 {
		strong := strongIntermediateAlbumMatches(matches)
		require.Len(t, strong, 2)
		require.Equal(t, model.ServiceSpotify, strong[0].Candidate.Service)
		require.Equal(t, model.ServiceTIDAL, strong[1].Candidate.Service)

		enriched := CloneAlbum(model.CanonicalAlbum{Service: model.ServiceDeezer})
		for _, candidate := range strong {
			mergeAlbumIdentifiers(&enriched, candidate.Candidate)
		}
		assert.Equal(t, "upc-from-spotify", enriched.UPC)
	}
}
