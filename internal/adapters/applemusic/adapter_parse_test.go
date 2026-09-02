package applemusic_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xmbshwll/ariadne/internal/adapters/applemusic"
	"github.com/xmbshwll/ariadne/internal/model"
)

// TestAdapterParseMethodsDelegateToPackageParses pins the Adapter method seam:
// the resolver calls the methods, the package functions own the rules, and the
// two must answer identically for every input.
func TestAdapterParseMethodsDelegateToPackageParses(t *testing.T) {
	adapter := applemusic.New(nil)

	tests := []struct {
		name           string
		raw            string
		wantErrText    string
		wantEntityType string
	}{
		{
			name:           "album url parses through the method seam",
			raw:            "https://music.apple.com/us/album/abbey-road/1441164426",
			wantEntityType: model.EntityTypeAlbum,
		},
		{
			name:        "a non-album url is refused with the service sentinel text",
			raw:         "https://music.apple.com/us/playlist/pl.x",
			wantErrText: "invalid apple music album path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			methodParsed, methodErr := adapter.ParseAlbumURL(tt.raw)
			funcParsed, funcErr := applemusic.ParseAlbumURL(tt.raw)

			if tt.wantErrText != "" {
				require.Error(t, methodErr, tt.name)
				assert.ErrorContains(t, methodErr, tt.wantErrText, tt.name)
			} else {
				require.NoError(t, methodErr, tt.name)
			}
			assert.Equal(t, funcParsed, methodParsed, tt.name)
			assert.Equal(t, funcErr, methodErr, tt.name)

			if methodParsed != nil {
				assert.Equal(t, model.ServiceAppleMusic, methodParsed.Service, tt.name)
				assert.Equal(t, tt.wantEntityType, methodParsed.EntityType, tt.name)
			}
		})
	}
}
