package tidal_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xmbshwll/ariadne/internal/adapters/tidal"
	"github.com/xmbshwll/ariadne/internal/model"
)

// TestAdapterParseMethodsDelegateToPackageParses pins the Adapter method seam:
// the resolver calls the methods, the package functions own the rules, and the
// two must answer identically for every input.
func TestAdapterParseMethodsDelegateToPackageParses(t *testing.T) {
	adapter := tidal.New(nil)

	tests := []struct {
		name           string
		parseMethod    func(string) (*model.ParsedURL, error)
		packageParser  func(string) (*model.ParsedURL, error)
		raw            string
		wantErrText    string
		wantEntityType string
	}{
		{
			name:           "album url parses through the method seam",
			parseMethod:    adapter.ParseAlbumURL,
			packageParser:  tidal.ParseAlbumURL,
			raw:            "https://tidal.com/album/12047952",
			wantEntityType: model.EntityTypeAlbum,
		},
		{
			name:           "song url parses through the method seam",
			parseMethod:    adapter.ParseSongURL,
			packageParser:  tidal.ParseSongURL,
			raw:            "https://tidal.com/track/156205494",
			wantEntityType: model.EntityTypeSong,
		},
		{
			name:          "a non-album url is refused with the service sentinel text",
			parseMethod:   adapter.ParseAlbumURL,
			packageParser: tidal.ParseAlbumURL,
			raw:           "https://tidal.com/playlist/x",
			wantErrText:   "tidal url is not an album url",
		},
		{
			name:          "a non-song url is refused with the service sentinel text",
			parseMethod:   adapter.ParseSongURL,
			packageParser: tidal.ParseSongURL,
			raw:           "https://tidal.com/playlist/x",
			wantErrText:   "tidal url is not a song url",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			methodParsed, methodErr := tt.parseMethod(tt.raw)
			funcParsed, funcErr := tt.packageParser(tt.raw)

			if tt.wantErrText != "" {
				require.Error(t, methodErr, tt.name)
				assert.ErrorContains(t, methodErr, tt.wantErrText, tt.name)
			} else {
				require.NoError(t, methodErr, tt.name)
			}
			assert.Equal(t, funcParsed, methodParsed, tt.name)
			assert.Equal(t, funcErr, methodErr, tt.name)

			if methodParsed != nil {
				assert.Equal(t, model.ServiceTIDAL, methodParsed.Service, tt.name)
				assert.Equal(t, tt.wantEntityType, methodParsed.EntityType, tt.name)
			}
		})
	}
}
