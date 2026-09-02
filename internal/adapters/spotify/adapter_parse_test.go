package spotify_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xmbshwll/ariadne/internal/adapters/spotify"
	"github.com/xmbshwll/ariadne/internal/model"
)

// TestAdapterParseMethodsDelegateToPackageParses pins the Adapter method seam:
// the resolver calls the methods, the package functions own the rules, and the
// two must answer identically for every input.
func TestAdapterParseMethodsDelegateToPackageParses(t *testing.T) {
	adapter := spotify.New(nil)

	tests := []struct {
		name           string
		raw            string
		wantErrText    string
		wantService    model.ServiceName
		wantEntityType string
	}{
		{
			name:           "album url parses through the method seam",
			raw:            "https://open.spotify.com/album/1DFixLWuPkv3KT3TnV35m3",
			wantService:    model.ServiceSpotify,
			wantEntityType: model.EntityTypeAlbum,
		},
		{
			name:        "a playlist url is not an album url",
			raw:         "https://open.spotify.com/playlist/37i9dQZF1DXcBWIGoYBM5M",
			wantErrText: "not an album url",
			wantService: model.ServiceSpotify,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			methodParsed, methodErr := adapter.ParseAlbumURL(tt.raw)
			funcParsed, funcErr := spotify.ParseAlbumURL(tt.raw)

			if tt.wantErrText != "" {
				require.Error(t, methodErr, tt.name)
				assert.ErrorContains(t, methodErr, tt.wantErrText, tt.name)
			} else {
				require.NoError(t, methodErr, tt.name)
			}
			assert.Equal(t, funcParsed, methodParsed, tt.name)
			assert.Equal(t, funcErr, methodErr, tt.name)

			if methodParsed != nil {
				assert.Equal(t, tt.wantService, methodParsed.Service, tt.name)
				assert.Equal(t, tt.wantEntityType, methodParsed.EntityType, tt.name)
			}
		})
	}
}
