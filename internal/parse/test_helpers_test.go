package parse

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
)

func requireParseError(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
}

func requireParsedURL(t *testing.T, got *model.ParsedAlbumURL, err error, wantID string, wantURL string, wantRegion string) {
	t.Helper()

	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, wantID, got.ID)
	assert.Equal(t, wantURL, got.CanonicalURL)
	assert.Equal(t, wantRegion, got.RegionHint)
}
