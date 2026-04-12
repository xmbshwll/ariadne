package applemusic

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHydrateOfficialAlbumsKeepsPartialResultsWhenLaterHydrationFails(t *testing.T) {
	fixture := newTestFixture(t, buildTestPayloads(t))

	results, err := fixture.authAdapter.hydrateOfficialAlbums(context.Background(), []string{"1441164426", "missing"}, "gb")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "1441164426", results[0].CandidateID)
}

func TestHydrateSongsKeepsPartialResultsWhenLaterHydrationFails(t *testing.T) {
	fixture := newTestFixture(t, buildTestPayloads(t))

	results, err := fixture.authAdapter.hydrateSongs(context.Background(), []string{"1441164430", "missing"}, "gb")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "1441164430", results[0].CandidateID)
}
