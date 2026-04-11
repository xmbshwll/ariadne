package tidal

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
)

func TestAdapterRequiresCredentialsForSourceAndSearch(t *testing.T) {
	adapter := New(nil)

	_, err := adapter.FetchAlbum(context.Background(), model.ParsedAlbumURL{Service: model.ServiceTIDAL, ID: "156205493", CanonicalURL: "https://tidal.com/album/156205493"})
	require.ErrorIs(t, err, ErrCredentialsNotConfigured)
	_, err = adapter.FetchSong(context.Background(), model.ParsedURL{Service: model.ServiceTIDAL, ID: "156205494", CanonicalURL: "https://tidal.com/track/156205494"})
	require.ErrorIs(t, err, ErrCredentialsNotConfigured)
	_, err = adapter.SearchByISRC(context.Background(), []string{"QZMHK2043414"})
	require.ErrorIs(t, err, ErrCredentialsNotConfigured)
	_, err = adapter.SearchByMetadata(context.Background(), model.CanonicalAlbum{Title: "Album"})
	require.ErrorIs(t, err, ErrCredentialsNotConfigured)
	_, err = adapter.SearchSongByISRC(context.Background(), "QZMHK2043414")
	require.ErrorIs(t, err, ErrCredentialsNotConfigured)
	_, err = adapter.SearchSongByMetadata(context.Background(), model.CanonicalSong{Title: "Song"})
	require.ErrorIs(t, err, ErrCredentialsNotConfigured)
}

func TestAdapterSkipsCredentialChecksForEmptySearches(t *testing.T) {
	adapter := New(nil)

	tests := []struct {
		name string
		fn   func() (any, error)
	}{
		{
			name: "album isrc search",
			fn: func() (any, error) {
				return adapter.SearchByISRC(context.Background(), []string{"", " "})
			},
		},
		{
			name: "album metadata search",
			fn: func() (any, error) {
				return adapter.SearchByMetadata(context.Background(), model.CanonicalAlbum{})
			},
		},
		{
			name: "song isrc search",
			fn: func() (any, error) {
				return adapter.SearchSongByISRC(context.Background(), " ")
			},
		},
		{
			name: "song metadata search",
			fn: func() (any, error) {
				return adapter.SearchSongByMetadata(context.Background(), model.CanonicalSong{})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := tt.fn()
			require.NoError(t, err)
			assert.Nil(t, results)
		})
	}
}
