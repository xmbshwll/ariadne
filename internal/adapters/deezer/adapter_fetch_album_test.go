package deezer_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xmbshwll/ariadne/internal/model"
)

// TestFetchAlbumCoversTheSourceSeam exercises the FetchAlbum wrapper the
// resolver calls: the album fixture is served for both the album id route and
// the UPC route, so the adapter's own parse-fetch path runs end to end.
func TestFetchAlbumCoversTheSourceSeam(t *testing.T) {
	tests := []struct {
		name          string
		parsedService model.ServiceName
		albumID       string
		wantTitle     string
		wantErrText   string
	}{
		{
			name:          "fetches the canonical album for a parsed deezer url",
			parsedService: model.ServiceDeezer,
			albumID:       "12047952",
			wantTitle:     "Abbey Road (Remastered)",
		},
		{
			name:          "refuses a url parsed for another service",
			parsedService: model.ServiceSpotify,
			albumID:       "12047952",
			wantErrText:   "unexpected deezer service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			albumBytes, trackBytes := mustReadDeezerAlbumFixtures(t)
			server := newTestServer(t, albumBytes, trackBytes, mustReadDeezerAlbumSearchFixture(t))
			adapter := newTestAdapter(server)

			album, err := adapter.FetchAlbum(context.Background(), model.ParsedAlbumURL{
				Service:      tt.parsedService,
				EntityType:   model.EntityTypeAlbum,
				ID:           tt.albumID,
				CanonicalURL: "https://www.deezer.com/album/" + tt.albumID,
			})

			if tt.wantErrText != "" {
				require.ErrorContains(t, err, tt.wantErrText, tt.name)
				assert.Nil(t, album)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, album)
			assert.Equal(t, tt.wantTitle, album.Title, tt.name)
		})
	}
}
